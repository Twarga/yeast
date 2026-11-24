package state

import "time"

const Schema = "yeast.state.v1"

type ProvisioningStatus string

const (
	ProvisioningStatusNotStarted ProvisioningStatus = "not_started"
	ProvisioningStatusRunning    ProvisioningStatus = "running"
	ProvisioningStatusReady      ProvisioningStatus = "provisioned"
	ProvisioningStatusFailed     ProvisioningStatus = "failed"
)

type State struct {
	Schema    string                   `json:"schema"`
	ProjectID string                   `json:"project_id"`
	Instances map[string]InstanceState `json:"instances"`
}

type InstanceState struct {
	Status               string                   `json:"status"`
	PID                  int                      `json:"pid,omitempty"`
	ManagementIP         string                   `json:"management_ip,omitempty"`
	SSHPort              int                      `json:"ssh_port,omitempty"`
	ServicePorts         []PortForwardState       `json:"service_ports,omitempty"`
	User                 string                   `json:"user,omitempty"`
	LabIP                string                   `json:"lab_ip,omitempty"`
	RuntimeDir           string                   `json:"runtime_dir,omitempty"`
	Snapshots            map[string]SnapshotState `json:"snapshots,omitempty"`
	ProvisionLogPath     string                   `json:"provision_log_path,omitempty"`
	ProvisioningStatus   ProvisioningStatus       `json:"provisioning_status,omitempty"`
	ProvisionFingerprint string                   `json:"provision_fingerprint,omitempty"`
	LastError            string                   `json:"last_error,omitempty"`
	Display              *DisplayState            `json:"display,omitempty"`
	Mounts               []MountState             `json:"mounts,omitempty"`
	// ActiveDisk is the file QEMU writes to when it is not the conventional
	// <RuntimeDir>/disk.qcow2 (e.g. a live-snapshot overlay inside
	// snapshots/<name>/). Empty means disk.qcow2.
	ActiveDisk string `json:"active_disk,omitempty"`
	// ForkOf marks instances created by `yeast fork`.
	ForkOf string `json:"fork_of,omitempty"`
}

// DisplayState persists a running instance's graphical display endpoint.
type DisplayState struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	URL      string `json:"url"`
	GPU      string `json:"gpu,omitempty"`
	Firmware string `json:"firmware,omitempty"`
}

// MountState persists one shared host→guest directory.
type MountState struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Driver   string `json:"driver"`
	ReadOnly bool   `json:"readonly,omitempty"`
}

type PortForwardState struct {
	Name      string `json:"name,omitempty"`
	Host      string `json:"host"`
	HostPort  int    `json:"host_port"`
	GuestPort int    `json:"guest_port"`
	Protocol  string `json:"protocol"`
}

type SnapshotState struct {
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	Description    string    `json:"description,omitempty"`
	DiskPath       string    `json:"disk_path"`
	SourceDiskSize string    `json:"source_disk_size,omitempty"`
	// Kind: "" / "copy" (legacy full-copy), "overlay" (stopped qcow2 chain),
	// "live" (RAM+disk checkpoint). Empty means pre-v1.4 copy snapshots.
	Kind       string `json:"kind,omitempty"`
	Dir        string `json:"dir,omitempty"`         // snapshot directory (overlay/live)
	MemoryPath string `json:"memory_path,omitempty"` // live: guest RAM image
	Method     string `json:"method,omitempty"`      // live: migration method used
	PauseMs    int64  `json:"pause_ms,omitempty"`    // live: guest freeze window
	Bytes      int64  `json:"bytes,omitempty"`       // allocated size on disk (listing only)
}

func New(projectID string) State {
	return State{
		Schema:    Schema,
		ProjectID: projectID,
		Instances: make(map[string]InstanceState),
	}
}
