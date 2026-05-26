package main

import (
	"fmt"
	"io"
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
	if !outputJSON {
		return nil, fmt.Errorf("--events requires --json")
	}
	return func(event app.Event) {
		_ = output.RenderJSONEvent(w, event)
	}, nil
}
