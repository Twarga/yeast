package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"yeast/internal/app"
	"yeast/internal/ui"

	"charm.land/lipgloss/v2"
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
	case app.CleanResult:
		return renderClean(w, theme, value)
	case app.ImageCleanResult:
		return renderImageClean(w, theme, value)
	case app.UpdateResult:
		return renderUpdate(w, theme, value)
	case app.UpdateNotice:
		return renderUpdateNotice(w, theme, value)
	default:
		return fmt.Errorf("unsupported human render type for %s: %T", command, data)
	}
}

// humanTheme wraps ui.Theme and adds a Value style alias for backwards compat
// with render functions that use theme.Value instead of theme.Text.
type humanTheme struct {
	ui.Theme
	Value lipgloss.Style
}

func newHumanTheme(w io.Writer) humanTheme {
	t := ui.NewTheme(w)
	return humanTheme{
		Theme: t,
		Value: t.Text,
	}
}

func renderInit(w io.Writer, theme humanTheme, value app.InitResult) error {
	lines := []string{
		theme.Success.Render("✓") + " " + theme.Header.Render("Project initialized"),
		keyValue(theme, "config", value.ConfigPath),
		keyValue(theme, "metadata", value.MetadataPath),
		keyValue(theme, "project", value.ProjectID),
	}
	if value.Template != "" {
		lines = append(lines, keyValue(theme, "template", value.Template))
	}
	lines = append(lines,
		"",
		theme.Header.Render("Next steps"),
		fmt.Sprintf("  %s %s", theme.Muted.Render("1."), theme.Text.Render("Review and edit yeast.yaml")),
		fmt.Sprintf("  %s %s", theme.Muted.Render("2."), theme.Text.Render("yeast up")),
		fmt.Sprintf("  %s %s", theme.Muted.Render("3."), theme.Text.Render("yeast ssh <instance-name>")),
		"",
		"  "+theme.Muted.Render("Docs: yeast docs quickstart"),
	)
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

	lines := []string{theme.Header.Render("Project templates")}
	lines = append(lines, renderRows(theme, rows)...)
	return writeBlock(w, theme, lines)
}

func renderPull(w io.Writer, theme humanTheme, value app.PullResult) error {
	if value.List && len(value.ImageGroups) > 0 {
		lines := []string{theme.Header.Render("Available images")}
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
		lines := []string{theme.Header.Render("Available images")}
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
		lines := []string{theme.Header.Render("Cached images")}
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
		theme.Success.Render("OK") + " " + theme.Header.Render("Image pulled"),
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
	lines := []string{theme.Header.Render("Host doctor")}

	// Environment and support tier summary.
	if value.Environment != "" {
		lines = append(lines, keyValue(theme, "environment", value.Environment))
	}
	if value.SupportTier != "" {
		lines = append(lines, keyValue(theme, "support", theme.SupportTierBadge(value.SupportTier)))
	}
	lines = append(lines, "")

	// Checks table.
	for _, check := range value.Checks {
		badge := doctorBadge(theme, check.Status)
		lines = append(lines, fmt.Sprintf(
			"  %s  %-24s  %s",
			badge,
			theme.Label.Render(check.Name),
			theme.Muted.Render(check.Details),
		))
	}

	lines = append(lines, "")

	// Summary line.
	switch {
	case value.Blockers > 0:
		lines = append(lines, fmt.Sprintf("  %s  %s  %s",
			theme.Blocker.Render(fmt.Sprintf("%d blocker(s)", value.Blockers)),
			theme.Muted.Render("|"),
			theme.Warning.Render(fmt.Sprintf("%d warning(s)", value.Warnings)),
		))
		lines = append(lines, "")
		lines = append(lines, "  "+theme.Muted.Render("Fix blockers before running 'yeast up'."))
	case value.Warnings > 0:
		lines = append(lines, fmt.Sprintf("  %s  %s",
			theme.Warning.Render(fmt.Sprintf("%d warning(s)", value.Warnings)),
			theme.Muted.Render("— host is usable but review warnings above"),
		))
	default:
		lines = append(lines, "  "+theme.Success.Render("✓")+" "+theme.Text.Render("Host is ready."))
	}

	return writeBlock(w, theme, lines)
}

func renderUp(w io.Writer, theme humanTheme, value app.UpResult) error {
	rows := [][]string{{"NAME", "STATUS", "SSH", "PORTS"}}
	for _, instance := range value.Instances {
		ssh := "-"
		if instance.SSHAddress != "" {
			ssh = instance.SSHAddress
		}
		rows = append(rows, []string{instance.Name, instance.Status, ssh, renderPortURLs(instance.Ports)})
	}

	lines := []string{theme.Success.Render("✓") + " " + theme.Header.Render("All instances ready")}
	lines = append(lines, "")
	lines = append(lines, renderRows(theme, rows)...)

	// Connect hints for all running instances.
	var sshHints []string
	for _, inst := range value.Instances {
		if inst.SSHAddress != "" {
			sshHints = append(sshHints, "yeast ssh "+inst.Name)
		}
	}
	if len(sshHints) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.Header.Render("Connect"))
		for _, hint := range sshHints {
			lines = append(lines, fmt.Sprintf("  %s %s", theme.Muted.Render("$"), theme.Value.Render(hint)))
		}
	}

	return writeBlock(w, theme, lines)
}

func renderStatus(w io.Writer, theme humanTheme, value app.StatusResult) error {
	instances := append([]app.StatusInstanceResult(nil), value.Instances...)
	sort.Slice(instances, func(i, j int) bool { return instances[i].Name < instances[j].Name })

	rows := [][]string{{"NAME", "STATUS", "SSH", "PORTS", "LAB IP"}}
	for _, instance := range instances {
		ssh := sshAddress(instance.ManagementIP, instance.SSHPort)
		labIP := "-"
		if instance.LabIP != "" {
			labIP = instance.LabIP
		}
		rows = append(rows, []string{instance.Name, instance.Status, ssh, renderPortURLs(instance.Ports), labIP})
	}

	lines := []string{theme.Header.Render("Status")}
	lines = append(lines, renderRows(theme, rows)...)

	if len(instances) == 0 {
		lines = append(lines, "  "+theme.Muted.Render("No instances. Run 'yeast up' to start."))
	}

	return writeBlock(w, theme, lines)
}

func renderPortURLs(ports []app.PortForwardResult) string {
	if len(ports) == 0 {
		return "-"
	}
	values := make([]string, 0, len(ports))
	for _, port := range ports {
		value := port.URL
		if value == "" {
			value = fmt.Sprintf("%s:%d", port.Host, port.HostPort)
		}
		values = append(values, value)
	}
	return strings.Join(values, ", ")
}

func renderProvision(w io.Writer, theme humanTheme, value app.ProvisionResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Header.Render("Provisioning finished"),
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
		theme.Success.Render("OK") + " " + theme.Header.Render("Command finished"),
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
		theme.Success.Render("OK") + " " + theme.Header.Render("Copy finished"),
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
		theme.Header.Render("Instance inspect"),
		keyValue(theme, "name", value.Instance.Name),
		keyValue(theme, "status", value.Instance.Status),
		keyValue(theme, "ssh", sshAddress(value.Instance.ManagementIP, value.Instance.SSHPort)),
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
		theme.Header.Render("Instance logs"),
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
		theme.Success.Render("OK") + " " + theme.Header.Render("Snapshot created"),
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
		theme.Success.Render("OK") + " " + theme.Header.Render("Snapshot restored"),
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
		theme.Header.Render("Instance snapshots"),
		keyValue(theme, "instance", value.Instance),
	}
	lines = append(lines, renderRows(theme, rows)...)
	return writeBlock(w, theme, lines)
}

func renderDeleteSnapshot(w io.Writer, theme humanTheme, value app.DeleteSnapshotResult) error {
	lines := []string{
		theme.Success.Render("OK") + " " + theme.Header.Render("Snapshot deleted"),
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

	lines := []string{theme.Success.Render("✓") + " " + theme.Header.Render("All instances stopped")}
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

	lines := []string{theme.Success.Render("✓") + " " + theme.Header.Render(title)}
	lines = append(lines, "")
	lines = append(lines, renderRows(theme, rows)...)
	return writeBlock(w, theme, lines)
}

func renderClean(w io.Writer, theme humanTheme, value app.CleanResult) error {
	rows := [][]string{{"NAME", "STATUS", "CLEANED PIDS"}}
	for _, instance := range value.Instances {
		rows = append(rows, []string{
			instance.Name,
			instance.Status,
			formatPIDs(instance.CleanedPIDs),
		})
	}

	lines := []string{theme.Success.Render("✓") + " " + theme.Header.Render("Project cleaned")}
	lines = append(lines, "")
	if len(value.Instances) == 0 {
		lines = append(lines, theme.Muted.Render("No instances needed cleanup."))
	} else {
		lines = append(lines, renderRows(theme, rows)...)
	}
	return writeBlock(w, theme, lines)
}

func renderImageClean(w io.Writer, theme humanTheme, value app.ImageCleanResult) error {
	if value.DryRun {
		lines := []string{theme.Header.Render("Would remove")}
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

	lines := []string{theme.Success.Render("✓") + " " + theme.Header.Render("Cached images removed")}
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

func formatPIDs(values []int) string {
	if len(values) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func sshAddress(host string, port int) string {
	if port <= 0 {
		return "-"
	}
	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", host, port)
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

func renderUpdate(w io.Writer, theme humanTheme, value app.UpdateResult) error {
	var lines []string

	switch {
	case value.Success:
		lines = []string{
			theme.Success.Render("✓") + " " + theme.Header.Render("Updated"),
			keyValue(theme, "from", value.CurrentVersion),
			keyValue(theme, "to", value.TargetVersion),
		}
		if value.ChecksumVerified {
			lines = append(lines, keyValue(theme, "checksum", theme.Success.Render("verified")))
		}
		if value.BinaryPath != "" {
			lines = append(lines, keyValue(theme, "binary", value.BinaryPath))
		}

	case value.AlreadyLatest:
		lines = []string{
			theme.Success.Render("✓") + " " + theme.Header.Render("Already up to date"),
			keyValue(theme, "version", value.CurrentVersion),
		}

	case value.CheckOnly && value.UpdateAvailable:
		lines = []string{
			theme.Warning.Render("!") + " " + theme.Header.Render("Update available"),
			keyValue(theme, "current", value.CurrentVersion),
			keyValue(theme, "latest", value.TargetVersion),
			"",
			"  " + theme.Muted.Render("Run 'yeast update' to install."),
		}

	case value.CheckOnly && !value.UpdateAvailable:
		lines = []string{
			theme.Success.Render("✓") + " " + theme.Header.Render("Up to date"),
			keyValue(theme, "version", value.CurrentVersion),
		}

	default:
		lines = []string{
			theme.Header.Render("Update"),
			keyValue(theme, "current", value.CurrentVersion),
			keyValue(theme, "target", value.TargetVersion),
		}
	}

	return writeBlock(w, theme, lines)
}

func renderUpdateNotice(w io.Writer, theme humanTheme, value app.UpdateNotice) error {
	lines := []string{
		theme.Warning.Render("!") + " " + theme.Header.Render("Update available"),
		keyValue(theme, "current", value.CurrentVersion),
		keyValue(theme, "latest", value.LatestVersion),
		"",
		"  " + theme.Muted.Render("Run 'yeast update' to install."),
	}
	return writeBlock(w, theme, lines)
}
