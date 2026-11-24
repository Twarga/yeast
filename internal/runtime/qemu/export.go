package qemu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	rtm "github.com/Twarga/yeast/internal/runtime"
)

var _ rtm.DiskExporter = (*Runtime)(nil)

// ExportDisk flattens src (qcow2, any backing depth) into a standalone,
// compressed qcow2 at dst. Written via a .part temp then renamed so a
// failed convert never leaves a half image at the destination.
func (r *Runtime) ExportDisk(ctx context.Context, srcDiskPath, dstPath string) error {
	if srcDiskPath == "" || dstPath == "" {
		return fmt.Errorf("export requires source and destination paths")
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create export dir: %w", err)
	}
	tmp := dstPath + ".part"
	_ = os.Remove(tmp)
	// -c compresses; convert resolves the whole backing chain into one file.
	args := []string{"convert", "-f", "qcow2", "-O", "qcow2", "-c", srcDiskPath, tmp}
	if err := runCommand(ctx, qemuImgBinary, args...); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("qemu-img convert: %w", err)
	}
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("move export into place: %w", err)
	}
	return nil
}

// CloneDisk creates a qcow2 overlay backed (read-only) by src inside dstDir.
func (r *Runtime) CloneDisk(ctx context.Context, srcDiskPath, dstDir string) (string, error) {
	if srcDiskPath == "" || dstDir == "" {
		return "", fmt.Errorf("clone requires source and destination dir")
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return "", fmt.Errorf("create clone dir: %w", err)
	}
	dst := filepath.Join(dstDir, fmt.Sprintf("clone-%d.qcow2", time.Now().UnixNano()))
	args := []string{"create", "-f", "qcow2", "-b", srcDiskPath, "-F", "qcow2", dst}
	if err := runCommand(ctx, qemuImgBinary, args...); err != nil {
		return "", fmt.Errorf("qemu-img clone: %w", err)
	}
	return dst, nil
}

// CreateBlankDisk creates an empty qcow2 — a scratch installer target.
func (r *Runtime) CreateBlankDisk(ctx context.Context, path, size string) error {
	if path == "" || size == "" {
		return fmt.Errorf("blank disk requires path and size")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return runCommand(ctx, qemuImgBinary, "create", "-f", "qcow2", path, size)
}
