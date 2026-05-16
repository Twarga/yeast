package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrProjectIDMismatch = errors.New("state project id mismatch")

func Load(path string, projectID string) (State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return New(projectID), nil
		}
		return State{}, fmt.Errorf("read state file %s: %w", path, err)
	}

	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, fmt.Errorf("parse state file %s: %w", path, err)
	}
	if state.Schema != Schema {
		return State{}, fmt.Errorf("invalid state file %s: unsupported schema %q", path, state.Schema)
	}
	if state.ProjectID != projectID {
		return State{}, fmt.Errorf("%w: state file %s belongs to %q, expected %q", ErrProjectIDMismatch, path, state.ProjectID, projectID)
	}
	if state.Instances == nil {
		state.Instances = make(map[string]InstanceState)
	}

	return state, nil
}

func Save(path string, state State) error {
	if state.Schema != Schema {
		return fmt.Errorf("invalid state schema %q, expected %q", state.Schema, Schema)
	}
	if state.ProjectID == "" {
		return fmt.Errorf("state project id is required")
	}
	if state.Instances == nil {
		state.Instances = make(map[string]InstanceState)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state directory %s: %w", dir, err)
	}

	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state file %s: %w", path, err)
	}
	raw = append(raw, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0644); err != nil {
		return fmt.Errorf("write state temp file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save state file %s: %w", path, err)
	}

	return nil
}

