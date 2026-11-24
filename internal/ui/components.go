package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Badge renders a short status pill: OK, WARN, BLOCK, or custom text.
func (t Theme) Badge(status string) string {
	switch strings.ToUpper(status) {
	case "OK":
		return t.Success.Render("OK")
	case "WARN", "WARNING":
		return t.Warning.Render("WARN")
	case "BLOCK", "BLOCKER":
		return t.Blocker.Render("BLOCK")
	default:
		return t.Muted.Render(status)
	}
}

// SupportTierBadge renders a support tier badge with the appropriate color.
func (t Theme) SupportTierBadge(tier string) string {
	switch tier {
	case "A":
		return t.Success.Render("Tier A")
	case "B":
		return t.Warning.Render("Tier B")
	default:
		return t.Muted.Render("Tier " + tier)
	}
}

// Heading renders a section heading line.
func (t Theme) Heading(text string) string {
	return t.Header.Render(text)
}

// KeyValue renders a two-column key: value line with consistent indentation.
func (t Theme) KeyValue(key, value string) string {
	return fmt.Sprintf("  %s %s", t.Label.Render(key+":"), t.Text.Render(value))
}

// CheckRow renders a single check result row: "  [BADGE] name   detail"
func (t Theme) CheckRow(badge, name, detail string) string {
	return fmt.Sprintf("  %s %s  %s",
		badge,
		t.Label.Width(24).Render(name),
		t.Muted.Render(detail),
	)
}

// SuccessLine renders a checkmark success line.
func (t Theme) SuccessLine(text string) string {
	return fmt.Sprintf("  %s %s", t.Success.Render("✓"), t.Text.Render(text))
}

// WarnLine renders a warning indicator line.
func (t Theme) WarnLine(text string) string {
	return fmt.Sprintf("  %s %s", t.Warning.Render("⚠"), t.Text.Render(text))
}

// BlockerLine renders a blocker indicator line.
func (t Theme) BlockerLine(text string) string {
	return fmt.Sprintf("  %s %s", t.Blocker.Render("✗"), t.Text.Render(text))
}

// NextStepBlock renders an indented list of next-step hints under a heading.
func (t Theme) NextStepBlock(steps []string) string {
	lines := []string{t.Header.Render("Next steps")}
	for i, s := range steps {
		lines = append(lines, fmt.Sprintf("  %s %s",
			t.Muted.Render(fmt.Sprintf("%d.", i+1)),
			t.Text.Render(s),
		))
	}
	return strings.Join(lines, "\n")
}

// cardBorder returns a rounded-border box style whose border color is only
// applied on styled terminals — plain output must stay ANSI-free.
func (t Theme) cardBorder(c color.Color) lipgloss.Style {
	s := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2)
	if t.styled {
		s = s.BorderForeground(c)
	}
	return s
}

// WarningBlock renders a bordered warning section.
func (t Theme) WarningBlock(title, body string) string {
	content := t.Warning.Render("⚠  "+title) + "\n\n" + t.Muted.Render(body)
	return t.cardBorder(t.warnCol).Render(content)
}

// ErrorBlock renders a bordered error section.
func (t Theme) ErrorBlock(title, body string) string {
	content := t.Blocker.Render("✗  "+title) + "\n\n" + t.Muted.Render(body)
	return t.cardBorder(t.blockCol).Render(content)
}

// ErrorCard renders the structured error card: what failed, why, how to fix,
// and an optional `yeast explain <code>` pointer for typed error codes.
// Error blocks are never cute — plain, precise, actionable.
func (t Theme) ErrorCard(what, why, fix, moreCode string) string {
	sym := SymbolsFor(true)
	var b strings.Builder
	b.WriteString(t.Blocker.Render(sym.Fail + "  " + what))
	if why != "" {
		b.WriteString("\n\n" + t.Muted.Render("why:  ") + t.Text.Render(why))
	}
	if fix != "" {
		b.WriteString("\n\n" + t.Muted.Render("fix:  ") + t.Text.Render(fix))
	}
	if moreCode != "" {
		b.WriteString("\n\n" + t.Muted.Render("more: ") + t.AccentSoft.Render("yeast explain "+moreCode))
	}
	return t.cardBorder(t.blockCol).Render(b.String())
}

// WidthTier buckets terminal width for responsive layout.
type WidthTier int

const (
	// WidthFull is >= 80 cols: side-by-side sprite + status layout.
	WidthFull WidthTier = iota
	// WidthCompact is 60–79 cols: single-line header, no sprite.
	WidthCompact
	// WidthNarrow is < 60 cols: plain stacked text only.
	WidthNarrow
)

// TierFor maps a column width to its tier.
func TierFor(width int) WidthTier {
	switch {
	case width >= 80:
		return WidthFull
	case width >= 60:
		return WidthCompact
	default:
		return WidthNarrow
	}
}

// Symbols holds the glyph set for status icons — Unicode or ASCII fallback.
type Symbols struct {
	OK     string // ✓ / [ok]
	Fail   string // ✗ / [x]
	Warn   string // ! / [!]
	Bullet string // • / *
	Arrow  string // → / ->
	Dot    string // ● / o
	Ring   string // ○ / o
}

// SymbolsFor returns the Unicode set when utf8 is true, ASCII otherwise.
func SymbolsFor(utf8 bool) Symbols {
	if !utf8 {
		return Symbols{OK: "[ok]", Fail: "[x]", Warn: "[!]", Bullet: "*", Arrow: "->", Dot: "o", Ring: "o"}
	}
	return Symbols{OK: "✓", Fail: "✗", Warn: "!", Bullet: "•", Arrow: "→", Dot: "●", Ring: "○"}
}

// Divider returns a simple separator line.
func (t Theme) Divider() string {
	return t.Border.Render(strings.Repeat("─", 40))
}

// Table renders rows as a padded plain table. rows[0] is the header row.
func (t Theme) Table(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if w := lipgloss.Width(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	lines := make([]string, 0, len(rows))
	for ri, row := range rows {
		cells := make([]string, 0, len(row))
		for ci, cell := range row {
			style := t.Text
			if ri == 0 {
				style = t.Header
			} else if ci == 1 {
				style = statusStyle(t, cell)
			}
			cells = append(cells, style.Width(widths[ci]).Render(cell))
		}
		lines = append(lines, "  "+strings.Join(cells, t.Border.Render("  ")))
	}
	return strings.Join(lines, "\n")
}

func statusStyle(t Theme, status string) lipgloss.Style {
	switch status {
	case "running":
		return t.Success
	case "stopped", "already_stopped":
		return t.Warning
	case "destroyed":
		return t.Blocker
	default:
		return t.Text
	}
}
