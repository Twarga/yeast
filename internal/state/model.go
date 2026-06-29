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
}

func New(projectID string) State {
	return State{
		Schema:    Schema,
		ProjectID: projectID,
		Instances: make(map[string]InstanceState),
	}
}
