package ssh

import (
	"context"
	"fmt"
	"strings"
	"time"
	"yeast/internal/provision"
)

const DefaultShellTimeout = 10 * time.Minute

type ShellRequest struct {
	User     string
	Host     string
	Port     int
	Commands []provision.ShellStep
	Timeout  time.Duration
}

type ShellStepResult struct {
	Command string
	Run     RunResult
}

type ShellResult struct {
	Steps []ShellStepResult
}

type ShellProvisioner struct {
	transport Transport
}

func NewShellProvisioner(transport Transport) *ShellProvisioner {
	return &ShellProvisioner{transport: transport}
}

func (p *ShellProvisioner) Run(ctx context.Context, request ShellRequest) (ShellResult, error) {
	commands, err := normalizeShellSteps(request.Commands)
	if err != nil {
		return ShellResult{}, err
	}
	if len(commands) == 0 {
		return ShellResult{}, nil
	}
	if request.User == "" {
		return ShellResult{}, fmt.Errorf("user is required")
	}
	if request.Host == "" {
		return ShellResult{}, fmt.Errorf("host is required")
	}
	if request.Port <= 0 {
		return ShellResult{}, fmt.Errorf("port must be greater than zero")
	}

	transport := p.transport
	if transport == nil {
		transport = NewLocalTransport()
	}

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = DefaultShellTimeout
	}

	result := ShellResult{
		Steps: make([]ShellStepResult, 0, len(commands)),
	}

	for index, command := range commands {
		runResult, err := transport.Run(ctx, RunRequest{
			User:    request.User,
			Host:    request.Host,
			Port:    request.Port,
			Command: command.Command,
			Timeout: timeout,
		})

		stepResult := ShellStepResult{
			Command: command.Command,
			Run:     runResult,
		}
		result.Steps = append(result.Steps, stepResult)

		if err != nil {
			return result, fmt.Errorf("shell step %d command %q failed: %w", index, command.Command, err)
		}
	}

	return result, nil
}

func normalizeShellSteps(steps []provision.ShellStep) ([]provision.ShellStep, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	commands := make([]provision.ShellStep, 0, len(steps))
	for i, step := range steps {
		command := strings.TrimSpace(step.Command)
		if command == "" {
			return nil, fmt.Errorf("shell step %d is empty", i)
		}
		commands = append(commands, provision.ShellStep{
			Command: command,
			Origin:  step.Origin,
		})
	}

	return commands, nil
}
