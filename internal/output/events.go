package output

import (
	"encoding/json"
	"github.com/Twarga/yeast/internal/app"
	"io"
)

func RenderJSONEvent(w io.Writer, event app.Event) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(event)
}
