package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestStylingEnabledNonTTY(t *testing.T) {
	var buf bytes.Buffer
	// A bytes.Buffer is not a TTY, so styling should be disabled.
	if StylingEnabled(&buf) {
		t.Error("expected StylingEnabled to return false for non-TTY writer")
	}
}

func TestStylingEnabledNOCOLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	if StylingEnabled(&buf) {
		t.Error("expected StylingEnabled to return false when NO_COLOR is set")
	}
}

func TestStylingEnabledDumbTerm(t *testing.T) {
	t.Setenv("TERM", "dumb")
	var buf bytes.Buffer
	if StylingEnabled(&buf) {
		t.Error("expected StylingEnabled to return false when TERM=dumb")
	}
}

func TestShouldAnimateCI(t *testing.T) {
	t.Setenv("CI", "true")
	var buf bytes.Buffer
	if ShouldAnimate(&buf) {
		t.Error("expected ShouldAnimate to return false when CI=true")
	}
}

func TestShouldAnimateDisableEnv(t *testing.T) {
	t.Setenv("YEAST_ANIMATION", "0")
	var buf bytes.Buffer
	if ShouldAnimate(&buf) {
		t.Error("expected ShouldAnimate to return false when YEAST_ANIMATION=0")
	}
}

func TestIsCIDetection(t *testing.T) {
	t.Setenv("CI", "true")
	if !IsCI() {
		t.Error("expected IsCI to return true when CI=true")
	}
	t.Setenv("CI", "false")
	if IsCI() {
		t.Error("expected IsCI to return false when CI=false")
	}
}

func TestIsDumbTerminal(t *testing.T) {
	t.Setenv("TERM", "dumb")
	if !IsDumbTerminal() {
		t.Error("expected IsDumbTerminal to return true when TERM=dumb")
	}
}

func TestNewThemePlain(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf)
	// On a non-TTY, all renders should be plain text with no ANSI codes.
	got := theme.Label.Render("hello")
	if got != "hello" {
		t.Errorf("expected plain Label render, got %q", got)
	}
}

func TestThemeBadgeOK(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf)
	got := theme.Badge("OK")
	if !strings.Contains(got, "OK") {
		t.Errorf("expected badge to contain 'OK', got %q", got)
	}
}

func TestThemeBadgeBlocker(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf)
	got := theme.Badge("BLOCKER")
	if !strings.Contains(got, "BLOCK") {
		t.Errorf("expected badge to contain 'BLOCK', got %q", got)
	}
}

func TestThemeKeyValue(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf)
	got := theme.KeyValue("host", "localhost")
	if !strings.Contains(got, "host:") {
		t.Errorf("expected KeyValue to contain 'host:', got %q", got)
	}
	if !strings.Contains(got, "localhost") {
		t.Errorf("expected KeyValue to contain 'localhost', got %q", got)
	}
}

func TestThemeNextStepBlock(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf)
	got := theme.NextStepBlock([]string{"yeast doctor", "yeast up"})
	if !strings.Contains(got, "yeast doctor") {
		t.Errorf("expected NextStepBlock to contain step text, got %q", got)
	}
	if !strings.Contains(got, "1.") {
		t.Errorf("expected NextStepBlock to number steps, got %q", got)
	}
}

func TestThemeSupportTierBadge(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf)
	for _, tier := range []string{"A", "B", "C", "D"} {
		got := theme.SupportTierBadge(tier)
		if !strings.Contains(got, "Tier") {
			t.Errorf("SupportTierBadge(%q): expected 'Tier' in output, got %q", tier, got)
		}
	}
}

func TestThemeSuccessWarningBlockerLines(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf)

	success := theme.SuccessLine("all good")
	if !strings.Contains(success, "all good") {
		t.Errorf("SuccessLine missing text: %q", success)
	}

	warn := theme.WarnLine("be careful")
	if !strings.Contains(warn, "be careful") {
		t.Errorf("WarnLine missing text: %q", warn)
	}

	blocker := theme.BlockerLine("broken")
	if !strings.Contains(blocker, "broken") {
		t.Errorf("BlockerLine missing text: %q", blocker)
	}
}

func TestThemeTable(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf)
	rows := [][]string{
		{"NAME", "STATUS"},
		{"web", "running"},
		{"db", "stopped"},
	}
	got := theme.Table(rows)
	for _, want := range []string{"NAME", "STATUS", "web", "running", "db", "stopped"} {
		if !strings.Contains(got, want) {
			t.Errorf("Table missing %q in output: %s", want, got)
		}
	}
}
