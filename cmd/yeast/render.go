package main

import (
	"fmt"
	"io"
	"time"
	"yeast/internal/app"
	"yeast/internal/output"
)

func renderCommandOutput(w io.Writer, command string, data any) error {
	if outputJSON {
		return output.RenderJSONSuccess(w, command, data)
	}
	return output.RenderHuman(w, command, data)
}

func renderCommandError(w io.Writer, err error) error {
	if !outputJSON {
		return err
	}
	if renderErr := output.RenderJSONError(w, err); renderErr != nil {
		return fmt.Errorf("render json error: %w", renderErr)
	}
	return nil
}

func eventSink(w io.Writer) (app.EventSink, error) {
	if !outputEvents {
		return nil, nil
	}
	return func(event app.Event) {
		_ = output.RenderJSONEvent(w, event)
	}, nil
}

func commandEventSink(jsonWriter, humanWriter io.Writer) (app.EventSink, error) {
	if outputEvents {
		if outputJSON {
			return eventSink(jsonWriter)
		}
		return eventSink(humanWriter)
	}
	if outputJSON {
		return nil, nil
	}
	return humanLifecycleProgress(humanWriter), nil
}

func humanLifecycleProgress(w io.Writer) app.EventSink {
	started := time.Now()
	return func(event app.Event) {
		label := humanEventLabel(event)
		if label == "" {
			return
		}
		instance := ""
		if event.Instance != "" {
			instance = " " + event.Instance
		}
		_, _ = fmt.Fprintf(w, "yeast %s: %s%s (%s)\n", event.Command, label, instance, time.Since(started).Round(time.Second))
	}
}

func humanEventLabel(event app.Event) string {
	switch event.Name {
	case app.EventProjectLoaded:
		return "loaded project"
	case app.EventConfigValidated:
		return "validated config"
	case app.EventImageReady:
		return "image ready"
	case app.EventDiskReady:
		return "disk ready"
	case app.EventVMStarting:
		return "starting vm"
	case app.EventSSHReady:
		return "ssh ready"
	case app.EventProvisionStarted:
		return "provisioning"
	case app.EventProvisionFinished:
		return "provisioned"
	case app.EventInstanceReady:
		return "instance ready"
	case app.EventInstanceStopped:
		return "stopped"
	case app.EventInstanceDestroyed:
		return "destroyed"
	case app.EventWorkflowCompleted:
		return "done"
	default:
		return ""
	}
}
