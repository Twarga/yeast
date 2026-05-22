package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"yeast/internal/guest"
	"yeast/internal/project"
	"yeast/internal/provision/cloudinit"
	provssh "yeast/internal/provision/ssh"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestUpStartsInstanceAndSavesState(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAATEST", nil }
	service.managementPortAvailable = func(port int) bool { return true }

	var userDataCalls int
	service.renderUserData = func(input cloudinit.UserDataInput) (string, error) {
		userDataCalls++
		if input.Hostname != "web-lab" {
			t.Fatalf("unexpected hostname: %q", input.Hostname)
		}
		return "#cloud-config\nhostname: web-lab\n", nil
	}
	service.renderMetaData = func(input cloudinit.MetaDataInput) (string, error) {
		if input.Hostname != "web-lab" {
			t.Fatalf("unexpected meta-data hostname: %q", input.Hostname)
		}
		return "instance-id: web-lab\nlocal-hostname: web-lab\n", nil
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
		if options.Address != "127.0.0.1:2205" {
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
    hostname: web-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 25 gb
    ssh_port: 2205
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
	if result.Instances[0].SSHAddress != "127.0.0.1:2205" {
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
	for _, want := range []string{`"status": "running"`, `"ssh_port": 2205`, `"runtime_dir": "`, `"provisioning_status": "provisioned"`, `"provision_log_path": "`} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected state content %q, got:\n%s", want, content)
		}
	}
}

func TestUpBuildsLabNetworkPlanAndSeedConfig(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAATEST", nil }
	service.managementPortAvailable = func(port int) bool { return true }
	service.renderUserData = func(input cloudinit.UserDataInput) (string, error) {
		return "#cloud-config\nhostname: web-lab\n", nil
	}
	service.renderMetaData = func(input cloudinit.MetaDataInput) (string, error) {
		return "instance-id: web-lab\nlocal-hostname: web-lab\n", nil
	}

	var gotNetworkConfig cloudinit.NetworkConfigInput
	service.renderNetworkConfig = func(input cloudinit.NetworkConfigInput) (string, error) {
		gotNetworkConfig = input
		return "version: 2\nethernets: {}\n", nil
	}

	var gotSeedInput cloudinit.SeedInput
	service.createSeedISO = func(ctx context.Context, input cloudinit.SeedInput) (cloudinit.SeedResult, error) {
		gotSeedInput = input
		return cloudinit.SeedResult{
			UserDataPath:      filepath.Join(input.RuntimeDir, "user-data"),
			MetaDataPath:      filepath.Join(input.RuntimeDir, "meta-data"),
			NetworkConfigPath: filepath.Join(input.RuntimeDir, "network-config"),
			ISOPath:           filepath.Join(input.RuntimeDir, "seed.iso"),
			Builder:           "genisoimage",
		}, nil
	}

	fakeRuntime := &fakeRuntime{}
	service.runtime = fakeRuntime
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }

	if _, err := service.Init(InitOptions{ProjectRoot: root, Now: time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	configPath := filepath.Join(root, ConfigFileName)
	configContent := `version: 1
networks:
  - name: lab
    cidr: 10.10.10.0/24
instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    ssh_port: 2205
    networks:
      - name: lab
        ipv4: 10.10.10.10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	imagePath := filepath.Join(yeastHome, "cache", "images", "ubuntu-24.04", "image.qcow2")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		t.Fatalf("create image cache dir: %v", err)
	}
	if err := os.WriteFile(imagePath, []byte("image"), 0644); err != nil {
		t.Fatalf("write cached image: %v", err)
	}

	if _, err := service.Up(context.Background(), UpOptions{ProjectRoot: root, ReadinessTimeout: 5 * time.Second}); err != nil {
		t.Fatalf("Up returned error: %v", err)
	}

	if gotNetworkConfig.InterfaceName != "yeastlab0" {
		t.Fatalf("unexpected lab interface name: %q", gotNetworkConfig.InterfaceName)
	}
	if gotNetworkConfig.IPv4.String() != "10.10.10.10" {
		t.Fatalf("unexpected lab ipv4: %s", gotNetworkConfig.IPv4)
	}
	if gotNetworkConfig.CIDR.String() != "10.10.10.0/24" {
		t.Fatalf("unexpected lab cidr: %s", gotNetworkConfig.CIDR)
	}
	if gotNetworkConfig.MACAddress == "" {
		t.Fatal("expected lab mac address")
	}
	if gotSeedInput.NetworkConfig == "" {
		t.Fatal("expected seed input network-config content")
	}
	if fakeRuntime.startPlan.Networks.Lab == nil {
		t.Fatal("expected lab network plan on runtime start")
	}
	if fakeRuntime.startPlan.Networks.Lab.InterfaceName != gotNetworkConfig.InterfaceName {
		t.Fatalf("runtime/network-config interface mismatch: %q vs %q", fakeRuntime.startPlan.Networks.Lab.InterfaceName, gotNetworkConfig.InterfaceName)
	}
	if fakeRuntime.startPlan.Networks.Lab.MACAddress != gotNetworkConfig.MACAddress {
		t.Fatalf("runtime/network-config mac mismatch: %q vs %q", fakeRuntime.startPlan.Networks.Lab.MACAddress, gotNetworkConfig.MACAddress)
	}
}

func TestUpRunsProvisioningInDocumentedOrder(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.runtime = &fakeRuntime{}

	siteDir := filepath.Join(root, "site")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatalf("mkdir site dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "index.html"), []byte("<h1>yeast</h1>\n"), 0644); err != nil {
		t.Fatalf("write site file: %v", err)
	}

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

	result, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Up returned error: %v", err)
	}
	if len(result.Instances) != 1 {
		t.Fatalf("expected 1 instance result, got %d", len(result.Instances))
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
	if !reflect.DeepEqual(gotCommands, wantCommands) {
		t.Fatalf("unexpected provisioning command order:\n got: %#v\nwant: %#v", gotCommands, wantCommands)
	}

	if len(uploadRequests) != 1 {
		t.Fatalf("expected 1 upload request, got %d", len(uploadRequests))
	}
	if uploadRequests[0].Source != filepath.Join(root, "site", "index.html") {
		t.Fatalf("unexpected upload source: %q", uploadRequests[0].Source)
	}
	if uploadRequests[0].Destination != "/home/yeast/site/index.html" {
		t.Fatalf("unexpected upload destination: %q", uploadRequests[0].Destination)
	}

	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
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
	logText := string(logContent)
	for _, want := range []string{"status: provisioned", "[package]", "[file]", "[shell] echo project", "[shell] echo instance"} {
		if !strings.Contains(logText, want) {
			t.Fatalf("expected provision log to contain %q, got:\n%s", want, logText)
		}
	}
}

func TestUpPreservesSnapshotsAcrossRestart(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.runtime = &fakeRuntime{}

	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
	}
	paths, err := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	runtimeDir, err := paths.InstanceDir("web")
	if err != nil {
		t.Fatalf("InstanceDir returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(runtimeDir, "snapshots"), 0755); err != nil {
		t.Fatalf("mkdir snapshots dir: %v", err)
	}

	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{
		Status:     "stopped",
		RuntimeDir: runtimeDir,
		Snapshots: map[string]state.SnapshotState{
			"clean": {
				Name:        "clean",
				CreatedAt:   time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC),
				Description: "baseline",
				DiskPath:    filepath.Join(runtimeDir, "snapshots", "clean.qcow2"),
			},
		},
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if _, err := service.Up(context.Background(), UpOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Up returned error: %v", err)
	}

	loaded, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if _, ok := loaded.Instances["web"].Snapshots["clean"]; !ok {
		t.Fatalf("expected snapshot metadata to survive restart, got %#v", loaded.Instances["web"])
	}
}

func TestUpMarksProvisioningFailureAndKeepsInstanceRunning(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	fake := &fakeRuntime{}
	service.runtime = fake

	runCalls := 0
	service.provisionTransport = provssh.FakeTransport{
		RunFunc: func(ctx context.Context, request provssh.RunRequest) (provssh.RunResult, error) {
			runCalls++
			if strings.Contains(request.Command, "apt-get install") {
				return provssh.RunResult{Stderr: "apt failed\n", ExitCode: 100, Duration: time.Millisecond}, errors.New("ssh failed")
			}
			return provssh.RunResult{Stdout: "ok\n", ExitCode: 0, Duration: time.Millisecond}, nil
		},
	}

	configContent := `version: 1
provision:
  packages:
    - curl
instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2205
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected provisioning failure")
	}
	assertAppErrorCode(t, err, ErrorCodePrecondition)
	if fake.stopCalls != 0 {
		t.Fatalf("expected instance to remain running after provisioning failure, got %d stop calls", fake.stopCalls)
	}
	if runCalls != 2 {
		t.Fatalf("expected bootstrap wait plus one provisioning run call, got %d", runCalls)
	}

	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
	}
	loaded, err := state.Load(filepath.Join(root, "yeast-home", "projects", metadata.ID, "state.json"), metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	instance := loaded.Instances["web"]
	if instance.Status != "running" {
		t.Fatalf("expected running status after provisioning failure, got %#v", instance)
	}
	if instance.ProvisioningStatus != state.ProvisioningStatusFailed {
		t.Fatalf("expected failed provisioning status, got %#v", instance)
	}
	if !strings.Contains(instance.LastError, "package provisioning failed") {
		t.Fatalf("unexpected last_error: %#v", instance)
	}
	logContent, err := os.ReadFile(instance.ProvisionLogPath)
	if err != nil {
		t.Fatalf("read provision log: %v", err)
	}
	if !strings.Contains(string(logContent), "status: failed") {
		t.Fatalf("expected failed provision log, got:\n%s", string(logContent))
	}
}

func TestUpRetriesBootstrapSSHReadinessWithinTimeout(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.runtime = &fakeRuntime{}
	service.sleep = func(time.Duration) {}

	runCalls := 0
	service.provisionTransport = provssh.FakeTransport{
		RunFunc: func(ctx context.Context, request provssh.RunRequest) (provssh.RunResult, error) {
			runCalls++
			switch runCalls {
			case 1:
				return provssh.RunResult{ExitCode: 255, Duration: time.Millisecond}, errors.New("ssh failed")
			case 2:
				return provssh.RunResult{Stdout: "done\n", ExitCode: 0, Duration: time.Millisecond}, nil
			default:
				return provssh.RunResult{Stdout: "ok\n", ExitCode: 0, Duration: time.Millisecond}, nil
			}
		},
	}

	configContent := `version: 1
provision:
  shell:
    - echo ready
instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2205
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := service.Up(context.Background(), UpOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Up returned error: %v", err)
	}
	if runCalls != 3 {
		t.Fatalf("expected bootstrap retry plus shell command, got %d run calls", runCalls)
	}
}

func TestUpClassifiesExternallyBoundRequestedSSHPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on test port: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	service, root := newUpServiceWithCachedImage(t)
	service.runtime = &fakeRuntime{
		startHook: func(plan rtm.MachinePlan) error {
			return os.WriteFile(plan.LogPath, []byte(fmt.Sprintf(
				"qemu-system-x86_64: -netdev user,id=mgmt0,hostfwd=tcp:127.0.0.1:%d-:22: Could not set up host forwarding rule 'tcp:127.0.0.1:%d-:22'\n",
				port,
				port,
			)), 0644)
		},
	}
	service.sleep = func(time.Duration) {}

	configContent := `version: 1
instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: %d
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(fmt.Sprintf(configContent, port)), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err = service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
	if !strings.Contains(err.Error(), "already bound on the host") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpRetriesHostForwardConflictBeforeFailing(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	fake := &fakeRuntime{}
	service.runtime = fake
	service.sleep = func(time.Duration) {}
	service.managementPortAvailable = func(port int) bool { return true }

	startAttempts := 0
	fake.startHook = func(plan rtm.MachinePlan) error {
		startAttempts++
		if startAttempts < 3 {
			return os.WriteFile(plan.LogPath, []byte(fmt.Sprintf(
				"qemu-system-x86_64: -netdev user,id=mgmt0,hostfwd=tcp:127.0.0.1:%d-:22: Could not set up host forwarding rule 'tcp:127.0.0.1:%d-:22'\n",
				plan.Networks.Management.SSHPort,
				plan.Networks.Management.SSHPort,
			)), 0644)
		}
		return os.WriteFile(plan.LogPath, []byte("booted cleanly\n"), 0644)
	}

	result, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Up returned error: %v", err)
	}
	if len(result.Instances) != 1 {
		t.Fatalf("expected one instance, got %#v", result.Instances)
	}
	if startAttempts != 3 {
		t.Fatalf("expected 3 start attempts, got %d", startAttempts)
	}
	if fake.stopCalls != 2 {
		t.Fatalf("expected 2 stop calls for conflict retries, got %d", fake.stopCalls)
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

func TestUpClassifiesRequestedSSHPortCollision(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.runtime = &fakeRuntime{}

	configContent := `version: 1
instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2222
  - name: api
    image: ubuntu-24.04
    ssh_port: 2222
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
	if !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpClassifiesCachedRunningSSHAddressFailure(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.sshAddress = func(host string, port int) (string, error) {
		return "", errors.New("bad ssh address")
	}

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{Status: "running", PID: os.Getpid(), SSHPort: 2222}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestUpClassifiesMissingSSHPublicKey(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.discoverSSHKey = func() (string, error) {
		return "", fmt.Errorf("%w: checked ~/.ssh/id_ed25519.pub and ~/.ssh/id_rsa.pub", cloudinit.ErrNoSSHPublicKey)
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestUpClassifiesUserDataRenderFailure(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.renderUserData = func(input cloudinit.UserDataInput) (string, error) {
		return "", errors.New("render user-data failed")
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestUpClassifiesMetaDataRenderFailure(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.renderMetaData = func(input cloudinit.MetaDataInput) (string, error) {
		return "", errors.New("render meta-data failed")
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestUpClassifiesSeedISOCreationFailure(t *testing.T) {
	service, root := newUpServiceWithCachedImage(t)
	service.createSeedISO = func(ctx context.Context, input cloudinit.SeedInput) (cloudinit.SeedResult, error) {
		return cloudinit.SeedResult{}, errors.New("seed iso failed")
	}

	_, err := service.Up(context.Background(), UpOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
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
	service.managementPortAvailable = func(port int) bool { return true }

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
	startHook   func(plan rtm.MachinePlan) error
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
	if f.startHook != nil {
		if err := os.MkdirAll(filepath.Dir(plan.LogPath), 0755); err != nil {
			return rtm.RuntimeInstance{}, err
		}
		if err := f.startHook(plan); err != nil {
			return rtm.RuntimeInstance{}, err
		}
	}
	return rtm.RuntimeInstance{
		Name:       plan.Name,
		RuntimeDir: plan.RuntimeDir,
		LogPath:    plan.LogPath,
		PID:        4242,
		Networks:   plan.Networks,
		StartedAt:  time.Now().UTC(),
	}, nil
}

func (f *fakeRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	f.stopCalls++
	return nil
}

func (f *fakeRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{PID: instance.PID, State: rtm.ProcessStateRunning}, nil
}

func (f *fakeRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeRuntime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	return nil
}

func (f *fakeRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}
