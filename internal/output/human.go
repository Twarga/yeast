package output

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"yeast/internal/app"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

func RenderHuman(w io.Writer, command string, data any) error {
	theme := newHumanTheme(w)

	switch value := data.(type) {
	case string:
		_, err := fmt.Fprintln(w, theme.Value.Render(value))
		return err
	case app.InitResult:
		return renderInit(w, theme, value)
	case app.TemplateListResult:
		return renderTemplateList(w, theme, value)
	case app.PullResult:
		return renderPull(w, theme, value)
	case app.DoctorResult:
		return renderDoctor(w, theme, value)
	case app.UpResult:
		return renderUp(w, theme, value)
	case app.StatusResult:
		return renderStatus(w, theme, value)
	case app.ExecResult:
		return renderExec(w, theme, value)
	case app.CopyResult:
		return renderCopy(w, theme, value)
	case app.InspectResult:
		return renderInspect(w, theme, value)
	case app.LogsResult:
		return renderLogs(w, theme, value)
	case app.ProvisionResult:
		return renderProvision(w, theme, value)
	case app.SnapshotResult:
		return renderSnapshot(w, theme, value)
	case app.RestoreResult:
		return renderRestore(w, theme, value)
	case app.SnapshotsResult:
		return renderSnapshots(w, theme, value)
	case app.DeleteSnapshotResult:
		return renderDeleteSnapshot(w, theme, value)
	case app.DownResult:
		return renderDown(w, theme, value)
	case app.DestroyResult:
		return renderDestroy(w, theme, value)
	case app.ImageCleanResult:
		return renderImageClean(w, theme, value)
	default:
		return fmt.Errorf("unsupported human render type for %s: %T", command, data)
	}
}

type humanTheme struct {
	Title   lipgloss.Style
	Muted   lipgloss.Style
	Label   lipgloss.Style
	Value   lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Blocker lipgloss.Style
	Border  lipgloss.Style
	Header  lipgloss.Style
	Box     lipgloss.Style
}

func newHumanTheme(w io.Writer) humanTheme {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	if !terminalStylingEnabled(w) {
		style := lipgloss.NewStyle()
		return humanTheme{
			Title:   style,
			Muted:   style,
			Label:   style,
			Value:   style,
			Success: style,
			Warning: style,
			Blocker: style,
			Border:  style,
			Header:  style,
			Box:     box,
		}
	}

	return humanTheme{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F5F0E8")),
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8C7355")),
		Label: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#D6A85F")),
		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F5F0E8")),
		Success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#14B8A6")),
		Warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#D6A85F")),
		Blocker: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EF4852")),
		Border: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3A352F")),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#D6A85F")),
		Box: box.
			BorderForeground(lipgloss.Color("#3A352F")),
	}
}

func terminalStylingEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func renderInit(w io.Writer, theme humanTheme, value app.InitResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Title.Render("Project initialized"),
		keyValue(theme, "config", value.ConfigPath),
		keyValue(theme, "metadata", value.MetadataPath),
		keyValue(theme, "project", value.ProjectID),
	}
	if value.Template != "" {
		lines = append(lines, keyValue(theme, "template", value.Template))
	}
	return writeBlock(w, theme, lines)
}

func renderTemplateList(w io.Writer, theme humanTheme, value app.TemplateListResult) error {
	rows := [][]string{{"NAME", "CATEGORY", "SOURCE", "DESCRIPTION"}}
	for _, template := range value.Templates {
		rows = append(rows, []string{
			template.Name,
			template.Category,
			template.Source,
			template.Description,
		})
	}

	lines := []string{theme.Title.Render("Project templates")}
	lines = append(lines, renderRows(theme, rows)...)
	return writeBlock(w, theme, lines)
}

func renderPull(w io.Writer, theme humanTheme, value app.PullResult) error {
	if value.List && len(value.ImageGroups) > 0 {
		lines := []string{theme.Title.Render("Available images")}
		lines = append(lines, "")
		categoryLabels := map[string]string{
			"general":    "General Purpose",
			"devops":     "DevOps & Cloud",
			"enterprise": "Enterprise",
			"security":   "Security",
			"minimal":    "Minimal",
			"niche":      "Niche",
		}
		for _, group := range value.ImageGroups {
			label := categoryLabels[group.Category]
			if label == "" {
				label = group.Category
			}
			lines = append(lines, "  "+theme.Label.Render(label)+":")
			for _, img := range group.Images {
				cloudInit := ""
				if !img.CloudInit {
					cloudInit = " " + theme.Muted.Render("(manual)")
				}
				lines = append(lines, fmt.Sprintf("    %-24s %s%s  %s",
					theme.Value.Render(img.Name),
					theme.Muted.Render(img.Description),
					cloudInit,
					theme.Muted.Render(img.Size),
				))
			}
			lines = append(lines, "")
		}
		lines = append(lines, "  "+theme.Muted.Render("Use: yeast pull <image> to download or get setup instructions"))
		return writeBlock(w, theme, lines)
	}

	if value.List {
		lines := []string{theme.Title.Render("Available images")}
		for _, image := range value.Images {
			lines = append(lines, "  "+theme.Success.Render("*")+" "+theme.Value.Render(image))
		}
		return writeBlock(w, theme, lines)
	}

	if value.ManualHint != "" {
		lines := []string{
			theme.Warning.Render("Manual setup required") + " " + theme.Value.Render(value.ImageName),
			"",
		}
		for _, line := range splitLines(value.ManualHint) {
			lines = append(lines, "  "+line)
		}
		return writeBlock(w, theme, lines)
	}

	if len(value.SearchResults) > 0 {
		lines := []string{theme.Warning.Render("Multiple images match:") + " " + theme.Value.Render(value.SearchResults[0])}
		for _, name := range value.SearchResults[1:] {
			lines = append(lines, "  "+theme.Muted.Render("-")+" "+theme.Value.Render(name))
		}
		lines = append(lines, "")
		lines = append(lines, "  "+theme.Muted.Render("Specify the full name: yeast pull "+value.SearchResults[0]))
		return writeBlock(w, theme, lines)
	}

	if len(value.CachedImages) > 0 {
		lines := []string{theme.Title.Render("Cached images")}
		lines = append(lines, "")
		for _, img := range value.CachedImages {
			lines = append(lines, fmt.Sprintf("  %s  %s",
				theme.Value.Render(img.Name),
				theme.Muted.Render(img.Path),
			))
		}
		lines = append(lines, "")
		lines = append(lines, "  "+theme.Muted.Render("Use: yeast images clean <name> to remove"))
		return writeBlock(w, theme, lines)
	}

	lines := []string{
		theme.Success.Render("OK") + " " + theme.Title.Render("Image pulled"),
		keyValue(theme, "image", value.ImageName),
		keyValue(theme, "path", value.ImagePath),
	}
	return writeBlock(w, theme, lines)
}

func splitLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		result = append(result, line)
	}
	return result
}

func renderDoctor(w io.Writer, theme humanTheme, value app.DoctorResult) error {
	lines := []string{theme.Title.Render("Host doctor")}
	for _, check := range value.Checks {
		lines = append(lines, fmt.Sprintf(
			"  %s  %s  %s",
			doctorBadge(theme, check.Status),
			theme.Label.Render(check.Name),
			theme.Muted.Render(check.Details),
		))
	}
	lines = append(lines,
		"",
		keyValue(theme, "blockers", fmt.Sprintf("%d", value.Blockers)),
		keyValue(theme, "warnings", fmt.Sprintf("%d", value.Warnings)),
	)
	return writeBlock(w, theme, lines)
}

func renderUp(w io.Writer, theme humanTheme, value app.UpResult) error {
	rows := [][]string{{"NAME", "STATUS", "SSH"}}
	for _, instance := range value.Instances {
		ssh := "-"
		if instance.SSHAddress != "" {
			ssh = instance.SSHAddress
		}
		rows = append(rows, []string{instance.Name, instance.Status, ssh})
	}

	lines := []string{theme.Success.Render("✓") + " " + theme.Title.Render("All instances ready")}
	lines = append(lines, "")
	lines = append(lines, renderRows(theme, rows)...)

	// Connect hint for the first instance.
	if len(value.Instances) > 0 {
		first := value.Instances[0]
		if first.SSHAddress != "" {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("  %s %s",
				theme.Muted.Render("Connect:"),
				theme.Value.Render("yeast ssh "+first.Name),
			))
		}
	}

	return writeBlock(w, theme, lines)
}

func renderStatus(w io.Writer, theme humanTheme, value app.StatusResult) error {
	instances := append([]app.StatusInstanceResult(nil), value.Instances...)
	sort.Slice(instances, func(i, j int) bool { return instances[i].Name < instances[j].Name })

	rows := [][]string{{"NAME", "STATUS", "SSH", "LAB IP"}}
	for _, instance := range instances {
		ssh := "-"
		if instance.SSHPort > 0 {
			ssh = fmt.Sprintf("127.0.0.1:%d", instance.SSHPort)
		}
		labIP := "-"
		if instance.LabIP != "" {
			labIP = instance.LabIP
		}
		rows = append(rows, []string{instance.Name, instance.Status, ssh, labIP})
	}

	lines := []string{theme.Title.Render("Project status")}
	lines = append(lines, renderRows(theme, rows)...)
	return writeBlock(w, theme, lines)
}

func renderProvision(w io.Writer, theme humanTheme, value app.ProvisionResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Title.Render("Provisioning finished"),
		keyValue(theme, "instance", value.Instance.Name),
		keyValue(theme, "status", string(value.Instance.ProvisioningStatus)),
		keyValue(theme, "ssh", value.Instance.SSHAddress),
		keyValue(theme, "log", value.Instance.ProvisionLogPath),
	}
	if value.Instance.LastError != "" {
		lines = append(lines, keyValue(theme, "last_error", value.Instance.LastError))
	}
	return writeBlock(w, theme, lines)
}

func renderExec(w io.Writer, theme humanTheme, value app.ExecResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Title.Render("Command finished"),
		keyValue(theme, "instance", value.Instance),
		keyValue(theme, "command", value.Run.Command),
		keyValue(theme, "exit_code", fmt.Sprintf("%d", value.Run.ExitCode)),
		keyValue(theme, "duration", value.Run.Duration.String()),
	}
	if value.Run.TimedOut {
		lines = append(lines, keyValue(theme, "timed_out", "true"))
	}
	if value.Run.Stdout != "" {
		lines = append(lines, "", theme.Label.Render("stdout:"), indentBlock(value.Run.Stdout))
	}
	if value.Run.Stderr != "" {
		lines = append(lines, "", theme.Label.Render("stderr:"), indentBlock(value.Run.Stderr))
	}
	return writeBlock(w, theme, lines)
}

func renderCopy(w io.Writer, theme humanTheme, value app.CopyResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Title.Render("Copy finished"),
		keyValue(theme, "instance", value.Instance),
		keyValue(theme, "direction", string(value.Direction)),
		keyValue(theme, "source", value.Source),
		keyValue(theme, "destination", value.Destination),
		keyValue(theme, "duration", value.Duration.String()),
	}
	return writeBlock(w, theme, lines)
}

func renderInspect(w io.Writer, theme humanTheme, value app.InspectResult) error {
	lines := []string{
		theme.Title.Render("Instance inspect"),
		keyValue(theme, "name", value.Instance.Name),
		keyValue(theme, "status", value.Instance.Status),
		keyValue(theme, "ssh", sshAddress(value.Instance.SSHPort)),
		keyValue(theme, "lab_ip", dashIfEmpty(value.Instance.LabIP)),
		keyValue(theme, "runtime_dir", dashIfEmpty(value.Instance.RuntimeDir)),
		keyValue(theme, "provision_log", dashIfEmpty(value.Instance.ProvisionLogPath)),
		keyValue(theme, "provision_status", string(value.Instance.ProvisioningStatus)),
		keyValue(theme, "snapshot_count", fmt.Sprintf("%d", value.SnapshotCount)),
	}
	if value.Instance.LastError != "" {
		lines = append(lines, keyValue(theme, "last_error", value.Instance.LastError))
	}
	if len(value.SnapshotNames) > 0 {
		lines = append(lines, "", theme.Label.Render("snapshots:"), "  "+theme.Value.Render(strings.Join(value.SnapshotNames, ", ")))
	}
	return writeBlock(w, theme, lines)
}

func renderLogs(w io.Writer, theme humanTheme, value app.LogsResult) error {
	lines := []string{
		theme.Title.Render("Instance logs"),
		keyValue(theme, "instance", value.Instance),
		keyValue(theme, "path", value.LogPath),
	}
	if value.Content != "" {
		lines = append(lines, "", theme.Label.Render("content:"), indentBlock(value.Content))
	}
	return writeBlock(w, theme, lines)
}

func renderSnapshot(w io.Writer, theme humanTheme, value app.SnapshotResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Title.Render("Snapshot created"),
		keyValue(theme, "instance", value.Instance),
		keyValue(theme, "snapshot", value.Snapshot.Name),
		keyValue(theme, "path", value.Snapshot.DiskPath),
		keyValue(theme, "created_at", value.Snapshot.CreatedAt.Format("2006-01-02 15:04:05 MST")),
	}
	if value.Snapshot.Description != "" {
		lines = append(lines, keyValue(theme, "description", value.Snapshot.Description))
	}
	return writeBlock(w, theme, lines)
}

func renderRestore(w io.Writer, theme humanTheme, value app.RestoreResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Title.Render("Snapshot restored"),
		keyValue(theme, "instance", value.Instance),
		keyValue(theme, "snapshot", value.Snapshot.Name),
		keyValue(theme, "path", value.Snapshot.DiskPath),
	}
	return writeBlock(w, theme, lines)
}

func renderSnapshots(w io.Writer, theme humanTheme, value app.SnapshotsResult) error {
	rows := [][]string{{"NAME", "CREATED", "DESCRIPTION", "PATH"}}
	for _, snapshot := range value.Snapshots {
		rows = append(rows, []string{
			snapshot.Name,
			snapshot.CreatedAt.Format("2006-01-02 15:04:05 MST"),
			snapshot.Description,
			snapshot.DiskPath,
		})
	}

	lines := []string{
		theme.Title.Render("Instance snapshots"),
		keyValue(theme, "instance", value.Instance),
	}
	lines = append(lines, renderRows(theme, rows)...)
	return writeBlock(w, theme, lines)
}

func renderDeleteSnapshot(w io.Writer, theme humanTheme, value app.DeleteSnapshotResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Title.Render("Snapshot deleted"),
		keyValue(theme, "instance", value.Instance),
		keyValue(theme, "snapshot", value.Snapshot),
	}
	return writeBlock(w, theme, lines)
}

func renderDown(w io.Writer, theme humanTheme, value app.DownResult) error {
	rows := [][]string{{"NAME", "STATUS"}}
	for _, instance := range value.Instances {
		name, status := actionFields(instance)
		rows = append(rows, []string{name, status})
	}

	lines := []string{theme.Success.Render("✓") + " " + theme.Title.Render("All instances stopped")}
	lines = append(lines, "")
	lines = append(lines, renderRows(theme, rows)...)
	return writeBlock(w, theme, lines)
}

func renderDestroy(w io.Writer, theme humanTheme, value app.DestroyResult) error {
	rows := [][]string{{"NAME", "STATUS"}}
	for _, instance := range value.Instances {
		name, status := actionFields(instance)
		rows = append(rows, []string{name, status})
	}

	title := "All instances destroyed"
	if !value.FilesDeleted {
		title = "Instances stopped (files kept)"
	}

	lines := []string{theme.Success.Render("✓") + " " + theme.Title.Render(title)}
	lines = append(lines, "")
	lines = append(lines, renderRows(theme, rows)...)
	return writeBlock(w, theme, lines)
}

func renderImageClean(w io.Writer, theme humanTheme, value app.ImageCleanResult) error {
	if value.DryRun {
		lines := []string{theme.Title.Render("Would remove")}
		lines = append(lines, "")
		for _, item := range value.Removed {
			lines = append(lines, fmt.Sprintf("  %s  %s",
				theme.Value.Render(item.Name),
				theme.Muted.Render(item.SizeH),
			))
		}
		lines = append(lines, "")
		lines = append(lines, "  "+theme.Muted.Render("Total: "+value.TotalSizeH))
		return writeBlock(w, theme, lines)
	}

	if len(value.Removed) == 0 {
		lines := []string{theme.Muted.Render("No cached images to remove")}
		return writeBlock(w, theme, lines)
	}

	lines := []string{theme.Success.Render("✓") + " " + theme.Title.Render("Cached images removed")}
	lines = append(lines, "")
	for _, item := range value.Removed {
		lines = append(lines, fmt.Sprintf("  %s  %s freed",
			theme.Value.Render(item.Name),
			theme.Muted.Render(item.SizeH),
		))
	}
	lines = append(lines, "")
	lines = append(lines, "  "+theme.Muted.Render("Total: "+value.TotalSizeH+" freed"))
	return writeBlock(w, theme, lines)
}

type instanceAction interface {
	app.DownInstanceResult | app.DestroyInstanceResult
}

func actionFields[T instanceAction](instance T) (string, string) {
	switch value := any(instance).(type) {
	case app.DownInstanceResult:
		return value.Name, value.Status
	case app.DestroyInstanceResult:
		return value.Name, value.Status
	default:
		return "", ""
	}
}

func keyValue(theme humanTheme, key, value string) string {
	return fmt.Sprintf("  %s %s", theme.Label.Render(key+":"), theme.Value.Render(value))
}

func dashIfEmpty(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func sshAddress(port int) string {
	if port <= 0 {
		return "-"
	}
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func indentBlock(value string) string {
	lines := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "  "
	}
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func doctorBadge(theme humanTheme, status app.CheckStatus) string {
	switch status {
	case app.CheckStatusOK:
		return theme.Success.Render("OK")
	case app.CheckStatusWarning:
		return theme.Warning.Render("WARN")
	case app.CheckStatusBlocker:
		return theme.Blocker.Render("BLOCK")
	default:
		return theme.Muted.Render(string(status))
	}
}

func renderRows(theme humanTheme, rows [][]string) []string {
	if len(rows) == 0 {
		return nil
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if width := lipgloss.Width(cell); width > widths[i] {
				widths[i] = width
			}
		}
	}

	lines := make([]string, 0, len(rows))
	for rowIndex, row := range rows {
		cells := make([]string, 0, len(row))
		for i, cell := range row {
			style := theme.Value
			if rowIndex == 0 {
				style = theme.Header
			} else if i == 1 {
				style = statusStyle(theme, cell)
			}
			cells = append(cells, style.Width(widths[i]).Render(cell))
		}
		lines = append(lines, "  "+strings.Join(cells, theme.Border.Render("  ")))
	}
	return lines
}

func statusStyle(theme humanTheme, status string) lipgloss.Style {
	switch status {
	case "running":
		return theme.Success
	case "stopped", "already_stopped":
		return theme.Warning
	case "destroyed":
		return theme.Blocker
	default:
		return theme.Value
	}
}

func writeBlock(w io.Writer, theme humanTheme, lines []string) error {
	content := strings.Join(lines, "\n")
	box := theme.Box.Render(content)
	_, err := fmt.Fprintln(w, box)
	return err
}
