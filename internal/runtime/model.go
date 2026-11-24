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
	Firmware      string       // "bios" (default) | "uefi"
	Display       *DisplayPlan // nil or Enabled=false → headless -nographic
	Mounts        []MountPlan  // host→guest shared directories
	// IncomingDefer boots QEMU paused waiting for a migration stream
	// (-incoming defer) — used by live-snapshot restore.
	IncomingDefer bool
	// Egress controls guest outbound networking; nil = open NAT.
	Egress *EgressPlan
	// BootISOPath boots the VM directly from a local .iso as the CD-ROM
	// (-boot d) — used by `yeast boot-iso` for ISO boot-testing. Mutually
	// exclusive with SeedImagePath.
	BootISOPath string
}

// DisplayPlan describes a graphical display for one machine.
// All fields are normalized (defaults applied) before reaching the runtime.
type DisplayPlan struct {
	Type       string // "vnc"
	Host       string
	Port       int    // VNC TCP port; QEMU display index = Port - 5900
	GPU        string // "virtio-vga" | "virtio-gpu" | "virgl"
	Width      int    // 0 = guest default
	Height     int
	RenderNode string // DRM render node for virgl (e.g. /dev/dri/renderD128)
}

// MountPlan describes one host directory shared into the guest.
type MountPlan struct {
	Source   string // absolute host path
	Target   string // absolute guest path
	ReadOnly bool
	Driver   string // resolved: "virtiofs" | "9p"
	Tag      string // mount tag, e.g. yeast_mnt_0
	// DaemonPath is the resolved virtiofsd binary (virtiofs mounts only).
	DaemonPath string
}

// DisplayInfo is the runtime-resolved view of a machine display.
type DisplayInfo struct {
	Type     string
	Host     string
	Port     int
	URL      string // vnc://host:port
	GPU      string
	Firmware string
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
	PortForwards  []PortForwardPlan
}

type PortForwardPlan struct {
	Name      string
	Host      string
	HostPort  int
	GuestPort int
	Protocol  string
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
	// Kind selects the snapshot implementation: "copy" (legacy full-copy),
	// "overlay" (instant stopped-VM qcow2 chain). Live RAM+disk snapshots go
	// through LiveSnapshotter, not this method.
	Kind string
	// RuntimeDir locates the instance's active disk for chain-aware
	// operations (overlay create/restore, rebase-safe delete).
	RuntimeDir string
}

// DeleteSnapshotPlan carries everything needed to remove a snapshot without
// breaking qcow2 backing chains.
type DeleteSnapshotPlan struct {
	Path       string // copy: the .qcow2 file; overlay/live: <snapDir>/base.qcow2
	Kind       string
	RuntimeDir string
	// ActiveDiskPath is the instance's current active disk when it differs
	// from <RuntimeDir>/disk.qcow2 (post-live-snapshot overlay).
	ActiveDiskPath string
}

// LiveSnapshotPlan drives a running-VM RAM+disk checkpoint.
type LiveSnapshotPlan struct {
	Instance    RuntimeInstance
	SnapshotDir string // <instance>/snapshots/<name>
	MemoryMiB   int
	Plan        MachinePlan // persisted to plan.json for live restore
}

type LiveSnapshotInfo struct {
	Method  string // "mapped-ram" | "multifd" | "exec" | "snapshot-save"
	PauseMs int64  // guest-visible freeze window (stop → cont)
}

// LiveRestorePlan respawns an instance from a live checkpoint.
type LiveRestorePlan struct {
	Instance    RuntimeInstance
	SnapshotDir string
}

type RuntimeInstance struct {
	Name       string
	RuntimeDir string
	LogPath    string
	PID        int
	Networks   NetworkPlan
	Display    *DisplayInfo
	MountPIDs  []int // sidecar pids (virtiofsd) to reap on Stop/Destroy
	StartedAt  time.Time
	// ActiveDisk is the file QEMU currently writes to when it is not the
	// conventional <RuntimeDir>/disk.qcow2 (post-live-snapshot overlay).
	ActiveDisk string
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

// EgressPlan controls outbound network access for the management NIC.
type EgressPlan struct {
	// Mode: "open" (default, no restrict), "offline" (restrict=on),
	// "allowlist" (restrict=on + guestfwd to local allowlist proxy).
	Mode string
	// ProxyPort is the host-side port of the yeast allowlist proxy
	// (allowlist mode only). QEMU forwards guest 10.0.2.100:3128 here.
	ProxyPort int
}
