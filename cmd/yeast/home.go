package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/ui"
	"github.com/Twarga/yeast/internal/ui/sprites"
)

// renderHome draws the Yeast home screen when invoked bare on an interactive
// terminal: the Yeast Cell beside a project summary and next-step suggestions.
// Returns false when the environment can't host it — caller falls back to help.
func renderHome(w io.Writer, service *app.Service) bool {
	if outputJSON || outputEvents || outputQuiet || outputPlain || !ui.StylingEnabled(w) {
		return false
	}
	if ui.TierFor(ui.WidthFor(w)) == ui.WidthNarrow {
		return false
	}
	if outputQuirk {
		renderQuirkDemo(w)
	}

	theme := ui.NewTheme(w)
	summary, suggestions, mood := homeSummary(service)
	if outputQuirk {
		mood = sprites.MoodQuirk
	}

	panel := strings.Join([]string{
		theme.Label.Render("yeast"),
		theme.Muted.Render(ui.Voice.HomeTag),
		"",
		theme.Text.Render(summary),
		"",
		theme.Header.Render("Try"),
		"  " + theme.AccentSoft.Render(suggestions[0]),
		"  " + theme.AccentSoft.Render(suggestions[1]),
		"  " + theme.AccentSoft.Render(suggestions[2]),
		"",
		theme.Muted.Render(ui.Voice.OneMore),
	}, "\n")

	depth := ui.DepthFor(w)
	sprite := strings.Join(sprites.RenderStill(mood, depth), "\n")
	var frame string
	if ui.TierFor(ui.WidthFor(w)) == ui.WidthFull && depth != sprites.DepthNone {
		frame = lipgloss.JoinHorizontal(lipgloss.Top, sprite, "   ", panel)
	} else {
		frame = panel
	}
	fmt.Fprintln(w, frame)
	return true
}

// renderQuirkDemo plays the hiccup-blush quirk loop once (~1.4s) for the
// `yeast --quirk` easter egg.
func renderQuirkDemo(w io.Writer) {
	stage := ui.NewLiveStage(w, ui.LiveOptions{NoMotion: outputNoMotion})
	stage.Start(sprites.MoodQuirk)
	time.Sleep(1400 * time.Millisecond)
	stage.Stop()
}

// homeSummary builds the status line and three smart next commands.
func homeSummary(service *app.Service) (string, [3]string, sprites.Mood) {
	result, err := service.Status(context.Background(), app.StatusOptions{})
	if err != nil || len(result.Instances) == 0 {
		return "No machines here yet.", [3]string{"yeast init", "yeast doctor", "yeast up"}, sprites.MoodSleep
	}
	running := 0
	first := ""
	for _, inst := range result.Instances {
		if inst.Status == "running" {
			running++
			if first == "" {
				first = inst.Name
			}
		}
	}
	total := len(result.Instances)
	if running == 0 {
		return fmt.Sprintf("%d machine(s), all resting.", total),
			[3]string{"yeast up", "yeast status", "yeast doctor"}, sprites.MoodSleep
	}
	return fmt.Sprintf("%d of %d machine(s) humming.", running, total),
		[3]string{"yeast status", "yeast ssh " + first, "yeast down"}, sprites.MoodIdle
}
