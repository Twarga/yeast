package sprites

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden files")

var allMoods = []Mood{MoodIdle, MoodWorking, MoodSuccess, MoodError, MoodSleep, MoodQuirk}

var allDepths = []struct {
	name  string
	depth ColorDepth
}{
	{"truecolor", DepthTruecolor},
	{"256", Depth256},
	{"16", Depth16},
	{"none", DepthNone},
}

// goldenName builds the golden file name for a mood/depth pair.
func goldenName(m Mood, depthName string) string {
	return filepath.Join("testdata", fmt.Sprintf("%s.%s.golden", m, depthName))
}

// moodGoldenContent renders every step of a mood's loop plus its still frame
// into one deterministic golden blob.
func moodGoldenContent(m Mood, depth ColorDepth) string {
	var sb strings.Builder
	steps := Steps(m)
	for i := range steps {
		fmt.Fprintf(&sb, "-- step %d --\n", i)
		for _, line := range Compose(steps[i], depth) {
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}
	sb.WriteString("-- still --\n")
	for _, line := range RenderStill(m, depth) {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// TestRenderGolden is the golden snapshot suite: every mood × every color
// depth. Regenerate with: go test ./internal/ui/sprites -update
func TestRenderGolden(t *testing.T) {
	for _, m := range allMoods {
		for _, d := range allDepths {
			t.Run(fmt.Sprintf("%s/%s", m, d.name), func(t *testing.T) {
				got := moodGoldenContent(m, d.depth)
				path := goldenName(m, d.name)
				if *update {
					if err := os.MkdirAll("testdata", 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
						t.Fatalf("write golden: %v", err)
					}
				}
				want, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("missing golden %s (run with -update): %v", path, err)
				}
				if got != string(want) {
					t.Fatalf("golden mismatch for %s — run `go test ./internal/ui/sprites -update` and review the diff", path)
				}
			})
		}
	}
}

// TestRenderShape verifies the canvas always renders the expected terminal
// dimensions (22 cols × 8 rows) and that DepthNone emits nothing.
func TestRenderShape(t *testing.T) {
	for _, m := range allMoods {
		for i, step := range Steps(m) {
			lines := Compose(step, DepthTruecolor)
			if len(lines) != CanvasH/2 {
				t.Fatalf("%s step %d: got %d rows, want %d", m, i, len(lines), CanvasH/2)
			}
			if got := Compose(step, DepthNone); got != nil {
				t.Fatalf("%s step %d: DepthNone should return nil, got %d lines", m, i, len(got))
			}
		}
	}
}

// TestRenderNoOverflow ensures no step draws outside the canvas width —
// clipped pixels would silently corrupt the sprite. We detect overflow by
// rendering to a wider canvas and confirming identical output is impossible
// to distinguish... simpler: assert every placement's extent fits the canvas.
func TestRenderNoOverflow(t *testing.T) {
	for _, m := range allMoods {
		for si, step := range Steps(m) {
			for _, pl := range step {
				frame, ok := Frames[pl.Frame]
				if !ok {
					t.Fatalf("%s step %d references unknown frame %q", m, si, pl.Frame)
				}
				for fy, row := range frame {
					if row == "" {
						continue
					}
					y := pl.OY + fy
					// Vertical clipping above the canvas is allowed (cap droop,
					// z drift) but pixels must not clip below it.
					if y >= CanvasH {
						t.Fatalf("%s step %d: frame %q row %d lands at y=%d >= %d", m, si, pl.Frame, fy, y, CanvasH)
					}
					if pl.OX < 0 || pl.OX+len(row) > CanvasW {
						t.Fatalf("%s step %d: frame %q row %d overflows x [%d,%d) of %d", m, si, pl.Frame, fy, pl.OX, pl.OX+len(row), CanvasW)
					}
				}
			}
		}
	}
}

// TestStillIsSettled asserts each mood's still frame is one of its loop steps
// (a real, reachable state — never an untested bespoke frame).
func TestStillIsSettled(t *testing.T) {
	for _, m := range allMoods {
		still := RenderStill(m, DepthTruecolor)
		found := false
		for _, step := range Steps(m) {
			if strings.Join(Compose(step, DepthTruecolor), "\n") == strings.Join(still, "\n") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s still frame does not match any loop step", m)
		}
	}
}
