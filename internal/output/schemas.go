package output

const SchemaVersion = "yeast.v1"

type SuccessEnvelope struct {
	OK            bool   `json:"ok"`
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Data          any    `json:"data,omitempty"`
}

type ErrorEnvelope struct {
	OK            bool      `json:"ok"`
	SchemaVersion string    `json:"schema_version"`
	Error         ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
