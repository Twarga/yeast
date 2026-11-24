package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	liveMemFileName  = "memory.bin"
	liveOverlayName  = "overlay.qcow2"
	livePlanFileName = "plan.json"
	liveBaseName     = "base.qcow2"

	// livePollInterval polls QMP while a migration job runs.
	livePollInterval = 25 * time.Millisecond
	// liveMigrateTimeout bounds the RAM save/restore wait.
	liveMigrateTimeout = 5 * time.Minute
	// headroomMiB is the free-space margin required beyond guest RAM.
	liveSpaceHeadroomMiB = 512
)

// statfsFreeFn returns free bytes at a path; injectable for tests.
var statfsFreeFn = func(dir string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Bsize), nil
}

// SaveLiveSnapshot freezes a running guest's RAM + disk into SnapshotDir:
//
//	stop → blockdev-snapshot-sync (fresh overlay becomes active) →
//	move frozen layer to <snap>/base.qcow2 → change-backing-file →
//	migrate RAM to <snap>/memory.bin → cont
//
// The guest pauses only between `stop` and `cont` (reported as PauseMs).
func (r *Runtime) SaveLiveSnapshot(ctx context.Context, plan rtm.LiveSnapshotPlan) (rtm.LiveSnapshotInfo, error) {
	if plan.Instance.RuntimeDir == "" || plan.SnapshotDir == "" {
		return rtm.LiveSnapshotInfo{}, fmt.Errorf("runtime dir and snapshot dir are required")
	}
	socketPath := qmpSocketPath(plan.Instance.RuntimeDir)
	if _, err := os.Stat(socketPath); err != nil {
		return rtm.LiveSnapshotInfo{}, fmt.Errorf("qmp socket not available — is %s running? %w", plan.Instance.Name, err)
	}
	if err := os.MkdirAll(plan.SnapshotDir, 0755); err != nil {
		return rtm.LiveSnapshotInfo{}, err
	}

	// Disk space: guest RAM + headroom must be free under the snapshot dir.
	if free, err := statfsFreeFn(plan.SnapshotDir); err == nil {
		need := uint64(plan.MemoryMiB+liveSpaceHeadroomMiB) * 1024 * 1024
		if plan.MemoryMiB > 0 && free < need {
			return rtm.LiveSnapshotInfo{}, fmt.Errorf("insufficient disk space for live snapshot: need ~%d MiB free (RAM %d MiB + headroom), have %d MiB",
				(plan.MemoryMiB + liveSpaceHeadroomMiB), plan.MemoryMiB, free/(1024*1024))
		}
	}

	client, err := newQMPClient(socketPath, 5*time.Second)
	if err != nil {
		return rtm.LiveSnapshotInfo{}, fmt.Errorf("connect to qmp: %w", err)
	}
	defer client.close()

	// Best-effort: enable fast migration paths the host advertises.
	method := enableMigrationCapabilities(client)

	diskPath := filepath.Join(plan.Instance.RuntimeDir, activeDiskName)
	if plan.Instance.ActiveDisk != "" {
		diskPath = plan.Instance.ActiveDisk
	}
	snapOverlay := filepath.Join(plan.SnapshotDir, liveOverlayName)
	snapBase := filepath.Join(plan.SnapshotDir, liveBaseName)
	memFile := filepath.Join(plan.SnapshotDir, liveMemFileName)

	pauseStart := time.Now()
	stopped := false
	resume := func() {
		if stopped {
			_ = client.execute("cont", nil)
		}
	}

	if err := client.execute("stop", nil); err != nil {
		return rtm.LiveSnapshotInfo{}, fmt.Errorf("pause guest: %w", err)
	}
	stopped = true

	fail := func(err error) (rtm.LiveSnapshotInfo, error) {
		resume()
		return rtm.LiveSnapshotInfo{}, err
	}

	// Freeze the disk layer: QEMU pivots the active disk onto a new overlay.
	device, err := findDiskDevice(client, diskPath)
	if err != nil {
		return fail(err)
	}
	if err := client.execute("blockdev-snapshot-sync", map[string]any{
		"device":        device,
		"snapshot-file": snapOverlay,
		"format":        "qcow2",
	}); err != nil {
		return fail(fmt.Errorf("blockdev-snapshot-sync: %w", err))
	}

	// The pre-pivot layer is frozen and QEMU no longer holds it — move it
	// into the snapshot dir, then re-point the overlay's recorded backing.
	movedBase := false
	if filepath.Clean(diskPath) != filepath.Clean(snapBase) {
		if err := os.Rename(diskPath, snapBase); err != nil {
			return fail(fmt.Errorf("move frozen disk into snapshot dir: %w", err))
		}
		movedBase = true
	} else {
		// Active disk already lived inside the snapshot dir (a second live
		// snapshot): keep a distinct base name instead.
		snapBase = filepath.Join(plan.SnapshotDir, "base-"+filepath.Base(diskPath))
		if err := os.Rename(diskPath, snapBase); err != nil {
			return fail(fmt.Errorf("move frozen disk into snapshot dir: %w", err))
		}
		movedBase = true
	}
	rollbackBase := func() {
		if movedBase {
			_ = os.Chmod(snapBase, 0644)
			_ = os.Rename(snapBase, diskPath)
		}
	}
	if err := os.Chmod(snapBase, 0444); err != nil {
		rollbackBase()
		return fail(fmt.Errorf("protect snapshot base: %w", err))
	}
	if node, err := findOverlayNode(client, device, snapOverlay); err == nil && node != "" {
		if err := client.execute("change-backing-file", map[string]any{
			"device":          device,
			"image-node-name": node,
			"backing-file":    snapBase,
		}); err != nil {
			rollbackBase()
			return fail(fmt.Errorf("change-backing-file: %w", err))
		}
	}

	// RAM → file.
	if err := client.execute("migrate", map[string]any{"uri": "file:" + memFile}); err != nil {
		rollbackBase()
		return fail(fmt.Errorf("start ram migration: %w", err))
	}
	if err := waitMigration(ctx, client, liveMigrateTimeout); err != nil {
		rollbackBase()
		return fail(fmt.Errorf("ram migration: %w", err))
	}

	resume()
	pause := time.Since(pauseStart).Milliseconds()

	// Persist the machine plan for identical-topology restore.
	if data, err := json.MarshalIndent(plan.Plan, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(plan.SnapshotDir, livePlanFileName), data, 0644)
	}

	return rtm.LiveSnapshotInfo{Method: method, PauseMs: pause}, nil
}

// RestoreLiveSnapshot respawns an instance from a saved RAM+disk checkpoint:
// fresh overlay on the frozen base, QEMU with -incoming defer, then
// migrate-incoming of the saved memory image, then cont.
func (r *Runtime) RestoreLiveSnapshot(ctx context.Context, plan rtm.LiveRestorePlan) (rtm.RuntimeInstance, error) {
	if plan.Instance.RuntimeDir == "" || plan.SnapshotDir == "" {
		return rtm.RuntimeInstance{}, fmt.Errorf("runtime dir and snapshot dir are required")
	}
	base := filepath.Join(plan.SnapshotDir, liveBaseName)
	memFile := filepath.Join(plan.SnapshotDir, liveMemFileName)
	if _, err := os.Stat(memFile); err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("live snapshot memory image missing: %w", err)
	}
	if _, err := os.Stat(base); os.IsNotExist(err) {
		// Non-standard base name (snapshot taken while a live overlay was active).
		if matches, _ := filepath.Glob(filepath.Join(plan.SnapshotDir, "base-*.qcow2")); len(matches) == 1 {
			base = matches[0]
		} else {
			return rtm.RuntimeInstance{}, fmt.Errorf("live snapshot disk image missing: %w", err)
		}
	} else if err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("inspect snapshot disk: %w", err)
	}

	// Rebuild the machine plan from what was saved (identical topology).
	var machinePlan rtm.MachinePlan
	if data, err := os.ReadFile(filepath.Join(plan.SnapshotDir, livePlanFileName)); err == nil {
		_ = json.Unmarshal(data, &machinePlan)
	}
	machinePlan.Name = plan.Instance.Name
	machinePlan.RuntimeDir = plan.Instance.RuntimeDir
	if machinePlan.LogPath == "" {
		machinePlan.LogPath = filepath.Join(plan.Instance.RuntimeDir, "vm.log")
	}
	machinePlan.IncomingDefer = true

	target := filepath.Join(plan.Instance.RuntimeDir, activeDiskName)
	if err := os.MkdirAll(plan.Instance.RuntimeDir, 0755); err != nil {
		return rtm.RuntimeInstance{}, err
	}
	if err := os.Chmod(base, 0644); err == nil {
		defer os.Chmod(base, 0444)
	}
	if _, err := os.Stat(target); err == nil {
		_ = os.Remove(target) // previous active disk is superseded
	}
	if err := runCommand(ctx, qemuImgBinary, buildCreateOverlayArgs(rtm.DiskPlan{
		BaseImagePath: base,
		DiskPath:      target,
	})...); err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("create restore overlay: %w", err)
	}
	machinePlan.Disk.DiskPath = target

	// Restart virtiofs and egress sidecars for mounts/egress recorded in the plan.
	if err := prepareEgress(&machinePlan); err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("prepare egress: %w", err)
	}
	if err := spawnMountSidecars(ctx, machinePlan); err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("restart virtiofs sidecars: %w", err)
	}
	if err := spawnEgressSidecar(machinePlan); err != nil {
		stopMountSidecars(plan.Instance.RuntimeDir)
		stopEgressSidecar(plan.Instance.RuntimeDir)
		return rtm.RuntimeInstance{}, fmt.Errorf("restart egress sidecar: %w", err)
	}
	cleanupSidecars := func() {
		stopMountSidecars(plan.Instance.RuntimeDir)
		stopEgressSidecar(plan.Instance.RuntimeDir)
	}
	binary, args, err := buildCommand(machinePlan)
	if err != nil {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, err
	}
	if err := removeStaleQMPSocket(plan.Instance.RuntimeDir); err != nil {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, err
	}
	logFile, err := os.OpenFile(machinePlan.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, fmt.Errorf("open log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	proc, err := startProcess(ctx, binary, args, logFile, logFile)
	if err != nil {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, fmt.Errorf("start qemu for live restore: %w", err)
	}
	pid := proc.PID()
	if pid <= 0 {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, fmt.Errorf("qemu returned invalid pid")
	}
	if err := proc.Release(); err != nil {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, fmt.Errorf("release qemu handle: %w", err)
	}

	socketPath := qmpSocketPath(plan.Instance.RuntimeDir)
	client, err := newQMPClient(socketPath, 10*time.Second)
	if err != nil {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, fmt.Errorf("connect qmp on respawned qemu: %w", err)
	}
	defer client.close()
	secureQMPSocket(plan.Instance.RuntimeDir)

	enableMigrationCapabilities(client)
	if err := client.execute("migrate-incoming", map[string]any{"uri": "file:" + memFile}); err != nil {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, fmt.Errorf("migrate-incoming: %w", err)
	}
	if err := waitRestore(ctx, client, liveMigrateTimeout); err != nil {
		cleanupSidecars()
		return rtm.RuntimeInstance{}, fmt.Errorf("incoming migration: %w", err)
	}
	// Guest lands paused when the source was stopped — resume it.
	if st, _ := queryStatus(client); st == "paused" {
		_ = client.execute("cont", nil)
	}

	inst := rtm.RuntimeInstance{
		Name:       plan.Instance.Name,
		RuntimeDir: plan.Instance.RuntimeDir,
		LogPath:    machinePlan.LogPath,
		PID:        pid,
		Networks:   machinePlan.Networks,
		StartedAt:  time.Now().UTC(),
	}
	if machinePlan.Display != nil {
		inst.Display = &rtm.DisplayInfo{
			Type:     machinePlan.Display.Type,
			Host:     machinePlan.Display.Host,
			Port:     machinePlan.Display.Port,
			URL:      fmt.Sprintf("vnc://%s:%d", machinePlan.Display.Host, machinePlan.Display.Port),
			GPU:      machinePlan.Display.GPU,
			Firmware: machinePlan.Firmware,
		}
	}
	return inst, nil
}

// --- helpers -----------------------------------------------------------------

// enableMigrationCapabilities turns on mapped-ram (+ multifd only when mapped-ram
// is also supported, since QEMU rejects multifd on file: URIs without mapped-ram).
func enableMigrationCapabilities(client *qmpClient) string {
	raw, err := client.executeQuery("query-migrate-capabilities", nil)
	if err != nil {
		return "file"
	}
	var caps []struct {
		Capability string `json:"capability"`
		State      bool   `json:"state"`
	}
	if err := json.Unmarshal(raw, &caps); err != nil {
		return "file"
	}
	have := map[string]bool{}
	for _, c := range caps {
		have[c.Capability] = true
	}
	var set []map[string]any
	method := "file"
	if have["mapped-ram"] {
		set = append(set, map[string]any{"capability": "mapped-ram", "state": true})
		method = "mapped-ram"
		if have["multifd"] {
			set = append(set, map[string]any{"capability": "multifd", "state": true})
		}
	}
	if len(set) == 0 {
		return method
	}
	if err := client.execute("migrate-set-capabilities", map[string]any{"capabilities": set}); err != nil {
		return "file"
	}
	return method
}

// findDiskDevice returns the QMP device id whose inserted file is diskPath.
func findDiskDevice(client *qmpClient, diskPath string) (string, error) {
	raw, err := client.executeQuery("query-block", nil)
	if err != nil {
		return "", fmt.Errorf("query-block: %w", err)
	}
	var blocks []struct {
		Device   string `json:"device"`
		Inserted struct {
			File string `json:"file"`
		} `json:"inserted"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", fmt.Errorf("parse query-block: %w", err)
	}
	want := filepath.Clean(diskPath)
	for _, b := range blocks {
		if filepath.Clean(b.Inserted.File) == want {
			return b.Device, nil
		}
	}
	// Fallback: first block with a qcow2 file.
	for _, b := range blocks {
		if strings.HasSuffix(b.Inserted.File, ".qcow2") {
			return b.Device, nil
		}
	}
	return "", fmt.Errorf("no block device found for disk %s", diskPath)
}

// findOverlayNode returns the node-name of the new active overlay for device.
func findOverlayNode(client *qmpClient, device, overlayPath string) (string, error) {
	raw, err := client.executeQuery("query-block", nil)
	if err != nil {
		return "", err
	}
	var blocks []struct {
		Device   string `json:"device"`
		Inserted struct {
			NodeName string `json:"node-name"`
			File     string `json:"file"`
		} `json:"inserted"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", err
	}
	for _, b := range blocks {
		if b.Device == device && filepath.Clean(b.Inserted.File) == filepath.Clean(overlayPath) {
			return b.Inserted.NodeName, nil
		}
	}
	return "", fmt.Errorf("overlay node not found for device %s", device)
}

// waitMigration polls query-migrate until status=completed.
func waitMigration(ctx context.Context, client *qmpClient, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		raw, err := client.executeQuery("query-migrate", nil)
		if err != nil {
			return err
		}
		var st struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(raw, &st)
		switch st.Status {
		case "completed":
			return nil
		case "failed", "cancelled":
			return fmt.Errorf("migration %s", st.Status)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("migration timed out (status=%s)", st.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(livePollInterval):
		}
	}
}

// waitRestore polls query-status until the guest reaches a post-restore
// state (running or paused — paused is resumed by the caller).
func waitRestore(ctx context.Context, client *qmpClient, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		st, err := queryStatus(client)
		if err == nil && (st == "running" || st == "paused") {
			return nil
		}
		raw, qerr := client.executeQuery("query-migrate", nil)
		if qerr == nil {
			var mst struct {
				Status string `json:"status"`
			}
			_ = json.Unmarshal(raw, &mst)
			if mst.Status == "failed" {
				return fmt.Errorf("incoming migration failed")
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("incoming migration timed out (status=%s)", st)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(livePollInterval):
		}
	}
}

func queryStatus(client *qmpClient) (string, error) {
	raw, err := client.executeQuery("query-status", nil)
	if err != nil {
		return "", err
	}
	var st struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(raw, &st)
	return st.Status, nil
}

// waitJobComplete polls query-jobs for the snapshot-save job finishing.
func waitJobComplete(ctx context.Context, client *qmpClient, jobID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		raw, err := client.executeQuery("query-jobs", nil)
		if err != nil {
			return err
		}
		var jobs []struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			LastError string `json:"error"`
		}
		_ = json.Unmarshal(raw, &jobs)
		found := false
		for _, j := range jobs {
			if j.ID != jobID {
				continue
			}
			found = true
			switch j.Status {
			case "concluded", "null":
				if j.LastError != "" {
					return fmt.Errorf("snapshot job failed: %s", j.LastError)
				}
				return nil
			case "aborting":
				return fmt.Errorf("snapshot job aborting")
			}
		}
		if !found {
			return nil // job already dismissed/gone = completed
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("snapshot job timed out")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(livePollInterval):
		}
	}
}
