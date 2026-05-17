package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"yeast/internal/guest"
	"yeast/internal/project"
	"yeast/internal/provision/cloudinit"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestUpStartsInstanceAndSavesState(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAATEST", nil }

	var userDataCalls int
	service.renderUserData = func(input cloudinit.UserDataInput) (string, error) {
		userDataCalls++
		if input.Hostname != "web" {
			t.Fatalf("unexpected hostname: %q", input.Hostname)
		}
		return "#cloud-config\nhostname: web\n", nil
	}
	service.renderMetaData = func(input cloudinit.MetaDataInput) (string, error) {
		return "instance-id: web\nlocal-hostname: web\n", nil
	}
	service.createSeedISO = func(ctx context.Context, input cloudinit.SeedInput) (cloudinit.SeedResult, error) {
		if input.InstanceName != "web" {
			t.Fatalf("unexpected instance for seed iso: %q", input.InstanceName)
		}
		return cloudinit.SeedResult{
			UserDataPath: filepath.Join(input.RuntimeDir, "user-data"),
			MetaDataPath: filepath.Join(input.RuntimeDir, "meta-data"),
			ISOPath:      filepath.Join(input.RuntimeDir, "seed.iso"),
			Builder:      "genisoimage",
		}, nil
	}

	fakeRuntime := &fakeRuntime{}
	service.runtime = fakeRuntime
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error {
		if options.Address != "127.0.0.1:2222" {
			t.Fatalf("unexpected readiness address: %q", options.Address)
		}
		return nil
	}

	if _, err := service.Init(InitOptions{ProjectRoot: root, Now: time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	configPath := filepath.Join(root, ConfigFileName)
	configContent := `version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 25 gb
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config with disk_size: %v", err)
	}

	imagePath := filepath.Join(yeastHome, "cache", "images", "ubuntu-24.04", "image.qcow2")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		t.Fatalf("create image cache dir: %v", err)
	}
	if err := os.WriteFile(imagePath, []byte("image"), 0644); err != nil {
		t.Fatalf("write cached image: %v", err)
	}

	result, err := service.Up(context.Background(), UpOptions{
		ProjectRoot:      root,
		ReadinessTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Up returned error: %v", err)
	}

	if len(result.Instances) != 1 {
		t.Fatalf("expected 1 instance result, got %d", len(result.Instances))
	}
	if result.Instances[0].SSHAddress != "127.0.0.1:2222" {
		t.Fatalf("unexpected ssh address: %q", result.Instances[0].SSHAddress)
	}
	if userDataCalls != 1 {
		t.Fatalf("expected one user-data render call, got %d", userDataCalls)
	}
	if fakeRuntime.preparePlan.Name != "web" {
		t.Fatalf("unexpected runtime prepare plan name: %q", fakeRuntime.preparePlan.Name)
	}
	if fakeRuntime.startPlan.Disk.BaseImagePath != imagePath {
		t.Fatalf("unexpected base image path: %q", fakeRuntime.startPlan.Disk.BaseImagePath)
	}
	if fakeRuntime.preparePlan.Disk.Size != "25G" {
		t.Fatalf("expected normalized disk size 25G, got %q", fakeRuntime.preparePlan.Disk.Size)
	}
	if fakeRuntime.startPlan.Disk.Size != "25G" {
		t.Fatalf("expected start plan disk size 25G, got %q", fakeRuntime.startPlan.Disk.Size)
	}

	statePath := filepath.Join(yeastHome, "projects", result.ProjectID, "state.json")
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	content := string(raw)
	for _, want := range []string{`"status": "running"`, `"ssh_port": 2222`, `"runtime_dir": "`} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected state content %q, got:\n%s", want, content)
		}
	}
}

func TestUpFailsClearlyWhenCachedImageMissing(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "run `yeast pull ubuntu-24.04`") {
		t.Fatalf("expected pull guidance, got %q", err)
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != ErrorCodeNotFound {
		t.Fatalf("expected not_found error code, got %q", appErr.Code)
	}
}

func TestUpReportsUnsupportedImageAsInvalidArgument(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	configContent := `version: 1
instances:
  - name: web
    image: unknown-image
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != ErrorCodeInvalidArgument {
		t.Fatalf("expected invalid_argument error code, got %q", appErr.Code)
	}
}

func TestUpClassifiesRuntimePrepareFailure(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.runtime = &fakeRuntime{prepareErr: errors.New("prepare failed")}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestUpClassifiesRuntimeStartFailure(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.runtime = &fakeRuntime{startErr: errors.New("start failed")}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestUpClassifiesSSHAddressFailureAndStopsStartedInstance(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	fake := &fakeRuntime{}
	service.runtime = fake
	service.sshAddress = func(host string, port int) (string, error) {
		return "", fmt.Errorf("invalid ssh target")
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
	if fake.stopCalls != 1 {
		t.Fatalf("expected one stop call after ssh address failure, got %d", fake.stopCalls)
	}
}

func TestUpClassifiesReadinessFailureAndStopsStartedInstance(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	fake := &fakeRuntime{}
	service.runtime = fake
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error {
		return errors.New("connection refused")
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
	if !strings.Contains(err.Error(), "wait for ssh readiness for web") {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.stopCalls != 1 {
		t.Fatalf("expected one stop call after readiness failure, got %d", fake.stopCalls)
	}
}

func TestUpClassifiesUninitializedProject(t *testing.T) {
	service := NewService()

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: t.TempDir()})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
	if !strings.Contains(err.Error(), "project metadata not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpClassifiesMissingConfig(t *testing.T) {
	root := t.TempDir()
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return filepath.Join(root, "yeast-home"), nil }

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if err := os.Remove(filepath.Join(root, ConfigFileName)); err != nil {
		t.Fatalf("remove config: %v", err)
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
	if !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpClassifiesInvalidConfig(t *testing.T) {
	root := t.TempDir()
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return filepath.Join(root, "yeast-home"), nil }

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	configContent := `version: 1
instances:
  - name: web
    image: ubuntu-24.04
    disk_size: not-a-size
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
	if !strings.Contains(err.Error(), "validate config file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpClassifiesStateProjectMismatch(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	imagePath := filepath.Join(yeastHome, "cache", "images", "ubuntu-24.04", "image.qcow2")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		t.Fatalf("create image cache dir: %v", err)
	}
	if err := os.WriteFile(imagePath, []byte("image"), 0644); err != nil {
		t.Fatalf("write cached image: %v", err)
	}

	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	if err := state.Save(paths.StateFile, state.New("wrong-project")); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
	if !strings.Contains(err.Error(), "state project id mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newUpServiceWithCachedImage(t *testing.T) (*Service, string) {
	t.Helper()

	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAATEST", nil }
	service.renderUserData = func(input cloudinit.UserDataInput) (string, error) { return "#cloud-config\n", nil }
	service.renderMetaData = func(input cloudinit.MetaDataInput) (string, error) { return "instance-id: web\n", nil }
	service.createSeedISO = func(ctx context.Context, input cloudinit.SeedInput) (cloudinit.SeedResult, error) {
		return cloudinit.SeedResult{ISOPath: filepath.Join(input.RuntimeDir, "seed.iso")}, nil
	}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	imagePath := filepath.Join(yeastHome, "cache", "images", "ubuntu-24.04", "image.qcow2")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		t.Fatalf("create image cache dir: %v", err)
	}
	if err := os.WriteFile(imagePath, []byte("image"), 0644); err != nil {
		t.Fatalf("write cached image: %v", err)
	}

	return service, root
}

type fakeRuntime struct {
	preparePlan rtm.MachinePlan
	startPlan   rtm.MachinePlan
	prepareErr  error
	startErr    error
	stopCalls   int
}

func (f *fakeRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	f.preparePlan = plan
	return plan.Disk, f.prepareErr
}

func (f *fakeRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	f.startPlan = plan
	if f.startErr != nil {
		return rtm.RuntimeInstance{}, f.startErr
	}
	return rtm.RuntimeInstance{
		Name:              plan.Name,
		RuntimeDir:        plan.RuntimeDir,
		LogPath:           plan.LogPath,
		PID:               4242,
		ManagementNetwork: plan.ManagementNetwork,
		StartedAt:         time.Now().UTC(),
	}, nil
}

func (f *fakeRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	f.stopCalls++
	return nil
}

func (f *fakeRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{PID: instance.PID, State: rtm.ProcessStateRunning}, nil
}

func (f *fakeRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}
