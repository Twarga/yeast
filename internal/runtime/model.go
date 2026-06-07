package runtime

import (
	"net/netip"
	"time"
)

type MachinePlan struct {
	Name          string
	RuntimeDir    string
	LogPath       string
	MemoryMiB     int
	CPUs          int
	Disk          DiskPlan
	SeedImagePath string
	Networks      NetworkPlan
}

type DiskPlan struct {
	BaseImagePath string
	DiskPath      string
	Size          string
}

type NetworkPlan struct {
	Management ManagementNetworkPlan
	Lab        *LabNetworkPlan
}

type ManagementNetworkPlan struct {
	SSHHost       string
	SSHPort       int
	InterfaceName string
	MACAddress    string
}

type LabNetworkPlan struct {
	Name          string
	CIDR          netip.Prefix
	IPv4          netip.Addr
	InterfaceName string
	MACAddress    string
}

type SnapshotPlan struct {
	InstanceDiskPath string
	SnapshotPath     string
}

type RuntimeInstance struct {
	Name       string
	RuntimeDir string
	LogPath    string
	PID        int
	Networks   NetworkPlan
	StartedAt  time.Time
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

type CleanupTarget struct {
	Name       string
	RuntimeDir string
	SSHHost    string
	SSHPort    int
}

type CleanupResult struct {
	Name string
	PID  int
}
