package runtime

import (
	"context"
	"time"
)

type Runtime interface {
	PrepareDisk(ctx context.Context, plan MachinePlan) (DiskPlan, error)
	Start(ctx context.Context, plan MachinePlan) (RuntimeInstance, error)
	Stop(ctx context.Context, instance RuntimeInstance, timeout time.Duration) error
	Inspect(ctx context.Context, instance RuntimeInstance) (ProcessInfo, error)
	CreateSnapshot(ctx context.Context, plan SnapshotPlan) error
	RestoreSnapshot(ctx context.Context, plan SnapshotPlan) error
	DeleteSnapshot(ctx context.Context, snapshotPath string) error
	Destroy(ctx context.Context, instance RuntimeInstance) error
}

// ProcessFinder is an optional interface that Runtimes can implement
// to support finding processes by name and runtime directory.
type ProcessFinder interface {
	FindProcesses(ctx context.Context, targets []CleanupTarget) ([]CleanupResult, error)
}
