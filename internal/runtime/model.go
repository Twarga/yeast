package runtime

import "time"

type MachinePlan struct {
	Name              string
	RuntimeDir        string
	LogPath           string
	MemoryMiB         int
	CPUs              int
	Disk              DiskPlan
	SeedImagePath     string
	ManagementNetwork NetworkOptions
}

type DiskPlan struct {
	BaseImagePath string
	DiskPath      string
	Size          string
}

type NetworkOptions struct {
	ManagementSSHPort int
}

type SnapshotPlan struct {
	InstanceDiskPath string
	SnapshotPath     string
}

type RuntimeInstance struct {
	Name              string
	RuntimeDir        string
	LogPath           string
	PID               int
	ManagementNetwork NetworkOptions
	StartedAt         time.Time
}

type ProcessState string

const (
	ProcessStateUnknown ProcessState = "unknown"
	ProcessStateRunning ProcessState = "running"
	ProcessStateStopped ProcessState = "stopped"
)

type ProcessInfo struct {
	PID       int
	State     ProcessState
	StartedAt time.Time
}
