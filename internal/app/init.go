package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"yeast/internal/project"
	"yeast/internal/templates"
)

const ConfigFileName = "yeast.yaml"

var ErrProjectAlreadyInitialized = errors.New("project already initialized")
var writeConfigFileAtomic = writeFileAtomic

type InitOptions struct {
	ProjectRoot string
	Now         time.Time
	Template    string
}

type InitResult struct {
	ProjectRoot  string `json:"project_root"`
	ConfigPath   string `json:"config_path"`
	MetadataPath string `json:"metadata_path"`
	ProjectID    string `json:"project_id"`
	Template     string `json:"template,omitempty"`
	Created      bool   `json:"created"`
}

func (s *Service) Init(options InitOptions) (InitResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return InitResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}

	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	configPath := filepath.Join(absoluteRoot, ConfigFileName)
	metadataPath := project.MetadataPath(absoluteRoot)
	result := InitResult{
		ProjectRoot:  absoluteRoot,
		ConfigPath:   configPath,
		MetadataPath: metadataPath,
	}

	if _, err := os.Stat(configPath); err == nil {
		cause := fmt.Errorf("%w: %s already exists", ErrProjectAlreadyInitialized, configPath)
		return result, WrapError(ErrorCodeConflict, cause.Error(), cause)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, WrapError(ErrorCodeInternal, fmt.Sprintf("inspect config file %s: %v", configPath, err), err)
	}

	if _, err := os.Stat(metadataPath); err == nil {
		cause := fmt.Errorf("%w: %s already exists", ErrProjectAlreadyInitialized, metadataPath)
		return result, WrapError(ErrorCodeConflict, cause.Error(), cause)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, WrapError(ErrorCodeInternal, fmt.Sprintf("inspect project metadata %s: %v", metadataPath, err), err)
	}

	if options.Template != "" {
		template, err := resolveInitTemplate(options.Template)
		if err != nil {
			return result, err
		}
		if _, err := templates.Materialize(template, templates.MaterializeOptions{Destination: absoluteRoot}); err != nil {
			if errors.Is(err, templates.ErrOutputExists) {
				return result, WrapError(ErrorCodeConflict, err.Error(), err)
			}
			return result, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		result.Template = template.Metadata.Name
	} else {
		if err := writeConfigFileAtomic(configPath, []byte(defaultConfig())); err != nil {
			return result, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}

	metadata, err := project.EnsureMetadata(absoluteRoot, now)
	if err != nil {
		return result, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	result.ProjectID = metadata.ID
	result.Created = true
	return result, nil
}

func resolveInitTemplate(value string) (templates.Template, error) {
	if looksLikePath(value) {
		template, err := templates.LoadLocal(value)
		if err != nil {
			return templates.Template{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
		}
		return template, nil
	}

	template, ok, err := templates.LookupBuiltin(value)
	if err != nil {
		return templates.Template{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if !ok {
		cause := fmt.Errorf("template %q was not found", value)
		return templates.Template{}, WrapError(ErrorCodeNotFound, cause.Error(), cause)
	}
	return template, nil
}

func looksLikePath(value string) bool {
	return filepath.IsAbs(value) || value == "." || value == ".." || filepath.Dir(value) != "."
}

func defaultConfig() string {
	return `version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
`
}

func writeFileAtomic(path string, content []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save file %s: %w", path, err)
	}
	return nil
}
