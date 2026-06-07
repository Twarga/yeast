package app

import (
	"fmt"
	"path"
	"strings"
	"yeast/internal/config"
	"yeast/internal/provision"
)

func validateProvisionSudoPolicy(instance config.Instance, plan provision.Plan) error {
	if plan.Empty() || strings.TrimSpace(instance.Sudo) == "nopasswd" {
		return nil
	}

	var needs []string
	if len(plan.Packages) > 0 {
		needs = append(needs, "packages use sudo apt-get")
	}
	for _, file := range plan.Files {
		if destinationNeedsPrivilegedWrite(file.Destination, instance.User) {
			needs = append(needs, fmt.Sprintf("file %s writes outside the guest user's writable area", file.Destination))
			break
		}
	}
	for _, shell := range plan.Shell {
		if shellUsesSudo(shell.Command) {
			needs = append(needs, "shell commands use sudo")
			break
		}
	}
	if len(needs) == 0 {
		return nil
	}

	return fmt.Errorf(
		"instance %s provisioning needs passwordless sudo (%s), but sudo is %q; set `sudo: nopasswd` or use custom user_data/password handling",
		instance.Name,
		strings.Join(needs, ", "),
		instance.Sudo,
	)
}

func destinationNeedsPrivilegedWrite(destination, user string) bool {
	clean := path.Clean(strings.TrimSpace(destination))
	if clean == "." || !strings.HasPrefix(clean, "/") {
		return false
	}
	user = strings.TrimSpace(user)
	if user != "" && (clean == "/home/"+user || strings.HasPrefix(clean, "/home/"+user+"/")) {
		return false
	}
	return !(clean == "/tmp" || strings.HasPrefix(clean, "/tmp/"))
}

func shellUsesSudo(command string) bool {
	trimmed := strings.TrimSpace(command)
	return trimmed == "sudo" || strings.HasPrefix(trimmed, "sudo ") || strings.Contains(trimmed, " sudo ")
}
