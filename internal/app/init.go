package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"yeast/internal/project"
)

const ConfigFileName = "yeast.yaml"

var ErrProjectAlreadyInitialized = errors.New("project already initialized")

type InitOptions struct {
	ProjectRoot string
	Now         time.Time
}

type InitResult struct {
	ProjectRoot  string
	ConfigPath   string
	MetadataPath string
	ProjectID    string
	Created      bool
}

func (s *Service) Init(options InitOptions) (InitResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return InitResult{}, fmt.Errorf("resolve project root: %w", err)
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
		return result, fmt.Errorf("%w: %s already exists", ErrProjectAlreadyInitialized, configPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("inspect config file %s: %w", configPath, err)
	}

	if _, err := os.Stat(metadataPath); err == nil {
		return result, fmt.Errorf("%w: %s already exists", ErrProjectAlreadyInitialized, metadataPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("inspect project metadata %s: %w", metadataPath, err)
	}

	metadata, err := project.EnsureMetadata(absoluteRoot, now)
	if err != nil {
		return result, err
	}
	if err := writeFileAtomic(configPath, []byte(defaultConfig())); err != nil {
		return result, err
	}

	result.ProjectID = metadata.ID
	result.Created = true
	return result, nil
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

