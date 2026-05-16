package project

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	YeastHomeDirName  = ".yeast"
	CacheDirName      = "cache"
	ImagesDirName     = "images"
	ProjectsDirName   = "projects"
	InstancesDirName  = "instances"
	StateFileName      = "state.json"
	StateLockFileName  = "state.lock"
	SnapshotsDirName   = "snapshots"
)

var instanceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Paths struct {
	YeastHome  string
	ProjectID  string
	ProjectDir string
	StateFile  string
	StateLock  string
	Instances  string
	ImageCache string
}

func DefaultYeastHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, YeastHomeDirName), nil
}

func NewPaths(yeastHome string, metadata Metadata) (Paths, error) {
	if strings.TrimSpace(yeastHome) == "" {
		return Paths{}, fmt.Errorf("yeast home path is required")
	}
	if err := ValidateMetadata(metadata); err != nil {
		return Paths{}, fmt.Errorf("invalid project metadata: %w", err)
	}

	home := filepath.Clean(yeastHome)
	projectDir := filepath.Join(home, ProjectsDirName, metadata.ID)
	return Paths{
		YeastHome:  home,
		ProjectID:  metadata.ID,
		ProjectDir: projectDir,
		StateFile:  filepath.Join(projectDir, StateFileName),
		StateLock:  filepath.Join(projectDir, StateLockFileName),
		Instances:  filepath.Join(projectDir, InstancesDirName),
		ImageCache: filepath.Join(home, CacheDirName, ImagesDirName),
	}, nil
}

func ResolvePaths(metadata Metadata) (Paths, error) {
	home, err := DefaultYeastHome()
	if err != nil {
		return Paths{}, err
	}
	return NewPaths(home, metadata)
}

func (p Paths) InstanceDir(name string) (string, error) {
	if !IsValidInstanceName(name) {
		return "", fmt.Errorf("invalid instance name %q", name)
	}
	return filepath.Join(p.Instances, name), nil
}

func (p Paths) SnapshotDir(instanceName string) (string, error) {
	instanceDir, err := p.InstanceDir(instanceName)
	if err != nil {
		return "", err
	}
	return filepath.Join(instanceDir, SnapshotsDirName), nil
}

func IsValidInstanceName(name string) bool {
	if !instanceNamePattern.MatchString(name) {
		return false
	}
	return !strings.Contains(name, "..")
}

