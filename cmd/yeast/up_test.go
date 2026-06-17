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

func TestShouldCheckUpdateNoticeOnlyInHumanOutput(t *testing.T) {
	tests := []struct {
		name   string
		json   bool
		events bool
		quiet  bool
		want   bool
	}{
		{name: "human", want: true},
		{name: "json", json: true},
		{name: "events", events: true},
		{name: "quiet", quiet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousJSON := outputJSON
			previousEvents := outputEvents
			previousQuiet := outputQuiet
			outputJSON = tt.json
			outputEvents = tt.events
			outputQuiet = tt.quiet
			defer func() {
				outputJSON = previousJSON
				outputEvents = previousEvents
				outputQuiet = previousQuiet
			}()

			if got := shouldCheckUpdateNotice(); got != tt.want {
				t.Fatalf("shouldCheckUpdateNotice() = %v, want %v", got, tt.want)
			}
		})
	}
}
