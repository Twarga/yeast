package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Twarga/yeast/internal/project"
)

// Registry maps project IDs to their source roots so the daemon can resolve
// /v1/projects/{id} requests. Persisted as JSON under the Yeast home dir so
// registrations survive daemon restarts.
type Registry struct {
	mu       sync.Mutex
	path     string
	projects map[string]string // project ID -> absolute project root
}

func NewRegistry(yeastHome string) (*Registry, error) {
	r := &Registry{
		path:     filepath.Join(yeastHome, "daemon-projects.json"),
		projects: map[string]string{},
	}
	if data, err := os.ReadFile(r.path); err == nil {
		_ = json.Unmarshal(data, &r.projects)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load daemon registry: %w", err)
	}
	return r, nil
}

// Register resolves the project metadata at root and records id -> root.
func (r *Registry) Register(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	metadata, err := project.LoadMetadata(abs)
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projects[metadata.ID] = abs
	return metadata.ID, r.saveLocked()
}

// Resolve returns the project root for an ID.
func (r *Registry) Resolve(id string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	root, ok := r.projects[id]
	return root, ok
}

// List returns a copy of the registry.
func (r *Registry) List() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.projects))
	for k, v := range r.projects {
		out[k] = v
	}
	return out
}

// Forget removes a registration (e.g. root no longer exists).
func (r *Registry) Forget(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.projects, id)
	return r.saveLocked()
}

func (r *Registry) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r.projects, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}
