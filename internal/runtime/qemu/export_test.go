package qemu

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportDiskInvokesQemuImgConvert(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "disk.qcow2")
	dst := filepath.Join(dir, "out.qcow2")

	previous := runCommand
	defer func() { runCommand = previous }()
	var gotArgs []string
	runCommand = func(_ context.Context, name string, args ...string) error {
		gotArgs = append([]string{name}, args...)
		// simulate qemu-img producing the .part file
		return os.WriteFile(dst+".part", []byte("img"), 0o644)
	}

	rt := &Runtime{}
	if err := rt.ExportDisk(context.Background(), src, dst); err != nil {
		t.Fatalf("ExportDisk: %v", err)
	}
	joined := strings.Join(gotArgs, " ")
	for _, want := range []string{"qemu-img", "convert", "-f", "qcow2", "-O", "qcow2", "-c", src} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("output not renamed into place: %v", err)
	}
	if _, err := os.Stat(dst + ".part"); !os.IsNotExist(err) {
		t.Fatal(".part file should be renamed away")
	}
}

func TestExportDiskFailureLeavesNoPartFile(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.qcow2")
	previous := runCommand
	defer func() { runCommand = previous }()
	runCommand = func(_ context.Context, _ string, _ ...string) error {
		return os.ErrPermission
	}
	rt := &Runtime{}
	if err := rt.ExportDisk(context.Background(), filepath.Join(dir, "d.qcow2"), dst); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(dst + ".part"); !os.IsNotExist(err) {
		t.Fatal("partial export must be removed on failure")
	}
}

func TestCloneDiskAndBlankDiskArgs(t *testing.T) {
	dir := t.TempDir()
	previous := runCommand
	defer func() { runCommand = previous }()
	var calls [][]string
	runCommand = func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}
	rt := &Runtime{}
	clone, err := rt.CloneDisk(context.Background(), "/base/disk.qcow2", dir)
	if err != nil {
		t.Fatalf("CloneDisk: %v", err)
	}
	if !strings.HasSuffix(clone, ".qcow2") || !strings.HasPrefix(clone, dir) {
		t.Fatalf("clone path unexpected: %s", clone)
	}
	j := strings.Join(calls[0], " ")
	if !strings.Contains(j, "create") || !strings.Contains(j, "-b /base/disk.qcow2") || !strings.Contains(j, "-F qcow2") {
		t.Fatalf("clone args: %s", j)
	}
	if err := rt.CreateBlankDisk(context.Background(), filepath.Join(dir, "scratch.qcow2"), "20G"); err != nil {
		t.Fatalf("CreateBlankDisk: %v", err)
	}
	j = strings.Join(calls[1], " ")
	if !strings.Contains(j, "create") || !strings.Contains(j, "20G") {
		t.Fatalf("blank args: %s", j)
	}
}
