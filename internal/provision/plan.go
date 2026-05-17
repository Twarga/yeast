package provision

import "yeast/internal/config"

type StepOrigin string

const (
	OriginProject  StepOrigin = "project"
	OriginInstance StepOrigin = "instance"
)

type Plan struct {
	InstanceName string
	Packages     []PackageStep
	Files        []FileStep
	Shell        []ShellStep
}

type PackageStep struct {
	Name   string
	Origin StepOrigin
}

type FileStep struct {
	Source      string
	Destination string
	Permissions string
	Origin      StepOrigin
}

type ShellStep struct {
	Command string
	Origin  StepOrigin
}

func BuildPlan(instance config.Instance, projectProvision *config.ProvisionConfig) Plan {
	plan := Plan{
		InstanceName: instance.Name,
	}

	appendProvision(&plan, OriginProject, projectProvision)
	appendProvision(&plan, OriginInstance, instance.Provision)

	return plan
}

func (p Plan) Empty() bool {
	return len(p.Packages) == 0 && len(p.Files) == 0 && len(p.Shell) == 0
}

func appendProvision(plan *Plan, origin StepOrigin, provision *config.ProvisionConfig) {
	if plan == nil || provision == nil {
		return
	}

	for _, pkg := range provision.Packages {
		plan.Packages = append(plan.Packages, PackageStep{
			Name:   pkg,
			Origin: origin,
		})
	}

	for _, file := range provision.Files {
		plan.Files = append(plan.Files, FileStep{
			Source:      file.Source,
			Destination: file.Destination,
			Permissions: file.Permissions,
			Origin:      origin,
		})
	}

	for _, command := range provision.Shell {
		plan.Shell = append(plan.Shell, ShellStep{
			Command: command,
			Origin:  origin,
		})
	}
}
