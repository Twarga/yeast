package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestInspectReturnsInstanceDetailsAndSnapshots(t *testing.T) {
	service, root, metadata, runtimeDir := newInspectLogsServiceWithState(t)

	result, err := service.Inspect(context.Background(), InspectOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if result.ProjectID != metadata.ID {
		t.Fatalf("unexpected project id %q", result.ProjectID)
	}
	if result.Instance.Name != "web" || result.Instance.RuntimeDir != runtimeDir {
		t.Fatalf("unexpected inspect instance: %#v", result.Instance)
	}
	if result.SnapshotCount != 2 {
		t.Fatalf("unexpected snapshot count: %d", result.SnapshotCount)
	}
	if len(result.SnapshotNames) != 2 || result.SnapshotNames[0] != "clean" || result.SnapshotNames[1] != "late" {
		t.Fatalf("unexpected snapshot names: %#v", result.SnapshotNames)
	}
}

func TestInspectMissingInstanceReturnsNotFound(t *testing.T) {
	service, root, _, _ := newInspectLogsServiceWithState(t)

	_, err := service.Inspect(context.Background(), InspectOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "api"},
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

func TestLogsReturnsRuntimeLogContent(t *testing.T) {
	service, root, metadata, runtimeDir := newInspectLogsServiceWithState(t)
	logPath := filepath.Join(runtimeDir, "vm.log")
	if err := os.WriteFile(logPath, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := service.Logs(context.Background(), LogsOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		TailLines:          2,
	})
	if err != nil {
		t.Fatalf("Logs returned error: %v", err)
	}
	if result.ProjectID != metadata.ID {
		t.Fatalf("unexpected project id %q", result.ProjectID)
	}
	if result.LogPath != logPath {
		t.Fatalf("unexpected log path %q", result.LogPath)
	}
	if result.Content != "line2\nline3" {
		t.Fatalf("unexpected log content %q", result.Content)
	}
}

func TestLogsMissingRuntimeLogReturnsNotFound(t *testing.T) {
	service, root, _, _ := newInspectLogsServiceWithState(t)

	_, err := service.Logs(context.Background(), LogsOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

func TestLogsRequiresRuntimeDir(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{}

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
	current.Instances["web"] = state.InstanceState{Status: "stopped"}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Logs(context.Background(), LogsOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
	})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestTailLogContentReturnsRequestedTail(t *testing.T) {
	t.Parallel()

	got := tailLogContent("a\nb\nc\n", 2)
	if got != "b\nc" {
		t.Fatalf("unexpected tailed content %q", got)
	}
}

func newInspectLogsServiceWithState(t *testing.T) (*Service, string, project.Metadata, string) {
	t.Helper()

	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}

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
	runtimeDir, err := paths.InstanceDir("web")
	if err != nil {
		t.Fatalf("InstanceDir returned error: %v", err)
	}
	snapshotsDir := filepath.Join(runtimeDir, "snapshots")
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{
		Status:             "running",
		PID:                4242,
		SSHPort:            2205,
		ManagementIP:       "127.0.0.1",
		LabIP:              "10.10.10.20",
		RuntimeDir:         runtimeDir,
		ProvisionLogPath:   filepath.Join(runtimeDir, "provision.log"),
		ProvisioningStatus: state.ProvisioningStatusReady,
		LastError:          "",
		Snapshots: map[string]state.SnapshotState{
			"late": {
				Name:      "late",
				CreatedAt: time.Date(2026, 5, 19, 13, 0, 0, 0, time.UTC),
				DiskPath:  filepath.Join(snapshotsDir, "late.qcow2"),
			},
			"clean": {
				Name:      "clean",
				CreatedAt: time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
				DiskPath:  filepath.Join(snapshotsDir, "clean.qcow2"),
			},
		},
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	return service, root, metadata, runtimeDir
}
