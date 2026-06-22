package main

import (
	"bytes"
	"strings"
	"testing"
	"yeast/internal/app"
)

func TestRenderExecCommandOutputUsesRawStreamsInHumanMode(t *testing.T) {
	previous := outputJSON
	outputJSON = false
	defer func() {
		outputJSON = previous
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	result := app.ExecResult{
		Instance: "web",
		Run: app.GuestCommandResult{
			Command:  "hostname",
			ExitCode: 0,
			Stdout:   "cloudinit-lab\n",
			Stderr:   "remote warning\n",
		},
	}

	if err := renderExecCommandOutput(&stdout, &stderr, result); err != nil {
		t.Fatalf("renderExecCommandOutput returned error: %v", err)
	}
	if got := stdout.String(); got != "cloudinit-lab\n" {
		t.Fatalf("expected raw stdout, got %q", got)
	}
	if got := stderr.String(); got != "remote warning\n" {
		t.Fatalf("expected raw stderr, got %q", got)
	}
}

func TestRenderExecCommandOutputKeepsJSONEnvelopeInJSONMode(t *testing.T) {
	previous := outputJSON
	outputJSON = true
	defer func() {
		outputJSON = previous
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	result := app.ExecResult{
		Instance: "web",
		Run: app.GuestCommandResult{
			Command:  "whoami",
			ExitCode: 0,
			Stdout:   "yeast\n",
		},
	}

	if err := renderExecCommandOutput(&stdout, &stderr, result); err != nil {
		t.Fatalf("renderExecCommandOutput returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{`"command":"exec"`, `"stdout":"yeast\n"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected JSON output to contain %s, got:\n%s", want, got)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected JSON mode not to write stderr, got %q", stderr.String())
	}
}
