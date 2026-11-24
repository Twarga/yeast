package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const activeDiskName = "disk.qcow2"

// CreateOverlaySnapshot freezes the instance's current disk into
// <snapDir>/base.qcow2 (read-only) and starts a fresh thin overlay at
// <RuntimeDir>/disk.qcow2. O(rename + qemu-img create) — well under a second.
func CreateOverlaySnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	if plan.InstanceDiskPath == "" || plan.SnapshotPath == "" || plan.RuntimeDir == "" {
		return fmt.Errorf("instance disk, snapshot path and runtime dir are required")
	}
	if _, err := os.Stat(plan.InstanceDiskPath); err != nil {
		return fmt.Errorf("inspect instance disk %s: %w", plan.InstanceDiskPath, err)
	}
	if _, err := os.Stat(plan.SnapshotPath); err == nil {
		return fmt.Errorf("snapshot file %s already exists", plan.SnapshotPath)
	}
	snapDir := filepath.Dir(plan.SnapshotPath)
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		return fmt.Errorf("create snapshot directory %s: %w", snapDir, err)
	}

	target := filepath.Join(plan.RuntimeDir, activeDiskName)
	if _, err := os.Stat(target); err == nil && filepath.Clean(plan.InstanceDiskPath) != filepath.Clean(target) {
		return fmt.Errorf("active disk target %s already exists", target)
	}

	// 1. Freeze: move the current disk file into the snapshot dir.
	if err := os.Rename(plan.InstanceDiskPath, plan.SnapshotPath); err != nil {
		return fmt.Errorf("freeze disk into snapshot: %w", err)
	}
	rollback := func() {
		_ = os.Chmod(plan.SnapshotPath, 0644)
		_ = os.Rename(plan.SnapshotPath, plan.InstanceDiskPath)
	}
	if err := os.Chmod(plan.SnapshotPath, 0444); err != nil {
		rollback()
		return fmt.Errorf("mark snapshot base read-only: %w", err)
	}

	// 2. Fresh thin overlay becomes the new active disk.
	if err := runCommand(ctx, qemuImgBinary, buildCreateOverlayArgs(rtm.DiskPlan{
		BaseImagePath: plan.SnapshotPath,
		DiskPath:      target,
	})...); err != nil {
		rollback()
		return fmt.Errorf("create overlay disk: %w", err)
	}
	return nil
}

// RestoreOverlaySnapshot swaps the active disk for a fresh thin overlay on the
// snapshot's frozen base. The previous active disk is moved aside first so a
// failed restore can be rolled back; it is deleted only on success.
func RestoreOverlaySnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	if plan.SnapshotPath == "" || plan.RuntimeDir == "" {
		return fmt.Errorf("snapshot path and runtime dir are required")
	}
	if _, err := os.Stat(plan.SnapshotPath); err != nil {
		return fmt.Errorf("inspect snapshot file %s: %w", plan.SnapshotPath, err)
	}
	if err := os.MkdirAll(plan.RuntimeDir, 0755); err != nil {
		return err
	}
	target := filepath.Join(plan.RuntimeDir, activeDiskName)
	backup := target + ".pre-restore"

	// Move any current active disk aside (both the conventional disk.qcow2 and
	// an external active layer recorded by the caller via InstanceDiskPath).
	replaced := []string{}
	for _, cur := range uniqueNonEmpty(target, plan.InstanceDiskPath) {
		if _, err := os.Stat(cur); err != nil {
			continue
		}
		dst := cur + ".pre-restore"
		if filepath.Clean(cur) == filepath.Clean(target) {
			dst = backup
		}
		_ = os.Remove(dst)
		if err := os.Rename(cur, dst); err != nil {
			return fmt.Errorf("move aside current disk %s: %w", cur, err)
		}
		replaced = append(replaced, dst)
	}

	err := runCommand(ctx, qemuImgBinary, buildCreateOverlayArgs(rtm.DiskPlan{
		BaseImagePath: plan.SnapshotPath,
		DiskPath:      target,
	})...)
	if err != nil {
		// roll back
		for _, dst := range replaced {
			orig := strings.TrimSuffix(dst, ".pre-restore")
			_ = os.Rename(dst, orig)
		}
		return fmt.Errorf("create restore overlay: %w", err)
	}
	for _, dst := range replaced {
		_ = os.Remove(dst)
	}
	return nil
}

// DeleteOverlaySnapshot removes a snapshot directory after rebasing every
// qcow2 that still references its base (active disk, sibling snapshot bases,
// live overlays). Chain integrity is preserved; nothing dangles.
func DeleteOverlaySnapshot(ctx context.Context, plan rtm.DeleteSnapshotPlan) error {
	if plan.Path == "" {
		return fmt.Errorf("snapshot path is required")
	}
	base := filepath.Clean(plan.Path)
	snapDir := filepath.Dir(base)
	snapshotsRoot := filepath.Dir(snapDir)

	if plan.ActiveDiskPath != "" {
		cleanActive := filepath.Clean(plan.ActiveDiskPath)
		if cleanActive == snapDir || strings.HasPrefix(cleanActive, snapDir+string(filepath.Separator)) {
			return fmt.Errorf("cannot delete snapshot %s while active disk %s lives inside it (restore or create a new snapshot first)", filepath.Base(snapDir), cleanActive)
		}
	}

	children, err := findChainChildren(ctx, base, plan.RuntimeDir, snapshotsRoot, plan.ActiveDiskPath)
	if err != nil {
		return err
	}
	newParent, err := qemuImgBackingFile(ctx, base)
	if err != nil {
		// Unreadable base — safest is to flatten children fully.
		newParent = ""
	}
	for _, child := range children {
		// Read-only sibling snapshot bases (0444) must be temporarily writable for qemu-img rebase.
		var restoreMode os.FileMode
		if info, statErr := os.Stat(child); statErr == nil && info.Mode().Perm()&0200 == 0 {
			restoreMode = info.Mode().Perm()
			if err := os.Chmod(child, 0644); err != nil {
				return fmt.Errorf("unprotect child %s for rebase: %w", child, err)
			}
		}
		args := []string{"rebase", "-f", "qcow2", "-b", newParent}
		if newParent != "" {
			args = append(args, "-F", "qcow2")
		}
		args = append(args, child)
		rebaseErr := runCommand(ctx, qemuImgBinary, args...)
		if restoreMode != 0 {
			_ = os.Chmod(child, restoreMode)
		}
		if rebaseErr != nil {
			return fmt.Errorf("rebase %s off deleted snapshot: %w", child, rebaseErr)
		}
	}
	if err := os.Chmod(base, 0644); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unprotect snapshot base: %w", err)
	}
	if err := os.RemoveAll(snapDir); err != nil {
		return fmt.Errorf("remove snapshot dir %s: %w", snapDir, err)
	}
	return nil
}

// findChainChildren returns every qcow2 whose backing file is base, including sibling forks.
func findChainChildren(ctx context.Context, base, runtimeDir, snapshotsRoot, activeDisk string) ([]string, error) {
	candidates := map[string]bool{}
	if runtimeDir != "" {
		if matches, _ := filepath.Glob(filepath.Join(runtimeDir, "*.qcow2")); len(matches) > 0 {
			for _, m := range matches {
				candidates[m] = true
			}
		}
		// Scan sibling instance/fork directories under the same project instances root.
		instancesRoot := filepath.Dir(runtimeDir)
		if matches, _ := filepath.Glob(filepath.Join(instancesRoot, "*", "*.qcow2")); len(matches) > 0 {
			for _, m := range matches {
				candidates[m] = true
			}
		}
	}
	if snapshotsRoot != "" {
		if matches, _ := filepath.Glob(filepath.Join(snapshotsRoot, "*", "*.qcow2")); len(matches) > 0 {
			for _, m := range matches {
				candidates[m] = true
			}
		}
	}
	if activeDisk != "" {
		candidates[activeDisk] = true
	}
	var children []string
	for cand := range candidates {
		if filepath.Clean(cand) == base {
			continue
		}
		backing, err := qemuImgBackingFile(ctx, cand)
		if err != nil {
			continue // unreadable/not qcow2 — can't be a child
		}
		if filepath.Clean(backing) == base {
			children = append(children, cand)
		}
	}
	return children, nil
}

// qemuImgBackingFile returns the resolved absolute backing filename recorded
// in a qcow2 header ("" when the image has no backing file).
func qemuImgBackingFile(ctx context.Context, path string) (string, error) {
	out, err := runCommandOutput(ctx, qemuImgBinary, "info", "--output=json", path)
	if err != nil {
		return "", err
	}
	var info struct {
		BackingFilename     string `json:"backing-filename"`
		FullBackingFilename string `json:"full-backing-filename"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return "", fmt.Errorf("parse qemu-img info: %w", err)
	}
	if info.FullBackingFilename != "" {
		return info.FullBackingFilename, nil
	}
	if info.BackingFilename == "" {
		return "", nil
	}
	if filepath.IsAbs(info.BackingFilename) {
		return info.BackingFilename, nil
	}
	abs, err := filepath.Abs(filepath.Join(filepath.Dir(path), info.BackingFilename))
	if err != nil {
		return "", err
	}
	return abs, nil
}

// commandOutput is injectable so tests can stub qemu-img info/rebase.
var commandOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

// runCommandOutput runs a command and returns stdout.
func runCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return commandOutput(ctx, name, args...)
}

func uniqueNonEmpty(paths ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		if p == "" || seen[filepath.Clean(p)] {
			continue
		}
		seen[filepath.Clean(p)] = true
		out = append(out, p)
	}
	return out
}
