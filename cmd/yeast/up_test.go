package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUpNoProvisionAndReprovisionConflict(t *testing.T) {
	cmd := newUpCmd(nil)
	cmd.SetArgs([]string{"--no-provision", "--reprovision"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when using --no-provision and --reprovision together")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("expected conflict message, got: %v", err)
	}
}
