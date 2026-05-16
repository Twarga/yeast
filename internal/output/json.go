package output

import (
	"encoding/json"
	"io"
	"yeast/internal/app"
)

func RenderJSONSuccess(w io.Writer, command string, data any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(SuccessEnvelope{
		OK:      true,
		Command: command,
		Data:    data,
	})
}

func RenderJSONError(w io.Writer, err error) error {
	normalized := app.NormalizeError(err)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(ErrorEnvelope{
		OK: false,
		Error: ErrorBody{
			Code:    string(normalized.Code),
			Message: normalized.Message,
			Details: normalized.Details,
		},
	})
}
