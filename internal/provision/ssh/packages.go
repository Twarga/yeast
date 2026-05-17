package ssh

import (
	"context"
	"fmt"
	"strings"
	"time"
	"yeast/internal/provision"
)

const DefaultPackageTimeout = 15 * time.Minute

type PackageRequest struct {
	User     string
	Host     string
	Port     int
	Packages []provision.PackageStep
	Timeout  time.Duration
}

type PackageResult struct {
	Packages []string
	Command  string
	Run      RunResult
}

type PackageProvisioner struct {
	transport Transport
}

func NewPackageProvisioner(transport Transport) *PackageProvisioner {
	return &PackageProvisioner{transport: transport}
}

func (p *PackageProvisioner) Install(ctx context.Context, request PackageRequest) (PackageResult, error) {
	packages, err := normalizePackageSteps(request.Packages)
	if err != nil {
		return PackageResult{}, err
	}
	if len(packages) == 0 {
		return PackageResult{}, nil
	}
	if request.User == "" {
		return PackageResult{}, fmt.Errorf("user is required")
	}
	if request.Host == "" {
		return PackageResult{}, fmt.Errorf("host is required")
	}
	if request.Port <= 0 {
		return PackageResult{}, fmt.Errorf("port must be greater than zero")
	}

	transport := p.transport
	if transport == nil {
		transport = NewLocalTransport()
	}

	command := buildAPTInstallCommand(packages)
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = DefaultPackageTimeout
	}

	runResult, err := transport.Run(ctx, RunRequest{
		User:    request.User,
		Host:    request.Host,
		Port:    request.Port,
		Command: command,
		Timeout: timeout,
	})

	result := PackageResult{
		Packages: packages,
		Command:  command,
		Run:      runResult,
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func buildAPTInstallCommand(packages []string) string {
	return "sudo DEBIAN_FRONTEND=noninteractive apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y " + strings.Join(packages, " ")
}

func normalizePackageSteps(steps []provision.PackageStep) ([]string, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	packages := make([]string, 0, len(steps))
	for i, step := range steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			return nil, fmt.Errorf("package step %d is empty", i)
		}
		if strings.ContainsAny(name, "\r\n\t ") {
			return nil, fmt.Errorf("package step %d has invalid package name %q", i, step.Name)
		}
		packages = append(packages, name)
	}

	return packages, nil
}
