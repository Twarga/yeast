package main

import (
	"fmt"
	"io"
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
