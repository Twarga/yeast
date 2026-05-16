package qemu

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"yeast/internal/runtime"
)

func TestBuildCreateOverlayArgsWithoutSize(t *testing.T) {
	t.Parallel()

	disk := runtime.DiskPlan{
		BaseImagePath: "/cache/image.qcow2",
		DiskPath:      "/runtime/web/disk.qcow2",
	}

	got := buildCreateOverlayArgs(disk)
	want := []string{
		"create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", "/cache/image.qcow2",
		"/runtime/web/disk.qcow2",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildCreateOverlayArgsWithSize(t *testing.T) {
	t.Parallel()

	disk := runtime.DiskPlan{
		BaseImagePath: "/cache/image.qcow2",
		DiskPath:      "/runtime/web/disk.qcow2",
		Size:          "20G",
	}

	got := buildCreateOverlayArgs(disk)
	want := []string{
		"create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", "/cache/image.qcow2",
		"/runtime/web/disk.qcow2",
		"20G",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestPrepareDiskCreatesDirectoriesAndInvokesQEMUImg(t *testing.T) {
	previous := runCommand
	defer func() {
		runCommand = previous
	}()

	var called bool
	var gotName string
	var gotArgs []string
	runCommand = func(_ context.Context, name string, args ...string) error {
		called = true
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	root := t.TempDir()
	plan := runtime.MachinePlan{
		RuntimeDir: filepath.Join(root, "instances", "web"),
		Disk: runtime.DiskPlan{
			BaseImagePath: filepath.Join(root, "cache", "ubuntu.qcow2"),
			DiskPath:      filepath.Join(root, "instances", "web", "disk.qcow2"),
			Size:          "10G",
		},
	}

	got, err := PrepareDisk(context.Background(), plan)
	if err != nil {
		t.Fatalf("PrepareDisk returned error: %v", err)
	}
	if !called {
		t.Fatal("expected qemu-img runner to be called")
	}
	if gotName != qemuImgBinary {
		t.Fatalf("unexpected binary: got %q want %q", gotName, qemuImgBinary)
	}

	wantArgs := []string{
		"create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", plan.Disk.BaseImagePath,
		plan.Disk.DiskPath,
		"10G",
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", gotArgs, wantArgs)
	}
	if _, err := os.Stat(plan.RuntimeDir); err != nil {
		t.Fatalf("runtime directory was not created: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(plan.Disk.DiskPath)); err != nil {
		t.Fatalf("disk directory was not created: %v", err)
	}
	if got != plan.Disk {
		t.Fatalf("unexpected disk plan returned: got %#v want %#v", got, plan.Disk)
	}
}

func TestPrepareDiskSkipsExistingDisk(t *testing.T) {
	previous := runCommand
	defer func() {
		runCommand = previous
	}()

	runCommand = func(_ context.Context, _ string, _ ...string) error {
		t.Fatal("runner should not be called for existing disk")
		return nil
	}

	root := t.TempDir()
	diskPath := filepath.Join(root, "instances", "web", "disk.qcow2")
	if err := os.MkdirAll(filepath.Dir(diskPath), 0755); err != nil {
		t.Fatalf("create disk directory: %v", err)
	}
	if err := os.WriteFile(diskPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("create existing disk: %v", err)
	}

	plan := runtime.MachinePlan{
		RuntimeDir: filepath.Join(root, "instances", "web"),
		Disk: runtime.DiskPlan{
			BaseImagePath: filepath.Join(root, "cache", "ubuntu.qcow2"),
			DiskPath:      diskPath,
		},
	}

	got, err := PrepareDisk(context.Background(), plan)
	if err != nil {
		t.Fatalf("PrepareDisk returned error: %v", err)
	}
	if got != plan.Disk {
		t.Fatalf("unexpected disk plan returned: got %#v want %#v", got, plan.Disk)
	}
}
