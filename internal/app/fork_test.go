package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
)

// fakeForkRuntime records disk/fork operations and can fail on demand.
type fakeForkRuntime struct {
	fakeLiveRuntime
	prepared   []rtm.MachinePlan
	prepareErr error
	started    []rtm.MachinePlan
	startErr   error
	destroyed  []string
	restores   []rtm.SnapshotPlan
	restoreErr error
	deletes    []rtm.DeleteSnapshotPlan
}

func (f *fakeForkRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	f.prepared = append(f.prepared, plan)
	if f.prepareErr != nil {
		return rtm.DiskPlan{}, f.prepareErr
	}
	return plan.Disk, nil
}

func (f *fakeForkRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	f.started = append(f.started, plan)
	if f.startErr != nil {
		return rtm.RuntimeInstance{}, f.startErr
	}
	return rtm.RuntimeInstance{Name: plan.Name, RuntimeDir: plan.RuntimeDir, PID: 5000 + len(f.started)}, nil
}

func (f *fakeForkRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	f.destroyed = append(f.destroyed, instance.Name)
	return nil
}

func (f *fakeForkRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	f.restores = append(f.restores, plan)
	return f.restoreErr
}

func (f *fakeForkRuntime) DeleteSnapshot(ctx context.Context, plan rtm.DeleteSnapshotPlan) error {
	f.deletes = append(f.deletes, plan)
	return nil
}

// seedStoppedWithPlan creates a stopped "web" instance with a boot plan.
func seedStoppedWithPlan(t *testing.T, service *Service, root string, metadata project.Metadata) (project.Paths, string) {
	t.Helper()
	paths, err := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := state.Load(paths.StateFile, metadata.ID)
	inst := st.Instances["web"]
	inst.SSHPort = 2222
	inst.Status = "stopped"
	inst.Display = &state.DisplayState{
		Type: "vnc", Host: "127.0.0.1", Port: 5900, URL: "vnc://127.0.0.1:5900",
	}
	inst.ServicePorts = []state.PortForwardState{
		{Name: "http", Host: "127.0.0.1", HostPort: 8080, GuestPort: 80, Protocol: "tcp"},
	}
	plan := rtm.MachinePlan{
		Name: "web", MemoryMiB: 512, RuntimeDir: inst.RuntimeDir,
		Networks: rtm.NetworkPlan{
			Management: rtm.ManagementNetworkPlan{
				SSHHost: "127.0.0.1", SSHPort: 2222, InterfaceName: "eth0",
				PortForwards: []rtm.PortForwardPlan{
					{Name: "http", Host: "127.0.0.1", HostPort: 8080, GuestPort: 80, Protocol: "tcp"},
				},
			},
		},
		Display: &rtm.DisplayPlan{Type: "vnc", Host: "127.0.0.1", Port: 5900},
	}
	if err := writeBootPlan(inst.RuntimeDir, plan); err != nil {
		t.Fatal(err)
	}
	st.Instances["web"] = inst
	if err := state.Save(paths.StateFile, st); err != nil {
		t.Fatal(err)
	}
	return paths, inst.RuntimeDir
}

func TestForkCreatesTwoStoppedSiblings(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := seedStoppedWithPlan(t, service, root, metadata)

	result, err := service.Fork(context.Background(), ForkOptions{
		ProjectRoot: root, Target: "web", Count: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Forks) != 2 {
		t.Fatalf("forks = %+v", result.Forks)
	}
	// names + distinct ports
	names := map[string]bool{}
	ports := map[int]bool{}
	for _, fk := range result.Forks {
		names[fk.Name] = true
		ports[fk.SSHPort] = true
		if fk.Status != "stopped" {
			t.Fatalf("fork status = %s", fk.Status)
		}
	}
	if !names["web-fork-1"] || !names["web-fork-2"] {
		t.Fatalf("unexpected fork names %v", names)
	}
	if len(ports) != 2 || ports[2222] {
		t.Fatalf("ssh ports not distinct/free: %v", ports)
	}
	// each fork got a PrepareDisk call backed by the implicit snapshot base
	if len(fake.prepared) != 2 {
		t.Fatalf("prepared = %d", len(fake.prepared))
	}
	for i, plan := range fake.prepared {
		if plan.Disk.BaseImagePath == "" || plan.Disk.BaseImagePath == plan.Disk.DiskPath {
			t.Fatalf("fork %d disk not backed by a frozen base: %+v", i, plan.Disk)
		}
	}
	// state: ForkOf + display + forwards re-allocated
	st, _ := state.Load(paths.StateFile, metadata.ID)
	f1 := st.Instances["web-fork-1"]
	if f1.ForkOf != "web" || f1.SSHPort == 2222 {
		t.Fatalf("fork state = %+v", f1)
	}
	if f1.Display == nil || f1.Display.Port == 5900 {
		t.Fatalf("fork display not relocated: %+v", f1.Display)
	}
	if len(f1.ServicePorts) != 1 || f1.ServicePorts[0].HostPort == 8080 {
		t.Fatalf("fork port forward not relocated: %+v", f1.ServicePorts)
	}
	// parent got an implicit forkseed snapshot
	parent := st.Instances["web"]
	if len(parent.Snapshots) != 1 {
		t.Fatalf("parent snapshots = %+v", parent.Snapshots)
	}
	for name, snap := range parent.Snapshots {
		if snap.Kind != "overlay" || len(name) < 8 || name[:8] != "forkseed" {
			t.Fatalf("implicit seed snapshot = %s %+v", name, snap)
		}
	}
}

func TestForkFromSnapshotUsesBase(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := seedStoppedWithPlan(t, service, root, metadata)

	// register an overlay snapshot
	st, _ := state.Load(paths.StateFile, metadata.ID)
	inst := st.Instances["web"]
	snapDir := filepath.Join(inst.RuntimeDir, "snapshots", "golden")
	base := filepath.Join(snapDir, "base.qcow2")
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base, []byte("base"), 0644); err != nil {
		t.Fatal(err)
	}
	inst.Snapshots = map[string]state.SnapshotState{
		"golden": {Name: "golden", Kind: "overlay", Dir: snapDir, DiskPath: base},
	}
	st.Instances["web"] = inst
	state.Save(paths.StateFile, st)

	result, err := service.Fork(context.Background(), ForkOptions{
		ProjectRoot: root, Target: "web", Count: 1, From: "golden",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fake.prepared[0].Disk.BaseImagePath != base {
		t.Fatalf("fork base = %s want %s", fake.prepared[0].Disk.BaseImagePath, base)
	}
	// no implicit forkseed snapshot should have been added
	st, _ = state.Load(paths.StateFile, metadata.ID)
	if len(st.Instances["web"].Snapshots) != 1 {
		t.Fatalf("parent snapshots changed: %+v", st.Instances["web"].Snapshots)
	}
	_ = result
}

func TestForkRunningParentWithoutFromFails(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	service.runtime = &fakeForkRuntime{}
	paths, _ := seedStoppedWithPlan(t, service, root, metadata)
	st, _ := state.Load(paths.StateFile, metadata.ID)
	inst := st.Instances["web"]
	inst.Status = "running"
	inst.PID = 1234
	st.Instances["web"] = inst
	state.Save(paths.StateFile, st)

	_, err := service.Fork(context.Background(), ForkOptions{ProjectRoot: root, Target: "web", Count: 1})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestForkBootStartsForks(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := seedStoppedWithPlan(t, service, root, metadata)

	result, err := service.Fork(context.Background(), ForkOptions{
		ProjectRoot: root, Target: "web", Count: 1, Boot: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Forks[0].Status != "running" || result.Forks[0].PID == 0 {
		t.Fatalf("fork = %+v", result.Forks[0])
	}
	st, _ := state.Load(paths.StateFile, metadata.ID)
	if st.Instances["web-fork-1"].Status != "running" {
		t.Fatalf("fork state = %+v", st.Instances["web-fork-1"])
	}
}

func TestForkRollsBackOnPrepareFailure(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{prepareErr: fmt.Errorf("qemu-img boom")}
	service.runtime = fake
	paths, _ := seedStoppedWithPlan(t, service, root, metadata)

	_, err := service.Fork(context.Background(), ForkOptions{
		ProjectRoot: root, Target: "web", Count: 2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, _ := state.Load(paths.StateFile, metadata.ID)
	if _, exists := st.Instances["web-fork-1"]; exists {
		t.Fatal("rolled-back fork still in state")
	}
	if len(fake.destroyed) != 1 {
		t.Fatalf("destroyed = %v", fake.destroyed)
	}
}

func TestDestroyForksRemovesOnlyForks(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := seedStoppedWithPlan(t, service, root, metadata)

	if _, err := service.Fork(context.Background(), ForkOptions{
		ProjectRoot: root, Target: "web", Count: 2,
	}); err != nil {
		t.Fatal(err)
	}
	result, err := service.DestroyForks(context.Background(), DestroyOptions{ProjectRoot: root}, "web")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Instances) != 2 {
		t.Fatalf("destroyed = %+v", result.Instances)
	}
	st, _ := state.Load(paths.StateFile, metadata.ID)
	if _, ok := st.Instances["web"]; !ok {
		t.Fatal("parent destroyed with forks")
	}
	if _, exists := st.Instances["web-fork-1"]; exists {
		t.Fatal("fork-1 still in state")
	}
	// second call: no forks left
	_, err = service.DestroyForks(context.Background(), DestroyOptions{ProjectRoot: root}, "web")
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

// --- --all snapshot / restore -------------------------------------------------

func seedSecondStopped(t *testing.T, root string, metadata project.Metadata, paths project.Paths) {
	t.Helper()
	rtDir, err := paths.InstanceDir("api")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rtDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rtDir, "disk.qcow2"), []byte("d2"), 0644); err != nil {
		t.Fatal(err)
	}
	st, _ := state.Load(paths.StateFile, metadata.ID)
	st.Instances["api"] = state.InstanceState{
		Status: "stopped", RuntimeDir: rtDir, SSHPort: 2223,
	}
	if err := state.Save(paths.StateFile, st); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotAllCreatesSharedCheckpoint(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	seedSecondStopped(t, root, metadata, paths)

	result, err := service.SnapshotAll(context.Background(), SnapshotAllOptions{
		ProjectRoot: root, Name: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Snapshots) != 2 {
		t.Fatalf("snapshots = %+v", result.Snapshots)
	}
	st, _ := state.Load(paths.StateFile, metadata.ID)
	for _, name := range []string{"web", "api"} {
		if _, ok := st.Instances[name].Snapshots["v1"]; !ok {
			t.Fatalf("instance %s missing snapshot v1", name)
		}
	}
}

func TestSnapshotAllRejectsDuplicateName(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	service.runtime = &fakeForkRuntime{}
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	seedSecondStopped(t, root, metadata, paths)

	if _, err := service.SnapshotAll(context.Background(), SnapshotAllOptions{
		ProjectRoot: root, Name: "v1",
	}); err != nil {
		t.Fatal(err)
	}
	// rename web's snapshot to collide
	st, _ := state.Load(paths.StateFile, metadata.ID)
	inst := st.Instances["web"]
	inst.Snapshots["v1b"] = inst.Snapshots["v1"]
	delete(inst.Snapshots, "v1")
	st.Instances["web"] = inst
	state.Save(paths.StateFile, st)

	// "v1" now collides only on api — should fail BEFORE creating web's
	_, err := service.SnapshotAll(context.Background(), SnapshotAllOptions{
		ProjectRoot: root, Name: "v1",
	})
	assertAppErrorCode(t, err, ErrorCodeConflict)
}

func TestRestoreAllPreflightBlocksPartialRestore(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	seedSecondStopped(t, root, metadata, paths)

	// only web has snapshot "v1"
	st, _ := state.Load(paths.StateFile, metadata.ID)
	inst := st.Instances["web"]
	inst.Snapshots = map[string]state.SnapshotState{
		"v1": {Name: "v1", Kind: "overlay", DiskPath: filepath.Join(inst.RuntimeDir, "snapshots", "v1", "base.qcow2")},
	}
	st.Instances["web"] = inst
	state.Save(paths.StateFile, st)

	_, err := service.RestoreAll(context.Background(), RestoreAllOptions{
		ProjectRoot: root, Name: "v1",
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
	if len(fake.restores) != 0 {
		t.Fatalf("preflight failed but restores ran: %+v", fake.restores)
	}
}

func TestRestoreAllRestoresEveryInstance(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	seedSecondStopped(t, root, metadata, paths)

	if _, err := service.SnapshotAll(context.Background(), SnapshotAllOptions{
		ProjectRoot: root, Name: "v1",
	}); err != nil {
		t.Fatal(err)
	}
	result, err := service.RestoreAll(context.Background(), RestoreAllOptions{
		ProjectRoot: root, Name: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Instances) != 2 {
		t.Fatalf("restored = %+v", result.Instances)
	}
	// each instance got 1 safety create + 1 restore + 1 safety delete
	if len(fake.restores) != 2 {
		t.Fatalf("restores = %d", len(fake.restores))
	}
	if len(fake.deletes) != 2 {
		t.Fatalf("safety deletes = %d", len(fake.deletes))
	}
}

func TestRestoreAllRollsBackOnFailure(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	seedSecondStopped(t, root, metadata, paths)

	if _, err := service.SnapshotAll(context.Background(), SnapshotAllOptions{
		ProjectRoot: root, Name: "v1",
	}); err != nil {
		t.Fatal(err)
	}
	// fail on the second restore call (instances sorted: api, web)
	calls := 0
	orig := fake.RestoreSnapshot
	_ = orig
	fakeRestore := &countingRestore{inner: fake, failOn: 2}
	service.runtime = fakeRestore

	_, err := service.RestoreAll(context.Background(), RestoreAllOptions{
		ProjectRoot: root, Name: "v1",
	})
	if err == nil {
		t.Fatal("expected restore failure")
	}
	_ = calls
	// the rolled-back instance gets a second RestoreSnapshot (safety restore)
	if len(fakeRestore.restores) < 3 {
		t.Fatalf("expected rollback restores, got %d", len(fakeRestore.restores))
	}
}

type countingRestore struct {
	inner    *fakeForkRuntime
	failOn   int
	count    int
	restores []rtm.SnapshotPlan
}

func (c *countingRestore) PrepareDisk(ctx context.Context, p rtm.MachinePlan) (rtm.DiskPlan, error) {
	return c.inner.PrepareDisk(ctx, p)
}
func (c *countingRestore) Start(ctx context.Context, p rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return c.inner.Start(ctx, p)
}
func (c *countingRestore) Stop(ctx context.Context, i rtm.RuntimeInstance, d time.Duration) error {
	return c.inner.Stop(ctx, i, d)
}
func (c *countingRestore) Inspect(ctx context.Context, i rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return c.inner.Inspect(ctx, i)
}
func (c *countingRestore) CreateSnapshot(ctx context.Context, p rtm.SnapshotPlan) error {
	return c.inner.CreateSnapshot(ctx, p)
}
func (c *countingRestore) RestoreSnapshot(ctx context.Context, p rtm.SnapshotPlan) error {
	c.count++
	c.restores = append(c.restores, p)
	if c.count == c.failOn {
		return fmt.Errorf("simulated restore failure")
	}
	return c.inner.RestoreSnapshot(ctx, p)
}
func (c *countingRestore) DeleteSnapshot(ctx context.Context, p rtm.DeleteSnapshotPlan) error {
	return c.inner.DeleteSnapshot(ctx, p)
}
func (c *countingRestore) Destroy(ctx context.Context, i rtm.RuntimeInstance) error {
	return c.inner.Destroy(ctx, i)
}

// --- clean --snapshots + snapshots bytes --------------------------------------

func TestCleanSnapshotsRemovesAll(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeForkRuntime{}
	service.runtime = fake
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)

	if _, err := service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root, Target: "web", Name: "s1",
	}); err != nil {
		t.Fatal(err)
	}
	st, _ := state.Load(paths.StateFile, metadata.ID)
	if len(st.Instances["web"].Snapshots) != 1 {
		t.Fatal("setup: snapshot missing")
	}

	result, err := service.CleanSnapshots(context.Background(), CleanOptions{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Instances) != 1 {
		t.Fatalf("clean result = %+v", result)
	}
	st, _ = state.Load(paths.StateFile, metadata.ID)
	if len(st.Instances["web"].Snapshots) != 0 {
		t.Fatalf("snapshots remain: %+v", st.Instances["web"].Snapshots)
	}
}

func TestSnapshotsReportsAllocatedBytes(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	service.runtime = &fakeForkRuntime{}
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)

	if _, err := service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root, Target: "web", Name: "s1",
	}); err != nil {
		t.Fatal(err)
	}
	// fake runtime doesn't write base.qcow2 — drop a real file so the walk sees bytes
	st, _ := state.Load(paths.StateFile, metadata.ID)
	snap := st.Instances["web"].Snapshots["s1"]
	if err := os.MkdirAll(snap.Dir, 0755); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 4096)
	if err := os.WriteFile(filepath.Join(snap.Dir, "base.qcow2"), payload, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := service.Snapshots(context.Background(), SnapshotsOptions{
		ProjectRoot: root, Target: "web",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Snapshots) != 1 || result.Snapshots[0].Bytes < 4096 {
		t.Fatalf("bytes = %+v", result.Snapshots)
	}
}
