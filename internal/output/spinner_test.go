package output

import (
	"bytes"
	"strings"
	"testing"

	"yeast/internal/app"
)

func TestSpinnerStartStop(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf)
	s.Start("loading")
	// Let a few frames tick.
	s.Stop("\033[32m✓\033[0m", "done")
	out := buf.String()
	if !strings.Contains(out, "done") {
		t.Errorf("expected 'done' in output, got: %q", out)
	}
	if !strings.Contains(out, "\r\033[K") {
		t.Errorf("expected ANSI clear sequence in output, got: %q", out)
	}
}

func TestSpinnerUpdate(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf)
	s.Start("step 1")
	s.Update("step 2")
	s.Stop("\033[32m✓\033[0m", "")
	out := buf.String()
	if !strings.Contains(out, "step 2") {
		t.Errorf("expected 'step 2' in output, got: %q", out)
	}
}

func TestSpinnerPrintLine(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf)
	s.PrintLine("\033[32m✓\033[0m", "finished")
	out := buf.String()
	if out != "  \033[32m✓\033[0m finished\n" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestSpinnerStopEmptyMessageUsesLast(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf)
	s.Start("my message")
	s.Stop("+", "")
	out := buf.String()
	if !strings.Contains(out, "my message") {
		t.Errorf("expected 'my message' in output, got: %q", out)
	}
}

func TestFormatProgressEvent(t *testing.T) {
	tests := []struct {
		name  string
		event app.Event
		want  string
	}{
		{
			name:  "silent event",
			event: app.Event{Name: app.EventProjectLoaded},
			want:  "",
		},
		{
			name:  "image pulling",
			event: app.Event{Name: app.EventImagePulling},
			want:  "  * Pulling image...",
		},
		{
			name:  "instance ready",
			event: app.Event{Name: app.EventInstanceReady, Instance: "web"},
			want:  "  + [web] Ready",
		},
		{
			name:  "workflow failed with message",
			event: app.Event{Name: app.EventWorkflowFailed, Message: "boot timed out"},
			want:  "  ! boot timed out",
		},
		{
			name:  "unknown event with message",
			event: app.Event{Name: "custom", Message: "something happened"},
			want:  "  * something happened",
		},
		{
			name:  "unknown event without message",
			event: app.Event{Name: "custom"},
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatProgressEvent(tt.event, false)
			if got != tt.want {
				t.Errorf("formatProgressEvent() = %q, want %q", got, tt.want)
			}
		})
	}
}
