package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	"yeast/internal/guest"
	"yeast/internal/project"
	provssh "yeast/internal/provision/ssh"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestCopyUploadsLocalFileToGuest(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	localFile := filepath.Join(root, "site.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }

	var gotRequest provssh.UploadRequest
	service.provisionTransport = provssh.FakeTransport{
		UploadFunc: func(ctx context.Context, request provssh.UploadRequest) error {
			gotRequest = request
			return nil
		},
	}

	prepareCopyProject(t, service, root, yeastHome)

	result, err := service.Copy(context.Background(), CopyOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Direction:          CopyToGuest,
		Source:             "./site.txt",
		Destination:        "/tmp/site.txt",
	})
	if err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}
	if gotRequest.Source != localFile || gotRequest.Destination != "/tmp/site.txt" {
		t.Fatalf("unexpected upload request: %#v", gotRequest)
	}
	if result.Direction != CopyToGuest || result.Source != localFile || result.Destination != "/tmp/site.txt" {
		t.Fatalf("unexpected copy result: %#v", result)
	}
}

func TestCopyDownloadsGuestFileToLocalPath(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	destination := filepath.Join(root, "artifacts", "report.txt")
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }

	var gotRequest provssh.DownloadRequest
	service.provisionTransport = provssh.FakeTransport{
		DownloadFunc: func(ctx context.Context, request provssh.DownloadRequest) error {
			gotRequest = request
			return nil
		},
	}

	prepareCopyProject(t, service, root, yeastHome)

	result, err := service.Copy(context.Background(), CopyOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Direction:          CopyFromGuest,
		Source:             "/var/tmp/report.txt",
		Destination:        "./artifacts/report.txt",
	})
	if err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}
	if gotRequest.Source != "/var/tmp/report.txt" || gotRequest.Destination != destination {
		t.Fatalf("unexpected download request: %#v", gotRequest)
	}
	if result.Direction != CopyFromGuest || result.Source != "/var/tmp/report.txt" || result.Destination != destination {
		t.Fatalf("unexpected copy result: %#v", result)
	}
}

func TestCopyRejectsMissingLocalSource(t *testing.T) {
	root := t.TempDir()

	service := NewService()

	_, err := service.Copy(context.Background(), CopyOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Direction:          CopyToGuest,
		Source:             "./missing.txt",
		Destination:        "/tmp/missing.txt",
	})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func TestCopyRejectsMissingLocalDestinationParent(t *testing.T) {
	root := t.TempDir()

	service := NewService()

	_, err := service.Copy(context.Background(), CopyOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Direction:          CopyFromGuest,
		Source:             "/tmp/report.txt",
		Destination:        "./missing/report.txt",
	})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func TestCopyRejectsInvalidDirection(t *testing.T) {
	service := NewService()

	_, err := service.Copy(context.Background(), CopyOptions{
		Direction:   CopyDirection("sideways"),
		Source:      "a",
		Destination: "b",
	})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func TestCopyClassifiesTransportFailureAsPrecondition(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	localFile := filepath.Join(root, "site.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }
	service.provisionTransport = provssh.FakeTransport{
		UploadFunc: func(ctx context.Context, request provssh.UploadRequest) error {
			return errors.New("scp failed")
		},
	}

	prepareCopyProject(t, service, root, yeastHome)

	_, err := service.Copy(context.Background(), CopyOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Direction:          CopyToGuest,
		Source:             "./site.txt",
		Destination:        "/tmp/site.txt",
		Timeout:            time.Second,
	})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func prepareCopyProject(t *testing.T, service *Service, root, yeastHome string) {
	t.Helper()

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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2205}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
}
