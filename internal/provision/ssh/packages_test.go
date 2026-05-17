package ssh

import (
	"context"
	"errors"
	"testing"
	"time"
	"yeast/internal/provision"
)

func TestPackageProvisionerInstallNoopOnEmptyPlan(t *testing.T) {
	transport := FakeTransport{
		RunFunc: func(ctx context.Context, request RunRequest) (RunResult, error) {
			t.Fatal("Run should not be called for empty package plans")
			return RunResult{}, nil
		},
	}
	provisioner := NewPackageProvisioner(transport)

	result, err := provisioner.Install(context.Background(), PackageRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
	})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if len(result.Packages) != 0 || result.Command != "" {
		t.Fatalf("expected empty package result, got %#v", result)
	}
}

func TestPackageProvisionerInstallBuildsAPTCommand(t *testing.T) {
	var seen RunRequest
	transport := FakeTransport{
		RunFunc: func(ctx context.Context, request RunRequest) (RunResult, error) {
			seen = request
			return RunResult{
				Stdout:   "installed\n",
				ExitCode: 0,
				Duration: time.Second,
			}, nil
		},
	}
	provisioner := NewPackageProvisioner(transport)

	result, err := provisioner.Install(context.Background(), PackageRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Packages: []provision.PackageStep{
			{Name: "caddy"},
			{Name: "curl"},
		},
	})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	wantCommand := "sudo DEBIAN_FRONTEND=noninteractive apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y caddy curl"
	if seen.Command != wantCommand {
		t.Fatalf("unexpected apt command:\n got: %q\nwant: %q", seen.Command, wantCommand)
	}
	if seen.Timeout != DefaultPackageTimeout {
		t.Fatalf("expected default package timeout %s, got %s", DefaultPackageTimeout, seen.Timeout)
	}
	if result.Command != wantCommand || result.Run.Stdout != "installed\n" {
		t.Fatalf("unexpected package result: %#v", result)
	}
}

func TestPackageProvisionerInstallPreservesTransportFailureResult(t *testing.T) {
	transport := FakeTransport{
		RunFunc: func(ctx context.Context, request RunRequest) (RunResult, error) {
			return RunResult{
				Stdout:   "partial\n",
				Stderr:   "apt failed\n",
				ExitCode: 100,
				Duration: 2 * time.Second,
			}, errors.New("ssh failed")
		},
	}
	provisioner := NewPackageProvisioner(transport)

	result, err := provisioner.Install(context.Background(), PackageRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Packages: []provision.PackageStep{
			{Name: "caddy"},
		},
	})
	if err == nil {
		t.Fatal("expected install error")
	}
	if result.Run.ExitCode != 100 || result.Run.Stderr != "apt failed\n" {
		t.Fatalf("expected failed run result to be preserved, got %#v", result)
	}
}

func TestPackageProvisionerInstallRejectsInvalidPackageName(t *testing.T) {
	provisioner := NewPackageProvisioner(FakeTransport{})

	_, err := provisioner.Install(context.Background(), PackageRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Packages: []provision.PackageStep{
			{Name: "bad package"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid package name error")
	}
}

func TestPackageProvisionerInstallRejectsMissingConnectionFields(t *testing.T) {
	provisioner := NewPackageProvisioner(FakeTransport{})

	_, err := provisioner.Install(context.Background(), PackageRequest{
		Host: "127.0.0.1",
		Port: 2205,
		Packages: []provision.PackageStep{
			{Name: "caddy"},
		},
	})
	if err == nil {
		t.Fatal("expected missing user validation error")
	}
}
