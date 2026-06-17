package ui

import (
	"fmt"
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

// WarningBlock renders a bordered warning section.
func (t Theme) WarningBlock(title, body string) string {
	content := t.Warning.Render("⚠  "+title) + "\n\n" + t.Muted.Render(body)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorWarning).
		Padding(0, 2).
		Render(content)
}

// ErrorBlock renders a bordered error section.
func (t Theme) ErrorBlock(title, body string) string {
	content := t.Blocker.Render("✗  "+title) + "\n\n" + t.Muted.Render(body)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlocker).
		Padding(0, 2).
		Render(content)
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
