package ssh

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"
	"yeast/internal/provision"
)

const DefaultFileTimeout = 5 * time.Minute

type FileRequest struct {
	User    string
	Host    string
	Port    int
	Files   []provision.FileStep
	Timeout time.Duration
}

type FileStepResult struct {
	Source      string
	Destination string
	Permissions string
	Mkdir       RunResult
	Upload      bool
	Chmod       RunResult
}

type FileResult struct {
	Files []FileStepResult
}

type FileProvisioner struct {
	transport Transport
}

func NewFileProvisioner(transport Transport) *FileProvisioner {
	return &FileProvisioner{transport: transport}
}

func (p *FileProvisioner) Upload(ctx context.Context, request FileRequest) (FileResult, error) {
	files, err := normalizeFileSteps(request.Files)
	if err != nil {
		return FileResult{}, err
	}
	if len(files) == 0 {
		return FileResult{}, nil
	}
	if request.User == "" {
		return FileResult{}, fmt.Errorf("user is required")
	}
	if request.Host == "" {
		return FileResult{}, fmt.Errorf("host is required")
	}
	if request.Port <= 0 {
		return FileResult{}, fmt.Errorf("port must be greater than zero")
	}

	transport := p.transport
	if transport == nil {
		transport = NewLocalTransport()
	}

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = DefaultFileTimeout
	}

	result := FileResult{
		Files: make([]FileStepResult, 0, len(files)),
	}

	for index, file := range files {
		stepResult := FileStepResult{
			Source:      file.Source,
			Destination: file.Destination,
			Permissions: file.Permissions,
		}

		mkdirCommand := buildMkdirCommand(file.Destination)
		mkdirResult, err := transport.Run(ctx, RunRequest{
			User:    request.User,
			Host:    request.Host,
			Port:    request.Port,
			Command: mkdirCommand,
			Timeout: timeout,
		})
		stepResult.Mkdir = mkdirResult
		if err != nil {
			result.Files = append(result.Files, stepResult)
			return result, fmt.Errorf("file step %d ensure parent directory for %s: %w", index, file.Destination, err)
		}

		if err := transport.Upload(ctx, UploadRequest{
			User:        request.User,
			Host:        request.Host,
			Port:        request.Port,
			Source:      file.Source,
			Destination: file.Destination,
			Timeout:     timeout,
		}); err != nil {
			result.Files = append(result.Files, stepResult)
			return result, fmt.Errorf("file step %d upload %s -> %s: %w", index, file.Source, file.Destination, err)
		}
		stepResult.Upload = true

		if file.Permissions != "" {
			chmodCommand := buildChmodCommand(file.Permissions, file.Destination)
			chmodResult, err := transport.Run(ctx, RunRequest{
				User:    request.User,
				Host:    request.Host,
				Port:    request.Port,
				Command: chmodCommand,
				Timeout: timeout,
			})
			stepResult.Chmod = chmodResult
			if err != nil {
				result.Files = append(result.Files, stepResult)
				return result, fmt.Errorf("file step %d chmod %s on %s: %w", index, file.Permissions, file.Destination, err)
			}
		}

		result.Files = append(result.Files, stepResult)
	}

	return result, nil
}

func normalizeFileSteps(steps []provision.FileStep) ([]provision.FileStep, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	files := make([]provision.FileStep, 0, len(steps))
	for i, step := range steps {
		source := strings.TrimSpace(step.Source)
		destination := strings.TrimSpace(step.Destination)
		permissions := strings.TrimSpace(step.Permissions)

		if source == "" {
			return nil, fmt.Errorf("file step %d source is empty", i)
		}
		if destination == "" {
			return nil, fmt.Errorf("file step %d destination is empty", i)
		}

		files = append(files, provision.FileStep{
			Source:      source,
			Destination: destination,
			Permissions: permissions,
			Origin:      step.Origin,
		})
	}

	return files, nil
}

func buildMkdirCommand(destination string) string {
	parent := path.Dir(destination)
	if parent == "." {
		parent = destination
	}
	return "mkdir -p " + shellQuote(parent)
}

func buildChmodCommand(permissions, destination string) string {
	return "chmod " + permissions + " " + shellQuote(destination)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
