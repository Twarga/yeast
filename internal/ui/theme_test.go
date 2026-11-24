package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestTierFor(t *testing.T) {
	cases := []struct {
		width int
		want  WidthTier
	}{
		{120, WidthFull}, {80, WidthFull}, {79, WidthCompact},
		{60, WidthCompact}, {59, WidthNarrow}, {40, WidthNarrow}, {0, WidthNarrow},
	}
	for _, c := range cases {
		if got := TierFor(c.width); got != c.want {
			t.Fatalf("TierFor(%d) = %v, want %v", c.width, got, c.want)
		}
	}
}

func TestSymbolsFor(t *testing.T) {
	u := SymbolsFor(true)
	if u.OK != "✓" || u.Fail != "✗" || u.Arrow != "→" {
		t.Fatalf("UTF-8 symbols wrong: %+v", u)
	}
	a := SymbolsFor(false)
	if a.OK != "[ok]" || a.Fail != "[x]" || a.Arrow != "->" {
		t.Fatalf("ASCII symbols wrong: %+v", a)
	}
	for _, s := range []string{a.OK, a.Fail, a.Warn, a.Bullet, a.Arrow, a.Dot, a.Ring} {
		for _, r := range s {
			if r > 127 {
				t.Fatalf("ASCII symbol %q contains non-ASCII %q", s, r)
			}
		}
	}
}

func TestThemeErrorCard(t *testing.T) {
	var buf bytes.Buffer
	theme := NewTheme(&buf) // plain (non-TTY)
	card := theme.ErrorCard("cannot start vm", "kvm blocked", "enable /dev/kvm", "kvm_unavailable")
	for _, want := range []string{"cannot start vm", "why:", "fix:", "yeast explain kvm_unavailable"} {
		if !strings.Contains(card, want) {
			t.Fatalf("ErrorCard missing %q:\n%s", want, card)
		}
	}
	if strings.Contains(card, "\x1b[") {
		t.Fatal("plain theme must emit no ANSI")
	}
}

func TestThemeAdaptiveTokens(t *testing.T) {
	for _, bg := range []string{"dark", "light"} {
		t.Setenv("YEAST_BACKGROUND", bg)
		var buf bytes.Buffer
		theme := NewTheme(&buf)
		if theme.Dark() != (bg == "dark") {
			t.Fatalf("YEAST_BACKGROUND=%s: Dark() = %v", bg, theme.Dark())
		}
	}
}
