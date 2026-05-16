package cloudinit

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type MetaDataInput struct {
	Hostname string
}

type metaData struct {
	InstanceID    string `yaml:"instance-id"`
	LocalHostname string `yaml:"local-hostname"`
}

func RenderMetaData(input MetaDataInput) (string, error) {
	hostname := strings.TrimSpace(input.Hostname)
	if hostname == "" {
		return "", fmt.Errorf("hostname is required")
	}

	body, err := yaml.Marshal(metaData{
		InstanceID:    hostname,
		LocalHostname: hostname,
	})
	if err != nil {
		return "", fmt.Errorf("marshal cloud-init meta-data: %w", err)
	}
	return string(body), nil
}
