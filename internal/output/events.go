package output

import (
	"encoding/json"
	"io"
	"yeast/internal/app"
)

func RenderJSONEvent(w io.Writer, event app.Event) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(event)
}
