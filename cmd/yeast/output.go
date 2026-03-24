package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const commandEnvelopeSchema = "yeast.command.v1"

type commandEnvelope struct {
	Schema  string        `json:"schema"`
	Command string        `json:"command"`
	OK      bool          `json:"ok"`
	Data    any           `json:"data,omitempty"`
	Error   *commandError `json:"error,omitempty"`
}

type commandError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type reportedJSONError struct {
	err error
}

func (e *reportedJSONError) Error() string {
	if e == nil || e.err == nil {
		return "command failed"
	}
	return e.err.Error()
}

func (e *reportedJSONError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func emitJSON(command string, ok bool, data any, errObj *commandError) error {
	env := commandEnvelope{
		Schema:  commandEnvelopeSchema,
		Command: command,
		OK:      ok,
		Data:    data,
		Error:   errObj,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

func jsonCommandSuccess(command string, data any) error {
	if !outputJSON {
		return nil
	}
	return emitJSON(command, true, data, nil)
}

func jsonCommandError(command, code string, err error) error {
	return jsonCommandErrorWithData(command, code, err, nil)
}

func jsonCommandErrorWithData(command, code string, err error, data any) error {
	if !outputJSON {
		return err
	}

	if err == nil {
		err = errors.New("command failed")
	}

	if emitErr := emitJSON(command, false, data, &commandError{
		Code:    code,
		Message: err.Error(),
	}); emitErr != nil {
		return fmt.Errorf("failed to emit JSON output: %w", emitErr)
	}

	return &reportedJSONError{err: err}
}

type initCommandData struct {
	Schema     string `json:"schema"`
	ConfigPath string `json:"config_path"`
	Created    bool   `json:"created"`
}

type statusCommandData struct {
	Schema        string           `json:"schema"`
	StatePath     string           `json:"state_path"`
	InstanceCount int              `json:"instance_count"`
	Instances     []instanceStatus `json:"instances"`
}

type instanceStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	PID     int    `json:"pid"`
	IP      string `json:"ip"`
	SSHPort int    `json:"ssh_port"`
}

type lifecycleResult struct {
	Name    string `json:"name"`
	Action  string `json:"action"`
	Message string `json:"message,omitempty"`
	PID     int    `json:"pid,omitempty"`
	SSHPort int    `json:"ssh_port,omitempty"`
}

type upCommandData struct {
	Schema         string            `json:"schema"`
	Results        []lifecycleResult `json:"results"`
	Started        int               `json:"started"`
	AlreadyRunning int               `json:"already_running"`
	Failed         int               `json:"failed"`
}

type stopCommandData struct {
	Schema         string            `json:"schema"`
	Results        []lifecycleResult `json:"results"`
	Stopped        int               `json:"stopped"`
	AlreadyStopped int               `json:"already_stopped"`
	Absent         int               `json:"absent"`
	Failed         int               `json:"failed"`
}

type restartCommandData struct {
	Schema     string            `json:"schema"`
	Results    []lifecycleResult `json:"results"`
	Restarted  int               `json:"restarted"`
	NotDefined int               `json:"not_defined"`
	Failed     int               `json:"failed"`
}

type destroyCommandData struct {
	Schema    string            `json:"schema"`
	Results   []lifecycleResult `json:"results"`
	Destroyed int               `json:"destroyed"`
	Absent    int               `json:"absent"`
	Failed    int               `json:"failed"`
}

type pullCommandData struct {
	Schema      string `json:"schema"`
	Image       string `json:"image"`
	SourceURL   string `json:"source_url"`
	SHA256      string `json:"sha256"`
	Destination string `json:"destination"`
	Action      string `json:"action"`
}

type pullListImage struct {
	Name      string `json:"name"`
	SourceURL string `json:"source_url"`
	SHA256    string `json:"sha256"`
}

type pullListCommandData struct {
	Schema string          `json:"schema"`
	Count  int             `json:"count"`
	Images []pullListImage `json:"images"`
}

type doctorCheckOutput struct {
	Name    string   `json:"name"`
	Level   string   `json:"level"`
	Message string   `json:"message"`
	Fixes   []string `json:"fixes,omitempty"`
}

type doctorCommandData struct {
	Schema   string              `json:"schema"`
	Checks   []doctorCheckOutput `json:"checks"`
	Total    int                 `json:"total"`
	Blockers int                 `json:"blockers"`
	Warnings int                 `json:"warnings"`
}
