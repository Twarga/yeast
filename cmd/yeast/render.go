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

func renderCommandOutputWithTiming(w io.Writer, command string, data any, elapsed time.Duration) error {
	if err := renderCommandOutput(w, command, data); err != nil {
		return err
	}
	if !outputJSON {
		fmt.Fprintf(w, "\n  Done in %s\n", formatDuration(elapsed))
	}
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if seconds == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds)
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
	if !outputJSON {
		return nil, fmt.Errorf("--events requires --json")
	}
	return func(event app.Event) {
		_ = output.RenderJSONEvent(w, event)
	}, nil
}
