package ui

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Twarga/yeast/internal/ui/sprites"
)

var updateGolden = flag.Bool("update-golden", false, "regenerate render golden files")

var goldenMoods = []sprites.Mood{
	sprites.MoodIdle, sprites.MoodWorking, sprites.MoodSuccess,
	sprites.MoodError, sprites.MoodSleep, sprites.MoodQuirk,
}

var goldenWidths = []int{40, 60, 80, 120}

var goldenDepths = []struct {
	name  string
	depth sprites.ColorDepth
}{
	{"truecolor", sprites.DepthTruecolor},
	{"256", sprites.Depth256},
	{"16", sprites.Depth16},
	{"none", sprites.DepthNone},
}

// renderGoldenOutput drives one full LiveStage lifecycle deterministically:
// Start → two steps → Finish(success). Live widths paint one settle frame;
// narrow widths emit static lines. The captured buffer is the golden.
func renderGoldenOutput(width int, depth sprites.ColorDepth, mood sprites.Mood) string {
	var buf bytes.Buffer
	s := NewLiveStage(&buf, LiveOptions{Force: true, Width: width, Depth: depth})
	s.Start(mood)
	s.SetStep("image", "Pull base image", StepDone, "")
	s.SetStep("ssh", "Wait for SSH", StepRunning, "web")
	s.Finish(sprites.MoodSuccess, "2 machines humming.")
	return buf.String()
}

// TestRenderGolden covers every mood × width × color depth end-to-end through
// the LiveStage. Regenerate with: go test ./internal/ui -update-golden
func TestRenderGolden(t *testing.T) {
	for _, mood := range goldenMoods {
		for _, width := range goldenWidths {
			for _, d := range goldenDepths {
				name := fmt.Sprintf("%s.w%d.%s.golden", mood, width, d.name)
				path := filepath.Join("testdata", "render", name)
				got := renderGoldenOutput(width, d.depth, mood)
				t.Run(fmt.Sprintf("%s/w%d/%s", mood, width, d.name), func(t *testing.T) {
					if *updateGolden {
						if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
							t.Fatalf("write golden: %v", err)
						}
					}
					want, err := os.ReadFile(path)
					if err != nil {
						t.Fatalf("missing golden %s (run with -update-golden): %v", path, err)
					}
					if got != string(want) {
						t.Fatalf("golden mismatch: %s — run `go test ./internal/ui -update-golden` and review", path)
					}
				})
			}
		}
	}
}

// TestRenderTierBehavior pins the responsive contract:
//   - width >= 80 → live in-place animation (cursor sequences present)
//   - width 60–79 → static lines (no cursor control, no sprite)
//   - width < 60 → same static treatment
func TestRenderTierBehavior(t *testing.T) {
	for _, width := range goldenWidths {
		var buf bytes.Buffer
		s := NewLiveStage(&buf, LiveOptions{Force: true, Width: width, Depth: sprites.Depth16})
		s.Start(sprites.MoodWorking)
		s.SetStep("x", "step", StepDone, "")
		s.Finish(sprites.MoodSuccess, "done")
		out := buf.String()

		if width >= 80 {
			if !strings.Contains(out, "\x1b[?25l") || !strings.Contains(out, "\x1b[?25h") {
				t.Fatalf("width %d: expected cursor hide+restore in live output", width)
			}
		} else {
			if strings.Contains(out, "\x1b[?25") || strings.Contains(out, "\x1b[1A") {
				t.Fatalf("width %d: static output must not control the cursor: %q", width, out)
			}
			if strings.ContainsRune(out, '▀') || strings.ContainsRune(out, '▄') {
				t.Fatalf("width %d: static output must not contain sprite art", width)
			}
		}
	}
}

// TestProfileEnvOverrides verifies YEAST_COLOR / YEAST_WIDTH / YEAST_UTF8.
func TestProfileEnvOverrides(t *testing.T) {
	var buf bytes.Buffer

	t.Setenv("YEAST_COLOR", "256")
	if DepthFor(&buf) != sprites.Depth256 {
		t.Fatal("YEAST_COLOR=256 should force Depth256")
	}
	t.Setenv("YEAST_COLOR", "none")
	if DepthFor(&buf) != sprites.DepthNone {
		t.Fatal("YEAST_COLOR=none should force DepthNone")
	}
	t.Setenv("YEAST_COLOR", "truecolor")
	if DepthFor(&buf) != sprites.DepthTruecolor {
		t.Fatal("YEAST_COLOR=truecolor should force DepthTruecolor")
	}
	os.Unsetenv("YEAST_COLOR")

	t.Setenv("YEAST_WIDTH", "123")
	if WidthFor(&buf) != 123 {
		t.Fatal("YEAST_WIDTH=123 should force width 123")
	}
	os.Unsetenv("YEAST_WIDTH")

	t.Setenv("YEAST_UTF8", "0")
	if UTF8For(&buf) {
		t.Fatal("YEAST_UTF8=0 should force ASCII")
	}
	t.Setenv("YEAST_UTF8", "1")
	// buf is not a TTY — the env override still wins per contract.
	if !UTF8For(&buf) {
		t.Fatal("YEAST_UTF8=1 should force UTF-8")
	}
}
