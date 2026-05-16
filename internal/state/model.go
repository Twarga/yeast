package state

const Schema = "yeast.state.v1"

type State struct {
	Schema    string                   `json:"schema"`
	ProjectID string                   `json:"project_id"`
	Instances map[string]InstanceState `json:"instances"`
}

type InstanceState struct {
	Status             string `json:"status"`
	PID                int    `json:"pid,omitempty"`
	ManagementIP       string `json:"management_ip,omitempty"`
	SSHPort            int    `json:"ssh_port,omitempty"`
	RuntimeDir         string `json:"runtime_dir,omitempty"`
	ProvisioningStatus string `json:"provisioning_status,omitempty"`
	LastError          string `json:"last_error,omitempty"`
}

func New(projectID string) State {
	return State{
		Schema:    Schema,
		ProjectID: projectID,
		Instances: make(map[string]InstanceState),
	}
}
