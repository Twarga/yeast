package ssh

import (
	"context"
	"errors"
	"testing"
	"time"
	"yeast/internal/provision"
)

type scriptedTransport struct {
	runRequests    []RunRequest
	uploadRequests []UploadRequest
	runResults     []RunResult
	runErrors      []error
	uploadErrors   []error
}

func (s *scriptedTransport) Run(ctx context.Context, request RunRequest) (RunResult, error) {
	s.runRequests = append(s.runRequests, request)
	index := len(s.runRequests) - 1
	var result RunResult
	var err error
	if index < len(s.runResults) {
		result = s.runResults[index]
	}
	if index < len(s.runErrors) {
		err = s.runErrors[index]
	}
	return result, err
}

func (s *scriptedTransport) Upload(ctx context.Context, request UploadRequest) error {
	s.uploadRequests = append(s.uploadRequests, request)
	index := len(s.uploadRequests) - 1
	if index < len(s.uploadErrors) {
		return s.uploadErrors[index]
	}
	return nil
}

func TestFileProvisionerUploadNoopOnEmptyPlan(t *testing.T) {
	transport := &scriptedTransport{}
	provisioner := NewFileProvisioner(transport)

	result, err := provisioner.Upload(context.Background(), FileRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
	})
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected empty file result, got %#v", result)
	}
	if len(transport.runRequests) != 0 || len(transport.uploadRequests) != 0 {
		t.Fatal("transport should not be used for empty plans")
	}
}

func TestFileProvisionerUploadRunsMkdirUploadAndChmod(t *testing.T) {
	transport := &scriptedTransport{
		runResults: []RunResult{
			{ExitCode: 0, Duration: time.Millisecond},
			{ExitCode: 0, Duration: 2 * time.Millisecond},
		},
	}
	provisioner := NewFileProvisioner(transport)

	result, err := provisioner.Upload(context.Background(), FileRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Files: []provision.FileStep{
			{
				Source:      "./site/index.html",
				Destination: "/srv/site/index.html",
				Permissions: "0644",
			},
		},
	})
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	if len(transport.runRequests) != 2 {
		t.Fatalf("expected 2 run requests, got %d", len(transport.runRequests))
	}
	if transport.runRequests[0].Command != "mkdir -p '/srv/site'" {
		t.Fatalf("unexpected mkdir command: %q", transport.runRequests[0].Command)
	}
	if transport.runRequests[1].Command != "chmod 0644 '/srv/site/index.html'" {
		t.Fatalf("unexpected chmod command: %q", transport.runRequests[1].Command)
	}

	if len(transport.uploadRequests) != 1 {
		t.Fatalf("expected 1 upload request, got %d", len(transport.uploadRequests))
	}
	if transport.uploadRequests[0].Destination != "/srv/site/index.html" {
		t.Fatalf("unexpected upload destination: %q", transport.uploadRequests[0].Destination)
	}
	if transport.uploadRequests[0].Timeout != DefaultFileTimeout {
		t.Fatalf("expected default file timeout %s, got %s", DefaultFileTimeout, transport.uploadRequests[0].Timeout)
	}

	if len(result.Files) != 1 || !result.Files[0].Upload {
		t.Fatalf("unexpected file result: %#v", result)
	}
}

func TestFileProvisionerUploadSkipsChmodWhenPermissionsEmpty(t *testing.T) {
	transport := &scriptedTransport{
		runResults: []RunResult{{ExitCode: 0}},
	}
	provisioner := NewFileProvisioner(transport)

	result, err := provisioner.Upload(context.Background(), FileRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Files: []provision.FileStep{
			{
				Source:      "./site",
				Destination: "/srv/site",
			},
		},
	})
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if len(transport.runRequests) != 1 {
		t.Fatalf("expected only mkdir request, got %d", len(transport.runRequests))
	}
	if result.Files[0].Chmod.ExitCode != 0 || result.Files[0].Permissions != "" {
		t.Fatalf("unexpected chmod result for no-permissions file: %#v", result.Files[0])
	}
}

func TestFileProvisionerUploadReturnsContextOnMkdirFailure(t *testing.T) {
	transport := &scriptedTransport{
		runResults: []RunResult{{Stderr: "mkdir failed\n", ExitCode: 1}},
		runErrors:  []error{errors.New("ssh failed")},
	}
	provisioner := NewFileProvisioner(transport)

	result, err := provisioner.Upload(context.Background(), FileRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Files: []provision.FileStep{
			{Source: "./site", Destination: "/srv/site"},
		},
	})
	if err == nil {
		t.Fatal("expected mkdir failure")
	}
	if len(result.Files) != 1 || result.Files[0].Mkdir.Stderr != "mkdir failed\n" {
		t.Fatalf("expected mkdir failure result to be preserved, got %#v", result)
	}
}

func TestFileProvisionerUploadReturnsUploadFailureContext(t *testing.T) {
	transport := &scriptedTransport{
		runResults:   []RunResult{{ExitCode: 0}},
		uploadErrors: []error{errors.New("scp failed")},
	}
	provisioner := NewFileProvisioner(transport)

	_, err := provisioner.Upload(context.Background(), FileRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Files: []provision.FileStep{
			{Source: "./site", Destination: "/srv/site"},
		},
	})
	if err == nil {
		t.Fatal("expected upload failure")
	}
}

func TestFileProvisionerUploadReturnsChmodFailureContext(t *testing.T) {
	transport := &scriptedTransport{
		runResults: []RunResult{
			{ExitCode: 0},
			{Stderr: "chmod failed\n", ExitCode: 1},
		},
		runErrors: []error{
			nil,
			errors.New("ssh failed"),
		},
	}
	provisioner := NewFileProvisioner(transport)

	result, err := provisioner.Upload(context.Background(), FileRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Files: []provision.FileStep{
			{Source: "./site", Destination: "/srv/site", Permissions: "0755"},
		},
	})
	if err == nil {
		t.Fatal("expected chmod failure")
	}
	if len(result.Files) != 1 || result.Files[0].Chmod.Stderr != "chmod failed\n" {
		t.Fatalf("expected chmod failure result to be preserved, got %#v", result)
	}
}

func TestFileProvisionerUploadRejectsInvalidRequest(t *testing.T) {
	provisioner := NewFileProvisioner(&scriptedTransport{})

	_, err := provisioner.Upload(context.Background(), FileRequest{
		User: "yeast",
		Host: "",
		Port: 2205,
		Files: []provision.FileStep{
			{Source: "./site", Destination: "/srv/site"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid request error")
	}
}

func TestFileProvisionerUploadRejectsEmptyFileFields(t *testing.T) {
	provisioner := NewFileProvisioner(&scriptedTransport{})

	_, err := provisioner.Upload(context.Background(), FileRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Files: []provision.FileStep{
			{Source: "", Destination: "/srv/site"},
		},
	})
	if err == nil {
		t.Fatal("expected empty source error")
	}
}
