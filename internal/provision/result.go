package provision

import (
	"time"
	"yeast/internal/state"
)

type StepKind string

const (
	StepKindPackage StepKind = "package"
	StepKindFile    StepKind = "file"
	StepKindShell   StepKind = "shell"
)

type StepResult struct {
	Kind        StepKind
	Description string
	StartedAt   time.Time
	FinishedAt  time.Time
	ExitCode    int
	Stdout      string
	Stderr      string
	Err         string
}

type Result struct {
	Status  state.ProvisioningStatus
	LogPath string
	Steps   []StepResult
}

func NewResult(logPath string) Result {
	return Result{
		Status:  state.ProvisioningStatusNotStarted,
		LogPath: logPath,
		Steps:   make([]StepResult, 0),
	}
}
