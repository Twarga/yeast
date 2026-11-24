package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Twarga/yeast/internal/project"
	"github.com/Twarga/yeast/internal/state"
)

// MigrateOptions controls `yeast migrate`.
type MigrateOptions struct {
	Root   string `json:"-"`
	DryRun bool   `json:"-"`
}

type MigrateChange struct {
	Instance string `json:"instance,omitempty"`
	Action   string `json:"action"`
	Detail   string `json:"detail,omitempty"`
}

type MigrateResult struct {
	ProjectID string          `json:"project_id"`
	DryRun    bool            `json:"dry_run"`
	Instances int             `json:"instances"`
	Changes   []MigrateChange `json:"changes"`
}

// Migrate upgrades a v1-era yeast project directory to v2 layout.
// State schema is additive (yeast.state.v1 across releases), so migration
// means: reconcile dead processes, backfill new instance fields, create
// directories v2 expects (snapshots/, runtime dirs), and remove stale
// partial-download artifacts.
func (s *Service) Migrate(ctx context.Context, options MigrateOptions) (*MigrateResult, error) {
	root, err := filepath.Abs(options.Root)
	if err != nil {
		return nil, err
	}
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		return nil, WrapError(ErrorCodeNotFound, "yeast project "+root, err)
	}
	paths, err := project.ResolvePaths(metadata)
	if err != nil {
		return nil, err
	}

	result := &MigrateResult{ProjectID: metadata.ID, DryRun: options.DryRun, Changes: []MigrateChange{}}
	if _, err := os.Stat(paths.ProjectDir); err != nil {
		if os.IsNotExist(err) {
			result.Changes = append(result.Changes, MigrateChange{
				Action: "create-project-dir", Detail: paths.ProjectDir,
			})
			if !options.DryRun {
				_ = os.MkdirAll(paths.Instances, 0755)
			}
		} else {
			return nil, err
		}
	}

	st, loadErr := state.Load(paths.StateFile, metadata.ID)
	if loadErr != nil {
		return nil, loadErr
	}
	result.Instances = len(st.Instances)

	// 1. Reconcile live processes (dead PID → stopped, orphaned → reattach).
	if s.reconcileStateWithRuntime(ctx, &st) {
		result.Changes = append(result.Changes, MigrateChange{
			Action: "reconcile", Detail: "instance statuses refreshed against live processes",
		})
	}

	for name, inst := range st.Instances {
		// 2. Ensure per-instance runtime + snapshots dirs exist.
		if inst.RuntimeDir != "" {
			if _, err := os.Stat(inst.RuntimeDir); os.IsNotExist(err) {
				result.Changes = append(result.Changes, MigrateChange{
					Instance: name, Action: "create-runtime-dir", Detail: inst.RuntimeDir,
				})
				if !options.DryRun {
					_ = os.MkdirAll(inst.RuntimeDir, 0755)
				}
			}
		}
		if snapDir, err := paths.SnapshotDir(name); err == nil {
			if _, err := os.Stat(snapDir); os.IsNotExist(err) {
				result.Changes = append(result.Changes, MigrateChange{
					Instance: name, Action: "create-snapshots-dir", Detail: snapDir,
				})
				if !options.DryRun {
					_ = os.MkdirAll(snapDir, 0755)
				}
			}
		}

		// 3. Backfill snapshot kinds: pre-v1.4 snapshots are full copies.
		for snapName, snap := range inst.Snapshots {
			if snap.Kind == "" {
				snap.Kind = "copy"
				inst.Snapshots[snapName] = snap
				result.Changes = append(result.Changes, MigrateChange{
					Instance: name, Action: "snapshot-kind",
					Detail: fmt.Sprintf("%s → copy (legacy full-copy snapshot)", snapName),
				})
			}
		}

		// 4. Clean stale partial artifacts from interrupted downloads/writes.
		if inst.RuntimeDir != "" {
			if entries, err := os.ReadDir(inst.RuntimeDir); err == nil {
				for _, e := range entries {
					if strings.HasSuffix(e.Name(), ".part") || strings.HasSuffix(e.Name(), ".tmp") {
						stale := filepath.Join(inst.RuntimeDir, e.Name())
						if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > time.Hour {
							result.Changes = append(result.Changes, MigrateChange{
								Instance: name, Action: "remove-stale", Detail: stale,
							})
							if !options.DryRun {
								_ = os.Remove(stale)
							}
						}
					}
				}
			}
		}
		st.Instances[name] = inst
	}

	if !options.DryRun && len(result.Changes) > 0 {
		if err := state.Save(paths.StateFile, st); err != nil {
			return nil, fmt.Errorf("write migrated state: %w", err)
		}
	}
	return result, nil
}
