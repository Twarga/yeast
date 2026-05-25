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
	Command    string
	ExitCode   int
	Stdout     string
	Stderr     string
	StartedAt  time.Time
	FinishedAt time.Time
	Duration   time.Duration
	TimedOut   bool
}

type ExecOptions struct {
	GuestTargetOptions
	Command []string
	Timeout time.Duration
}

type ExecResult struct {
	ProjectID string
	Instance  string
	Run       GuestCommandResult
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
	ProjectID   string
	Instance    string
	Direction   CopyDirection
	Source      string
	Destination string
	StartedAt   time.Time
	FinishedAt  time.Time
	Duration    time.Duration
}

type InspectOptions struct {
	GuestTargetOptions
}

type InspectResult struct {
	ProjectID     string
	Instance      StatusInstanceResult
	SnapshotNames []string
	SnapshotCount int
}

type LogsOptions struct {
	GuestTargetOptions
	TailLines int
}

type LogsResult struct {
	ProjectID string
	Instance  string
	LogPath   string
	Content   string
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
