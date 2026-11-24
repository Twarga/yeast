package ui

import (
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/Twarga/yeast/internal/ui/sprites"
	"golang.org/x/term"
)

// DepthFor reports the best color depth for a writer, honoring the
// YEAST_COLOR=truecolor|256|16|none override. With no override it probes
// COLORTERM and TERM, falling back to DepthNone on non-styled writers.
func DepthFor(w io.Writer) sprites.ColorDepth {
	if v := os.Getenv("YEAST_COLOR"); v != "" {
		switch strings.ToLower(v) {
		case "truecolor", "24bit":
			return sprites.DepthTruecolor
		case "256":
			return sprites.Depth256
		case "16":
			return sprites.Depth16
		default:
			return sprites.DepthNone
		}
	}
	if !StylingEnabled(w) {
		return sprites.DepthNone
	}
	switch strings.ToLower(os.Getenv("COLORTERM")) {
	case "truecolor", "24bit":
		return sprites.DepthTruecolor
	}
	termName := strings.ToLower(os.Getenv("TERM"))
	for _, tc := range []string{"truecolor", "24bit", "direct", "alacritty", "foot", "kitty", "wezterm", "ghostty"} {
		if strings.Contains(termName, tc) {
			return sprites.DepthTruecolor
		}
	}
	if strings.Contains(termName, "256color") || strings.Contains(termName, "tmux") || strings.Contains(termName, "screen") {
		return sprites.Depth256
	}
	return sprites.Depth16
}

// WidthFor reports the terminal width for a writer, honoring YEAST_WIDTH.
// Returns 0 when the width cannot be determined (non-TTY writers).
func WidthFor(w io.Writer) int {
	if v := os.Getenv("YEAST_WIDTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return 0
	}
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0
	}
	return width
}

// UTF8For reports whether Unicode symbols/blocks are safe on this terminal,
// honoring YEAST_UTF8=0|1. Defaults to true on styled terminals and false on
// dumb/non-terminal writers.
func UTF8For(w io.Writer) bool {
	switch os.Getenv("YEAST_UTF8") {
	case "0":
		return false
	case "1":
		return true
	}
	if !StylingEnabled(w) {
		return false
	}
	lang := os.Getenv("LC_ALL") + os.Getenv("LC_CTYPE") + os.Getenv("LANG")
	if lang != "" && !strings.Contains(strings.ToUpper(lang), "UTF") {
		return false
	}
	return true
}
