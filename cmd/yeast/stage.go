package main

import (
	"fmt"
	"io"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/ui"
	"github.com/Twarga/yeast/internal/ui/sprites"
)

// newProgressStage creates the animated Yeast Cell stage plus the event sink
// feeding it. Returns (nil, nil) whenever machine-grade output is active —
// --json, --events, and --quiet suppress all progress rendering.
func newProgressStage(w io.Writer, mood sprites.Mood, title string) (*ui.LiveStage, app.EventSink) {
	if outputQuiet || outputJSON || outputEvents {
		return nil, nil
	}
	stage := ui.NewLiveStage(w, ui.LiveOptions{
		Title:    title,
		NoMotion: outputNoMotion,
		Plain:    outputPlain,
	})
	stage.Start(mood)
	return stage, func(ev app.Event) { applyStageEvent(stage, ev) }
}

// stageStop is a nil-safe deferred cleanup: restores the cursor if the
// command path exits without Finish (error returns, panics re-raised later).
func stageStop(stage *ui.LiveStage) {
	if stage != nil {
		stage.Stop()
	}
}

// stageFinish settles the stage on the final mood for a command result.
func stageFinish(stage *ui.LiveStage, err error) {
	if stage == nil {
		return
	}
	if err != nil {
		stage.Finish(sprites.MoodError, "")
		return
	}
	stage.Finish(sprites.MoodSuccess, "")
}

// renderStatusSprite prints the Yeast Cell still beside the status table —
// MoodIdle when anything hums, MoodSleep (nightcap) when all rest.
func renderStatusSprite(w io.Writer, result app.StatusResult) {
	if outputQuiet || outputJSON || outputEvents || outputPlain || !ui.StylingEnabled(w) {
		return
	}
	if ui.TierFor(ui.WidthFor(w)) != ui.WidthFull {
		return
	}
	mood := sprites.MoodSleep
	for _, inst := range result.Instances {
		if inst.Status == "running" {
			mood = sprites.MoodIdle
			break
		}
	}
	depth := ui.DepthFor(w)
	if depth == sprites.DepthNone {
		return
	}
	for _, line := range sprites.RenderStill(mood, depth) {
		fmt.Fprintln(w, line)
	}
}

// applyStageEvent maps engine lifecycle events onto live stage steps.
func applyStageEvent(stage *ui.LiveStage, ev app.Event) {
	if stage == nil {
		return
	}
	detail := ev.Instance
	if detail == "" {
		detail = ev.Message
	}
	step := func(id, label string, state ui.StepState) {
		stage.SetStep(id, label, state, detail)
	}
	switch ev.Name {
	case app.EventProjectLoaded:
		step("config", "Load project", ui.StepDone)
	case app.EventConfigValidated:
		step("config", "Validate config", ui.StepDone)
	case app.EventImagePulling:
		step("image", "Pull base image", ui.StepRunning)
	case app.EventImageReady:
		step("image", "Pull base image", ui.StepDone)
	case app.EventDiskReady:
		step("disk", "Prepare disks", ui.StepDone)
	case app.EventCloudInitGenerated:
		step("cloud-init", "Generate cloud-init", ui.StepDone)
	case app.EventVMStarting:
		step("vm", "Boot VM", ui.StepRunning)
	case app.EventSSHWaiting:
		step("ssh", "Wait for SSH", ui.StepRunning)
	case app.EventSSHReady:
		step("ssh", "Wait for SSH", ui.StepDone)
	case app.EventProvisionStarted:
		step("provision", "Provision", ui.StepRunning)
	case app.EventProvisionFinished:
		step("provision", "Provision", ui.StepDone)
	case app.EventProvisionSkipped:
		step("provision", "Provision", ui.StepSkipped)
	case app.EventSnapshotCreated:
		step("snapshot", "Create snapshot", ui.StepDone)
	case app.EventRestoreStarted:
		step("restore", "Restore snapshot", ui.StepRunning)
	case app.EventRestoreFinished:
		step("restore", "Restore snapshot", ui.StepDone)
	case app.EventClockResynced:
		step("clock", "Resync guest clock", ui.StepDone)
	case app.EventForkCreated:
		step("fork."+ev.Instance, "Fork "+ev.Instance, ui.StepDone)
	case app.EventInstanceReady:
		step("instance."+ev.Instance, "Instance "+ev.Instance+" ready", ui.StepDone)
	case app.EventInstanceStopped:
		step("instance."+ev.Instance, "Instance "+ev.Instance+" stopped", ui.StepDone)
	case app.EventInstanceDestroyed:
		step("instance."+ev.Instance, "Instance "+ev.Instance+" destroyed", ui.StepDone)
	case app.EventWorkflowCompleted:
		stage.SetMood(sprites.MoodSuccess)
	case app.EventWorkflowFailed:
		stage.SetMood(sprites.MoodError)
	}
}
