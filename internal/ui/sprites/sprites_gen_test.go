package sprites

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// spriteArtifact mirrors the JSON written by scripts/export-sprites.mjs.
type spriteArtifact struct {
	Name    string                      `json:"name"`
	Cell    struct{ Width, Height int } `json:"cell"`
	Palette map[string]string           `json:"palette"`
	Frames  map[string][]string         `json:"frames"`
}

// loadArtifact reads the committed assets/sprites/pixel.json source artifact.
func loadArtifact(t *testing.T) spriteArtifact {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "assets", "sprites", "pixel.json"))
	if err != nil {
		t.Fatalf("read pixel.json: %v", err)
	}
	var a spriteArtifact
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("parse pixel.json: %v", err)
	}
	return a
}

// TestSpriteDrift ensures sprites_gen.go stays in sync with the committed
// assets/sprites/pixel.json. Regenerate with `go generate ./internal/ui/sprites`.
func TestSpriteDrift(t *testing.T) {
	a := loadArtifact(t)

	if !reflect.DeepEqual(a.Palette, Palette) {
		t.Fatalf("Palette drift: json=%v gen=%v — run go generate ./internal/ui/sprites", a.Palette, Palette)
	}
	if !reflect.DeepEqual(a.Frames, Frames) {
		t.Fatalf("Frames drift: json=%v gen=%v — run go generate ./internal/ui/sprites", a.Frames, Frames)
	}
	if a.Cell.Width != CellWidth || a.Cell.Height != CellHeight {
		t.Fatalf("cell dims drift: json=%dx%d gen=%dx%d", a.Cell.Width, a.Cell.Height, CellWidth, CellHeight)
	}
	if a.Name != SpriteName {
		t.Fatalf("name drift: json=%q gen=%q", a.Name, SpriteName)
	}
}

// TestSpriteContract pins the Yeast Cell mascot contract: only yeast frames,
// exact expected frame set, 16x15 cell, 7 hex colors, clean pixel charset.
func TestSpriteContract(t *testing.T) {
	want := []string{
		"baby", "blush_big", "bubble", "bud_m", "bud_s", "cap", "cell",
		"eyes_closed", "eyes_happy", "eyes_open", "eyes_strain",
		"mouth_frown", "mouth_open", "mouth_smile", "pop", "z",
	}
	got := append([]string(nil), FrameNames...)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("frame set mismatch:\nwant %v\ngot  %v", want, got)
	}
	if len(Palette) != 7 {
		t.Fatalf("expected 7 palette colors, got %d", len(Palette))
	}
	for key, hex := range Palette {
		if len(key) != 1 {
			t.Fatalf("palette key %q must be one byte", key)
		}
		if len(hex) != 7 || hex[0] != '#' {
			t.Fatalf("palette %q has non-hex value %q", key, hex)
		}
	}
	if len(Frames["cell"]) != CellHeight {
		t.Fatalf("cell frame must be %d rows, got %d", CellHeight, len(Frames["cell"]))
	}
	for i, row := range Frames["cell"] {
		if len(row) != CellWidth {
			t.Fatalf("cell row %d is %d px wide, want %d", i, len(row), CellWidth)
		}
	}
	for name, rows := range Frames {
		for _, row := range rows {
			for _, ch := range []byte(row) {
				if ch != '.' {
					if _, ok := Palette[string(ch)]; !ok {
						t.Fatalf("frame %q uses unknown pixel %q", name, ch)
					}
				}
			}
		}
	}
}
