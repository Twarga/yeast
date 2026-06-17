// Package ui provides the shared visual design system for Yeast's human-mode CLI output.
// All colors, styles, and component functions live here. The output package's
// human renderer consumes these to ensure visual consistency across commands.
package ui

import (
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Color palette — warm amber/cream/teal theme.
var (
	ColorText    = lipgloss.Color("#F5F0E8") // primary text
	ColorMuted   = lipgloss.Color("#8C7355") // secondary / dim
	ColorLabel   = lipgloss.Color("#D6A85F") // field labels, headings
	ColorSuccess = lipgloss.Color("#14B8A6") // OK / done
	ColorWarning = lipgloss.Color("#D6A85F") // warn (same as label by design)
	ColorBlocker = lipgloss.Color("#EF4852") // error / blocker
	ColorBorder  = lipgloss.Color("#3A352F") // box borders
	ColorTierA   = lipgloss.Color("#14B8A6") // support tier A
	ColorTierB   = lipgloss.Color("#D6A85F") // support tier B
	ColorTierC   = lipgloss.Color("#8C7355") // support tier C / D
)

// Theme holds all lipgloss styles for a render context. Use NewTheme to
// construct one; it degrades automatically on non-TTY or NO_COLOR writers.
type Theme struct {
	Text    lipgloss.Style
	Muted   lipgloss.Style
	Label   lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Blocker lipgloss.Style
	Border  lipgloss.Style
	Header  lipgloss.Style
	Box     lipgloss.Style
}

// NewTheme returns a Theme appropriate for the given writer.
// On non-TTY writers or when NO_COLOR / TERM=dumb is set, all styles are plain.
func NewTheme(w io.Writer) Theme {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	if !StylingEnabled(w) {
		plain := lipgloss.NewStyle()
		return Theme{
			Text:    plain,
			Muted:   plain,
			Label:   plain,
			Success: plain,
			Warning: plain,
			Blocker: plain,
			Border:  plain,
			Header:  plain,
			Box:     box,
		}
	}

	return Theme{
		Text: lipgloss.NewStyle().
			Foreground(ColorText),
		Muted: lipgloss.NewStyle().
			Foreground(ColorMuted),
		Label: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorLabel),
		Success: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess),
		Warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning),
		Blocker: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBlocker),
		Border: lipgloss.NewStyle().
			Foreground(ColorBorder),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorLabel),
		Box: box.
			BorderForeground(ColorBorder),
	}
}

// StylingEnabled reports whether ANSI styling should be emitted for w.
// Returns false when NO_COLOR, TERM=dumb, or w is not a terminal.
func StylingEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
