package ssh

import (
	"context"
	"errors"
	"testing"
	"time"
	"yeast/internal/provision"
)

func TestShellProvisionerRunNoopOnEmptyPlan(t *testing.T) {
	transport := &scriptedTransport{}
	provisioner := NewShellProvisioner(transport)

	result, err := provisioner.Run(context.Background(), ShellRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Steps) != 0 {
		t.Fatalf("expected empty shell result, got %#v", result)
	}
	if len(transport.runRequests) != 0 {
		t.Fatal("transport should not run for empty shell plans")
	}
}

func TestShellProvisionerRunExecutesCommandsInOrder(t *testing.T) {
	transport := &scriptedTransport{
		runResults: []RunResult{
			{Stdout: "first\n", ExitCode: 0, Duration: time.Millisecond},
			{Stdout: "second\n", ExitCode: 0, Duration: 2 * time.Millisecond},
		},
	}
	provisioner := NewShellProvisioner(transport)

	result, err := provisioner.Run(context.Background(), ShellRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Commands: []provision.ShellStep{
			{Command: "echo first"},
			{Command: "echo second"},
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(transport.runRequests) != 2 {
		t.Fatalf("expected 2 run requests, got %d", len(transport.runRequests))
	}
	if transport.runRequests[0].Command != "echo first" || transport.runRequests[1].Command != "echo second" {
		t.Fatalf("unexpected command order: %#v", transport.runRequests)
	}
	if transport.runRequests[0].Timeout != DefaultShellTimeout {
		t.Fatalf("expected default shell timeout %s, got %s", DefaultShellTimeout, transport.runRequests[0].Timeout)
	}
	if len(result.Steps) != 2 || result.Steps[1].Run.Stdout != "second\n" {
		t.Fatalf("unexpected shell result: %#v", result)
	}
}

func TestShellProvisionerRunStopsOnFailure(t *testing.T) {
	transport := &scriptedTransport{
		runResults: []RunResult{
			{Stdout: "first\n", ExitCode: 0},
			{Stderr: "boom\n", ExitCode: 9},
		},
		runErrors: []error{
			nil,
			errors.New("ssh failed"),
		},
	}
	provisioner := NewShellProvisioner(transport)

	result, err := provisioner.Run(context.Background(), ShellRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Commands: []provision.ShellStep{
			{Command: "echo first"},
			{Command: "false"},
			{Command: "echo third"},
		},
	})
	if err == nil {
		t.Fatal("expected shell failure")
	}
	if len(transport.runRequests) != 2 {
		t.Fatalf("expected stop after second command, got %d requests", len(transport.runRequests))
	}
	if len(result.Steps) != 2 || result.Steps[1].Run.ExitCode != 9 {
		t.Fatalf("expected failed command result to be preserved, got %#v", result)
	}
}

func TestShellProvisionerRunRejectsInvalidRequest(t *testing.T) {
	provisioner := NewShellProvisioner(&scriptedTransport{})

	_, err := provisioner.Run(context.Background(), ShellRequest{
		User: "",
		Host: "127.0.0.1",
		Port: 2205,
		Commands: []provision.ShellStep{
			{Command: "echo ok"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid request error")
	}
}

func TestShellProvisionerRunRejectsEmptyCommand(t *testing.T) {
	provisioner := NewShellProvisioner(&scriptedTransport{})

	_, err := provisioner.Run(context.Background(), ShellRequest{
		User: "yeast",
		Host: "127.0.0.1",
		Port: 2205,
		Commands: []provision.ShellStep{
			{Command: "   "},
		},
	})
	if err == nil {
		t.Fatal("expected empty command error")
	}
}
