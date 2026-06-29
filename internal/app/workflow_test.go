package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	"yeast/internal/guest"
	"yeast/internal/provision/cloudinit"
	rtm "yeast/internal/runtime"
)

func TestAppWorkflowsRunWithoutQEMUUsingFakeRuntime(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAATEST", nil }
	service.renderUserData = func(input cloudinit.UserDataInput) (string, error) {
		return "#cloud-config\nhostname: " + input.Hostname + "\n", nil
	}
	service.renderMetaData = func(input cloudinit.MetaDataInput) (string, error) {
		return "instance-id: " + input.Hostname + "\nlocal-hostname: " + input.Hostname + "\n", nil
	}
	service.createSeedISO = func(ctx context.Context, input cloudinit.SeedInput) (cloudinit.SeedResult, error) {
		return cloudinit.SeedResult{
			UserDataPath: filepath.Join(input.RuntimeDir, "user-data"),
			MetaDataPath: filepath.Join(input.RuntimeDir, "meta-data"),
			ISOPath:      filepath.Join(input.RuntimeDir, "seed.iso"),
			Builder:      "fake",
		}, nil
	}
	service.provisionTransport = fakeBootstrapTransport(t)
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }

	fakeRuntime := &workflowRuntime{
		nextPID: 4200,
	}
	service.runtime = fakeRuntime

	if _, err := service.Init(InitOptions{
		ProjectRoot: root,
		Now:         time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	imagePath := filepath.Join(yeastHome, "cache", "images", "ubuntu-24.04", "image.qcow2")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		t.Fatalf("create image cache dir: %v", err)
	}
	if err := os.WriteFile(imagePath, []byte("image"), 0644); err != nil {
		t.Fatalf("write cached image: %v", err)
	}

	upResult, err := service.Up(context.Background(), UpOptions{
		ProjectRoot:      root,
		ReadinessTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Up returned error: %v", err)
	}
	if len(upResult.Instances) != 1 {
		t.Fatalf("expected 1 up result, got %d", len(upResult.Instances))
	}
	if upResult.Instances[0].Status != "running" {
		t.Fatalf("expected running after up, got %#v", upResult.Instances[0])
	}

	statusAfterUp, err := service.Status(context.Background(), StatusOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Status after up returned error: %v", err)
	}
	if len(statusAfterUp.Instances) != 1 || statusAfterUp.Instances[0].Status != "running" {
		t.Fatalf("expected running status after up, got %#v", statusAfterUp.Instances)
	}

	downResult, err := service.Down(context.Background(), DownOptions{
		ProjectRoot: root,
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Down returned error: %v", err)
	}
	if len(downResult.Instances) != 1 || downResult.Instances[0].Status != "stopped" {
		t.Fatalf("expected stopped after down, got %#v", downResult.Instances)
	}

	statusAfterDown, err := service.Status(context.Background(), StatusOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Status after down returned error: %v", err)
	}
	if len(statusAfterDown.Instances) != 1 || statusAfterDown.Instances[0].Status != "stopped" {
		t.Fatalf("expected stopped status after down, got %#v", statusAfterDown.Instances)
	}

	destroyResult, err := service.Destroy(context.Background(), DestroyOptions{
		ProjectRoot: root,
		KeepFiles:   true,
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Destroy returned error: %v", err)
	}
	if len(destroyResult.Instances) != 1 || destroyResult.Instances[0].Status != "already_stopped" {
		t.Fatalf("expected destroyed result, got %#v", destroyResult.Instances)
	}

	statusAfterDestroy, err := service.Status(context.Background(), StatusOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Status after destroy returned error: %v", err)
	}
	if len(statusAfterDestroy.Instances) != 1 || statusAfterDestroy.Instances[0].Status != "stopped" {
		t.Fatalf("expected stopped state after keep-files destroy, got %#v", statusAfterDestroy.Instances)
	}

	if fakeRuntime.prepareCalls != 1 {
		t.Fatalf("expected one prepare call, got %d", fakeRuntime.prepareCalls)
	}
	if fakeRuntime.startCalls != 1 {
		t.Fatalf("expected one start call, got %d", fakeRuntime.startCalls)
	}
	if fakeRuntime.stopCalls != 1 {
		t.Fatalf("expected one stop call, got %d", fakeRuntime.stopCalls)
	}
	if fakeRuntime.destroyCalls != 0 {
		t.Fatalf("expected no destroy calls during keep-files destroy, got %d", fakeRuntime.destroyCalls)
	}
}

type workflowRuntime struct {
	nextPID      int
	prepareCalls int
	startCalls   int
	stopCalls    int
	destroyCalls int
	running      map[int]bool
}

func (f *workflowRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	f.prepareCalls++
	return plan.Disk, nil
}

func (f *workflowRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	f.startCalls++
	f.nextPID++
	if f.running == nil {
		f.running = make(map[int]bool)
	}
	f.running[f.nextPID] = true
	return rtm.RuntimeInstance{
		Name:       plan.Name,
		RuntimeDir: plan.RuntimeDir,
		LogPath:    plan.LogPath,
		PID:        f.nextPID,
		Networks:   plan.Networks,
		StartedAt:  time.Now().UTC(),
	}, nil
}

func (f *workflowRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	f.stopCalls++
	if f.running != nil {
		delete(f.running, instance.PID)
	}
	return nil
}

func (f *workflowRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	state := rtm.ProcessStateStopped
	if f.running != nil && f.running[instance.PID] {
		state = rtm.ProcessStateRunning
	}
	return rtm.ProcessInfo{
		PID:   instance.PID,
		State: state,
	}, nil
}

func (f *workflowRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *workflowRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *workflowRuntime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	return nil
}

func (f *workflowRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	f.destroyCalls++
	if f.running != nil {
		delete(f.running, instance.PID)
	}
	return nil
}
