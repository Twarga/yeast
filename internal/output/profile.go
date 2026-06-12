package output

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type ProfilePhase struct {
	Name     string
	Started  time.Time
	Finished time.Time
}

type ProfileResult struct {
	Phases []ProfilePhase
	Total  time.Duration
}

func NewProfileResult() *ProfileResult {
	return &ProfileResult{}
}

func (p *ProfileResult) StartPhase(name string) {
	p.Phases = append(p.Phases, ProfilePhase{
		Name:    name,
		Started: time.Now(),
	})
}

func (p *ProfileResult) FinishCurrentPhase() {
	if len(p.Phases) == 0 {
		return
	}
	phase := &p.Phases[len(p.Phases)-1]
	if phase.Finished.IsZero() {
		phase.Finished = time.Now()
	}
}

func (p *ProfileResult) Finish() {
	p.FinishCurrentPhase()
	if len(p.Phases) > 0 {
		p.Total = p.Phases[len(p.Phases)-1].Finished.Sub(p.Phases[0].Started)
	}
}

func (p *ProfileResult) PhaseDurations() map[string]time.Duration {
	result := make(map[string]time.Duration, len(p.Phases))
	for _, phase := range p.Phases {
		if !phase.Finished.IsZero() {
			result[phase.Name] = phase.Finished.Sub(phase.Started)
		}
	}
	return result
}

func (p *ProfileResult) SlowestPhase() (string, time.Duration) {
	var slowest string
	var maxDuration time.Duration
	for _, phase := range p.Phases {
		if !phase.Finished.IsZero() {
			d := phase.Finished.Sub(phase.Started)
			if d > maxDuration {
				maxDuration = d
				slowest = phase.Name
			}
		}
	}
	return slowest, maxDuration
}

func RenderProfile(w io.Writer, result *ProfileResult) {
	if result == nil || len(result.Phases) == 0 {
		return
	}

	result.Finish()

	fmt.Fprintf(w, "\nBoot profile:\n")
	fmt.Fprintf(w, "─────────────────────────────────────────\n")

	durations := result.PhaseDurations()
	maxNameLen := 0
	for name := range durations {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	var totalWithProvision time.Duration
	for _, phase := range result.Phases {
		if !phase.Finished.IsZero() {
			d := phase.Finished.Sub(phase.Started)
			barLen := int(d.Seconds() * 2)
			if barLen > 40 {
				barLen = 40
			}
			if barLen < 1 && d > 0 {
				barLen = 1
			}
			bar := strings.Repeat("█", barLen)
			padding := strings.Repeat(" ", maxNameLen-len(phase.Name)+2)
			fmt.Fprintf(w, "  %s:%s%s  %s\n", phase.Name, padding, formatDuration(d), bar)
			totalWithProvision += d
		}
	}

	fmt.Fprintf(w, "─────────────────────────────────────────\n")
	fmt.Fprintf(w, "  total:         %s\n", formatDuration(result.Total))

	slowest, slowestDur := result.SlowestPhase()
	if slowest != "" && result.Total > 0 {
		pct := int(float64(slowestDur) / float64(result.Total) * 100)
		fmt.Fprintf(w, "  slowest phase: %s (%d%% of total)\n", slowest, pct)
	}

	suggestion := profileSuggestion(slowest, durations)
	if suggestion != "" {
		fmt.Fprintf(w, "\n  Suggestion: %s\n", suggestion)
	}
	fmt.Fprintf(w, "\n")
}

func profileSuggestion(slowestPhase string, durations map[string]time.Duration) string {
	switch slowestPhase {
	case "pkg_install":
		return "Use snapshots for faster resets to avoid re-installing packages."
	case "ssh_wait":
		return "Ensure KVM is enabled (check yeast doctor)."
	case "cloud_init":
		return "Consider custom user_data that skips apt-update."
	case "provision":
		return "Provisioning is the bottleneck. Consider reducing package count or using a pre-provisioned snapshot."
	default:
		return ""
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %02ds", mins, secs)
}
