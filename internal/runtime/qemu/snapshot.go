package qemu

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"yeast/internal/runtime"
)

func CreateSnapshotCopy(ctx context.Context, plan runtime.SnapshotPlan) error {
	if plan.InstanceDiskPath == "" {
		return fmt.Errorf("instance disk path is required")
	}
	if plan.SnapshotPath == "" {
		return fmt.Errorf("snapshot path is required")
	}
	if _, err := os.Stat(plan.InstanceDiskPath); err != nil {
		return fmt.Errorf("inspect instance disk %s: %w", plan.InstanceDiskPath, err)
	}
	if _, err := os.Stat(plan.SnapshotPath); err == nil {
		return fmt.Errorf("snapshot file %s already exists", plan.SnapshotPath)
	} else if !errorsIsNotExist(err) {
		return fmt.Errorf("inspect snapshot path %s: %w", plan.SnapshotPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(plan.SnapshotPath), 0755); err != nil {
		return fmt.Errorf("create snapshot directory %s: %w", filepath.Dir(plan.SnapshotPath), err)
	}
	if err := copyFileAtomic(ctx, plan.InstanceDiskPath, plan.SnapshotPath, false); err != nil {
		return fmt.Errorf("create snapshot copy %s: %w", plan.SnapshotPath, err)
	}
	return nil
}

func RestoreSnapshotCopy(ctx context.Context, plan runtime.SnapshotPlan) error {
	if plan.InstanceDiskPath == "" {
		return fmt.Errorf("instance disk path is required")
	}
	if plan.SnapshotPath == "" {
		return fmt.Errorf("snapshot path is required")
	}
	if _, err := os.Stat(plan.SnapshotPath); err != nil {
		return fmt.Errorf("inspect snapshot file %s: %w", plan.SnapshotPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(plan.InstanceDiskPath), 0755); err != nil {
		return fmt.Errorf("create instance disk directory %s: %w", filepath.Dir(plan.InstanceDiskPath), err)
	}
	if err := copyFileAtomic(ctx, plan.SnapshotPath, plan.InstanceDiskPath, true); err != nil {
		return fmt.Errorf("restore instance disk %s from snapshot %s: %w", plan.InstanceDiskPath, plan.SnapshotPath, err)
	}
	return nil
}

func DeleteSnapshotFile(path string) error {
	if path == "" {
		return fmt.Errorf("snapshot path is required")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove snapshot file %s: %w", path, err)
	}
	return nil
}

func copyFileAtomic(ctx context.Context, sourcePath string, destinationPath string, allowOverwrite bool) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", sourcePath, err)
	}
	defer func() { _ = source.Close() }()

	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if !allowOverwrite {
		flags |= os.O_EXCL
	}

	tmpPath := destinationPath + ".tmp"
	tmp, err := os.OpenFile(tmpPath, flags, 0644)
	if err != nil {
		return fmt.Errorf("open temp file %s: %w", tmpPath, err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tmp, &contextReader{ctx: ctx, file: source}); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, destinationPath); err != nil {
		return fmt.Errorf("rename temp file %s to %s: %w", tmpPath, destinationPath, err)
	}
	return nil
}

type contextReader struct {
	ctx  context.Context
	file *os.File
}

func (r *contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.file.Read(p)
	}
}
