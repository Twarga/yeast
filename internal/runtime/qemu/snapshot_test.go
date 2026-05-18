package qemu

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"yeast/internal/runtime"
)

func TestCreateSnapshotCopyCopiesDisk(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	diskPath := filepath.Join(root, "instances", "web", "disk.qcow2")
	snapshotPath := filepath.Join(root, "instances", "web", "snapshots", "clean.qcow2")
	if err := os.MkdirAll(filepath.Dir(diskPath), 0755); err != nil {
		t.Fatalf("create disk dir: %v", err)
	}
	if err := os.WriteFile(diskPath, []byte("disk-data"), 0644); err != nil {
		t.Fatalf("write disk: %v", err)
	}

	if err := CreateSnapshotCopy(context.Background(), runtime.SnapshotPlan{
		InstanceDiskPath: diskPath,
		SnapshotPath:     snapshotPath,
	}); err != nil {
		t.Fatalf("CreateSnapshotCopy returned error: %v", err)
	}

	content, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(content) != "disk-data" {
		t.Fatalf("unexpected snapshot content %q", string(content))
	}
}

func TestCreateSnapshotCopyFailsWhenSourceMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	err := CreateSnapshotCopy(context.Background(), runtime.SnapshotPlan{
		InstanceDiskPath: filepath.Join(root, "missing.qcow2"),
		SnapshotPath:     filepath.Join(root, "snapshots", "clean.qcow2"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateSnapshotCopyDoesNotOverwriteExistingSnapshot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	diskPath := filepath.Join(root, "disk.qcow2")
	snapshotPath := filepath.Join(root, "snapshots", "clean.qcow2")
	if err := os.WriteFile(diskPath, []byte("new-data"), 0644); err != nil {
		t.Fatalf("write disk: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		t.Fatalf("create snapshot dir: %v", err)
	}
	if err := os.WriteFile(snapshotPath, []byte("existing-data"), 0644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	err := CreateSnapshotCopy(context.Background(), runtime.SnapshotPlan{
		InstanceDiskPath: diskPath,
		SnapshotPath:     snapshotPath,
	})
	if err == nil {
		t.Fatal("expected overwrite protection error, got nil")
	}

	content, readErr := os.ReadFile(snapshotPath)
	if readErr != nil {
		t.Fatalf("read snapshot after failed create: %v", readErr)
	}
	if string(content) != "existing-data" {
		t.Fatalf("snapshot content changed unexpectedly to %q", string(content))
	}
}

func TestRestoreSnapshotCopyOverwritesInstanceDisk(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	diskPath := filepath.Join(root, "instances", "web", "disk.qcow2")
	snapshotPath := filepath.Join(root, "instances", "web", "snapshots", "clean.qcow2")
	if err := os.MkdirAll(filepath.Dir(diskPath), 0755); err != nil {
		t.Fatalf("create disk dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		t.Fatalf("create snapshot dir: %v", err)
	}
	if err := os.WriteFile(diskPath, []byte("broken-data"), 0644); err != nil {
		t.Fatalf("write disk: %v", err)
	}
	if err := os.WriteFile(snapshotPath, []byte("clean-data"), 0644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	if err := RestoreSnapshotCopy(context.Background(), runtime.SnapshotPlan{
		InstanceDiskPath: diskPath,
		SnapshotPath:     snapshotPath,
	}); err != nil {
		t.Fatalf("RestoreSnapshotCopy returned error: %v", err)
	}

	content, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("read restored disk: %v", err)
	}
	if string(content) != "clean-data" {
		t.Fatalf("unexpected restored disk content %q", string(content))
	}
}

func TestRestoreSnapshotCopyFailsWhenSnapshotMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	err := RestoreSnapshotCopy(context.Background(), runtime.SnapshotPlan{
		InstanceDiskPath: filepath.Join(root, "disk.qcow2"),
		SnapshotPath:     filepath.Join(root, "missing.qcow2"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteSnapshotFileRemovesSnapshot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	snapshotPath := filepath.Join(root, "snapshots", "clean.qcow2")
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		t.Fatalf("create snapshot dir: %v", err)
	}
	if err := os.WriteFile(snapshotPath, []byte("snapshot-data"), 0644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	if err := DeleteSnapshotFile(snapshotPath); err != nil {
		t.Fatalf("DeleteSnapshotFile returned error: %v", err)
	}
	if _, err := os.Stat(snapshotPath); !os.IsNotExist(err) {
		t.Fatalf("expected snapshot to be removed, stat err=%v", err)
	}
}
