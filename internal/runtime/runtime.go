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
	DeleteSnapshot(ctx context.Context, plan DeleteSnapshotPlan) error
	Destroy(ctx context.Context, instance RuntimeInstance) error
}

// ProcessFinder is an optional interface that Runtimes can implement
// to support finding processes by name and runtime directory.
type ProcessFinder interface {
	FindProcesses(ctx context.Context, targets []CleanupTarget) ([]CleanupResult, error)
}

// LiveSnapshotter is implemented by runtimes that can checkpoint a running
// guest (RAM + disk) and respawn from the checkpoint.
type LiveSnapshotter interface {
	SaveLiveSnapshot(ctx context.Context, plan LiveSnapshotPlan) (LiveSnapshotInfo, error)
	RestoreLiveSnapshot(ctx context.Context, plan LiveRestorePlan) (RuntimeInstance, error)
}

// DisplayController is an optional interface for runtimes that can drive a
// machine's graphical display through the monitor protocol (QMP for QEMU).
// It works without any guest agent — even during UEFI boot or a crashed DE.
type DisplayController interface {
	// Screenshot dumps the current framebuffer to outputPath as PNG.
	Screenshot(ctx context.Context, instance RuntimeInstance, outputPath string) error
	// SendKeys presses a simultaneous key chord (QKeyCode names, e.g.
	// "meta_l"+"ret") held for holdTimeMs milliseconds.
	SendKeys(ctx context.Context, instance RuntimeInstance, keys []string, holdTimeMs int) error
	// SendText types printable ASCII text (with \n → Return, \t → Tab).
	SendText(ctx context.Context, instance RuntimeInstance, text string) error
}

// DiskExporter is an optional interface for runtimes that can flatten a VM
// disk into a standalone image and create throwaway clones for sysprep.
type DiskExporter interface {
	// ExportDisk flattens srcDiskPath (resolving any backing chain) into a
	// standalone compressed qcow2 at dstPath. src must not be in use.
	ExportDisk(ctx context.Context, srcDiskPath, dstPath string) error
	// CloneDisk creates a qcow2 overlay backed by srcDiskPath inside dstDir
	// (a throwaway copy-on-write clone for sysprep/export work). The base
	// file is never modified.
	CloneDisk(ctx context.Context, srcDiskPath, dstDir string) (string, error)
	// CreateBlankDisk creates an empty qcow2 of the given size ("20G") —
	// used by `yeast boot-iso --disk` for installer targets.
	CreateBlankDisk(ctx context.Context, path, size string) error
}
