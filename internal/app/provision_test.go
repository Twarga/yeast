package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"yeast/internal/guest"
	"yeast/internal/project"
	provssh "yeast/internal/provision/ssh"
	"yeast/internal/state"
)

func TestProvisionRerunsForReachableInstance(t *testing.T) {
	service, root, metadata := newProvisionServiceWithRunningInstance(t)

	runRequests := make([]provssh.RunRequest, 0)
	uploadRequests := make([]provssh.UploadRequest, 0)
	service.provisionTransport = provssh.FakeTransport{
		RunFunc: func(ctx context.Context, request provssh.RunRequest) (provssh.RunResult, error) {
			runRequests = append(runRequests, request)
			return provssh.RunResult{Stdout: "ok\n", ExitCode: 0, Duration: time.Millisecond}, nil
		},
		UploadFunc: func(ctx context.Context, request provssh.UploadRequest) error {
			uploadRequests = append(uploadRequests, request)
			return nil
		},
	}

	result, err := service.Provision(context.Background(), ProvisionOptions{ProjectRoot: root, Target: "web"})
	if err != nil {
		t.Fatalf("Provision returned error: %v", err)
	}
	if result.ProjectID != metadata.ID {
		t.Fatalf("unexpected project id %q", result.ProjectID)
	}
	if result.Instance.Name != "web" || result.Instance.ProvisioningStatus != state.ProvisioningStatusReady {
		t.Fatalf("unexpected provision result: %#v", result)
	}

	wantCommands := []string{
		"cloud-init status --wait",
		"sudo -n DEBIAN_FRONTEND=noninteractive apt-get update && sudo -n DEBIAN_FRONTEND=noninteractive apt-get install -y curl caddy",
		"mkdir -p '/home/yeast/site'",
		"chmod 0644 '/home/yeast/site/index.html'",
		"echo project",
		"echo instance",
	}
	gotCommands := make([]string, 0, len(runRequests))
	for _, request := range runRequests {
		gotCommands = append(gotCommands, request.Command)
	}
	if strings.Join(gotCommands, "\n") != strings.Join(wantCommands, "\n") {
		t.Fatalf("unexpected provision command order:\n got: %#v\nwant: %#v", gotCommands, wantCommands)
	}
	if len(uploadRequests) != 1 {
		t.Fatalf("expected 1 upload request, got %d", len(uploadRequests))
	}
	if uploadRequests[0].Source != filepath.Join(root, "site", "index.html") {
		t.Fatalf("unexpected upload source: %q", uploadRequests[0].Source)
	}

	loaded, err := state.Load(filepath.Join(root, "yeast-home", "projects", metadata.ID, "state.json"), metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	instance := loaded.Instances["web"]
	if instance.ProvisioningStatus != state.ProvisioningStatusReady {
		t.Fatalf("expected provisioned status, got %#v", instance)
	}
	logContent, err := os.ReadFile(instance.ProvisionLogPath)
	if err != nil {
		t.Fatalf("read provision log: %v", err)
	}
	if !strings.Contains(string(logContent), "status: provisioned") {
		t.Fatalf("expected provisioned log, got:\n%s", string(logContent))
	}
}

func TestProvisionMarksFailureAndPreservesRunningInstance(t *testing.T) {
	service, root, metadata := newProvisionServiceWithRunningInstance(t)

	service.provisionTransport = provssh.FakeTransport{
		RunFunc: func(ctx context.Context, request provssh.RunRequest) (provssh.RunResult, error) {
			if strings.Contains(request.Command, "apt-get install") {
				return provssh.RunResult{Stderr: "apt failed\n", ExitCode: 100, Duration: time.Millisecond}, errors.New("ssh failed")
			}
			return provssh.RunResult{Stdout: "ok\n", ExitCode: 0, Duration: time.Millisecond}, nil
		},
	}

	_, err := service.Provision(context.Background(), ProvisionOptions{ProjectRoot: root, Target: "web"})
	if err == nil {
		t.Fatal("expected provisioning failure")
	}
	assertAppErrorCode(t, err, ErrorCodePrecondition)

	loaded, err := state.Load(filepath.Join(root, "yeast-home", "projects", metadata.ID, "state.json"), metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	instance := loaded.Instances["web"]
	if instance.Status != "running" {
		t.Fatalf("expected instance to remain running, got %#v", instance)
	}
	if instance.ProvisioningStatus != state.ProvisioningStatusFailed {
		t.Fatalf("expected failed provisioning status, got %#v", instance)
	}
	if !strings.Contains(instance.LastError, "package provisioning failed") {
		t.Fatalf("unexpected last_error: %#v", instance)
	}
}

func TestProvisionRequiresReachableRunningInstance(t *testing.T) {
	service, root, _ := newProvisionServiceWithRunningInstance(t)
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error {
		return errors.New("connection refused")
	}

	_, err := service.Provision(context.Background(), ProvisionOptions{ProjectRoot: root, Target: "web"})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func newProvisionServiceWithRunningInstance(t *testing.T) (*Service, string, project.Metadata) {
	t.Helper()

	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeRuntime{}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	siteDir := filepath.Join(root, "site")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatalf("mkdir site dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "index.html"), []byte("<h1>yeast</h1>\n"), 0644); err != nil {
		t.Fatalf("write site file: %v", err)
	}

	configContent := `version: 1
provision:
  packages:
    - curl
  files:
    - source: ./site/index.html
      destination: /home/yeast/site/index.html
      permissions: "0644"
  shell:
    - echo project
instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2205
    provision:
      packages:
        - caddy
      shell:
        - echo instance
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	runtimeDir, err := paths.InstanceDir("web")
	if err != nil {
		t.Fatalf("InstanceDir returned error: %v", err)
	}
	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{
		Status:             "running",
		PID:                os.Getpid(),
		ManagementIP:       "127.0.0.1",
		SSHPort:            2205,
		RuntimeDir:         runtimeDir,
		ProvisionLogPath:   filepath.Join(runtimeDir, "provision.log"),
		ProvisioningStatus: state.ProvisioningStatusReady,
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	return service, root, metadata
}
