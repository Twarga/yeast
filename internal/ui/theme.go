// Package ui provides the shared visual design system for Yeast's human-mode CLI output.
// All colors, styles, and component functions live here. The output package's
// human renderer consumes these to ensure visual consistency across commands.
package ui

import (
	"image/color"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Crew color tokens — adaptive dark/light pairs matching the Yeast Cell
// palette. Values resolve against the detected terminal background.
var (
	crewText       = lipgloss.Color("#EFE3C8")
	crewTextLight  = lipgloss.Color("#1A1009")
	crewMuted      = lipgloss.Color("#9A8F7A")
	crewMutedLight = lipgloss.Color("#6B5D4A")
	crewAccent     = lipgloss.Color("#2FBF72")
	crewAccentDark = lipgloss.Color("#14804A")
	crewSoft       = lipgloss.Color("#8FF0B6")
	crewWarning    = lipgloss.Color("#E9A23B")
	crewWarnLight  = lipgloss.Color("#A86A12")
	crewBlocker    = lipgloss.Color("#FF4D4D")
	crewBlockLight = lipgloss.Color("#C62828")
	crewNight      = lipgloss.Color("#5A4AA8")
	crewBorder     = lipgloss.Color("#3A352F")
	crewBordLight  = lipgloss.Color("#D9CFBD")
)

// Theme holds all lipgloss styles for a render context plus the resolved
// border colors. Use NewTheme to construct one; it degrades automatically on
// non-TTY or NO_COLOR writers.
type Theme struct {
	Text       lipgloss.Style
	Muted      lipgloss.Style
	Label      lipgloss.Style
	Success    lipgloss.Style
	Warning    lipgloss.Style
	Blocker    lipgloss.Style
	Border     lipgloss.Style
	Header     lipgloss.Style
	Box        lipgloss.Style
	Accent     lipgloss.Style
	AccentSoft lipgloss.Style
	Night      lipgloss.Style

	dark      bool
	styled    bool
	warnCol   color.Color
	blockCol  color.Color
	borderCol color.Color
}

// Dark reports whether the theme resolved to a dark terminal background.
func (t Theme) Dark() bool { return t.dark }

// detectBackground picks the light/dark variant set for a writer without
// writing OSC-11 queries to the terminal (which leak literally as "]11;?"
// in PTYs, tmux, and embedded IDE terminals). Honors YEAST_BACKGROUND,
// YEAST_THEME, and COLORFGBG, defaulting to dark.
func detectBackground(_ io.Writer) bool {
	for _, key := range []string{"YEAST_BACKGROUND", "YEAST_THEME"} {
		switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
		case "light":
			return false
		case "dark":
			return true
		}
	}
	if fgbg := os.Getenv("COLORFGBG"); fgbg != "" {
		parts := strings.Split(fgbg, ";")
		bg := strings.TrimSpace(parts[len(parts)-1])
		// Standard ANSI background indices 7 (white) and 15 (bright white) indicate light terminals.
		if bg == "7" || bg == "15" {
			return false
		}
	}
	return true
}

// NewTheme returns a Theme appropriate for the given writer.
// On non-TTY writers or when NO_COLOR / TERM=dumb is set, all styles are plain.
func NewTheme(w io.Writer) Theme {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	dark := detectBackground(w)
	pick := lipgloss.LightDark(dark)
	text := pick(crewTextLight, crewText)
	muted := pick(crewMutedLight, crewMuted)
	accent := pick(crewAccentDark, crewAccent)
	soft := pick(crewAccentDark, crewSoft)
	warn := pick(crewWarnLight, crewWarning)
	block := pick(crewBlockLight, crewBlocker)
	border := pick(crewBordLight, crewBorder)

	if !StylingEnabled(w) {
		plain := lipgloss.NewStyle()
		return Theme{
			Text: plain, Muted: plain, Label: plain, Success: plain,
			Warning: plain, Blocker: plain, Border: plain, Header: plain,
			Accent: plain, AccentSoft: plain, Night: plain,
			Box: box, dark: dark,
			warnCol: warn, blockCol: block, borderCol: border,
		}
	}

	return Theme{
		Text:       lipgloss.NewStyle().Foreground(text),
		Muted:      lipgloss.NewStyle().Foreground(muted),
		Label:      lipgloss.NewStyle().Bold(true).Foreground(soft),
		Success:    lipgloss.NewStyle().Bold(true).Foreground(accent),
		Warning:    lipgloss.NewStyle().Bold(true).Foreground(warn),
		Blocker:    lipgloss.NewStyle().Bold(true).Foreground(block),
		Border:     lipgloss.NewStyle().Foreground(border),
		Header:     lipgloss.NewStyle().Bold(true).Foreground(soft),
		Accent:     lipgloss.NewStyle().Bold(true).Foreground(accent),
		AccentSoft: lipgloss.NewStyle().Foreground(soft),
		Night:      lipgloss.NewStyle().Foreground(crewNight),
		Box:        box.BorderForeground(border),
		dark:       dark,
		styled:     true,
		warnCol:    warn,
		blockCol:   block,
		borderCol:  border,
	}
}

// Backwards-compatible token accessors for older callers. New code should use
// the Theme fields directly.
var (
	ColorText    = crewText
	ColorMuted   = crewMuted
	ColorLabel   = crewSoft
	ColorSuccess = crewAccent
	ColorWarning = crewWarning
	ColorBlocker = crewBlocker
	ColorBorder  = crewBorder
	ColorTierA   = crewAccent
	ColorTierB   = crewWarning
	ColorTierC   = crewMuted
)

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
