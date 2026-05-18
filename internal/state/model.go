package state

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
	Status             string             `json:"status"`
	PID                int                `json:"pid,omitempty"`
	ManagementIP       string             `json:"management_ip,omitempty"`
	SSHPort            int                `json:"ssh_port,omitempty"`
	RuntimeDir         string             `json:"runtime_dir,omitempty"`
	ProvisionLogPath   string             `json:"provision_log_path,omitempty"`
	ProvisioningStatus ProvisioningStatus `json:"provisioning_status,omitempty"`
	LastError          string             `json:"last_error,omitempty"`
}

func New(projectID string) State {
	return State{
		Schema:    Schema,
		ProjectID: projectID,
		Instances: make(map[string]InstanceState),
	}
}
