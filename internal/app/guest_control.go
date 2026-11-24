package app

import (
	"path/filepath"
	"strings"
	"time"
)

type GuestTargetOptions struct {
	ProjectRoot string
	Target      string
}

type GuestCommandResult struct {
	Command    string        `json:"command"`
	ExitCode   int           `json:"exit_code"`
	Stdout     string        `json:"stdout"`
	Stderr     string        `json:"stderr"`
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt time.Time     `json:"finished_at"`
	Duration   time.Duration `json:"duration"`
	TimedOut   bool          `json:"timed_out"`
	// Truncated is true when stdout and/or stderr exceeded the agent-safe
	// output cap; the untruncated payload is at FullOutputPath.
	Truncated      bool   `json:"truncated,omitempty"`
	FullOutputPath string `json:"full_output_path,omitempty"`
}

type ExecOptions struct {
	GuestTargetOptions
	Command []string
	Timeout time.Duration
	Cwd     string
	Env     map[string]string
	User    string
	Stdin   []byte
}

type ExecResult struct {
	ProjectID string             `json:"project_id"`
	Instance  string             `json:"instance"`
	Run       GuestCommandResult `json:"run"`
}

type CopyDirection string

const (
	CopyToGuest   CopyDirection = "to_guest"
	CopyFromGuest CopyDirection = "from_guest"
)

type CopyOptions struct {
	GuestTargetOptions
	Direction   CopyDirection
	Source      string
	Destination string
	Timeout     time.Duration
}

type CopyResult struct {
	ProjectID   string        `json:"project_id"`
	Instance    string        `json:"instance"`
	Direction   CopyDirection `json:"direction"`
	Source      string        `json:"source"`
	Destination string        `json:"destination"`
	StartedAt   time.Time     `json:"started_at"`
	FinishedAt  time.Time     `json:"finished_at"`
	Duration    time.Duration `json:"duration"`
}

type InspectOptions struct {
	GuestTargetOptions
}

type InspectResult struct {
	ProjectID     string               `json:"project_id"`
	Instance      StatusInstanceResult `json:"instance"`
	SnapshotNames []string             `json:"snapshot_names"`
	SnapshotCount int                  `json:"snapshot_count"`
}

type LogsOptions struct {
	GuestTargetOptions
	TailLines int
	// Network reads the egress-proxy decision log (network.log) instead of
	// the VM console log — populated when egress.mode=allowlist.
	Network bool
}

type LogsResult struct {
	ProjectID string `json:"project_id"`
	Instance  string `json:"instance"`
	LogPath   string `json:"log_path"`
	Content   string `json:"content"`
}

func commandString(parts []string) string {
	return strings.Join(parts, " ")
}

func cleanLocalPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
