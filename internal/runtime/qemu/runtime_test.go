package qemu

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	rtm "yeast/internal/runtime"
)

func TestRuntimeStartCreatesLogAndReleasesHandle(t *testing.T) {
	previous := startProcess
	defer func() {
		startProcess = previous
	}()

	var gotName string
	var gotArgs []string
	var gotLogPath string
	var released bool
	startProcess = func(_ context.Context, name string, args []string, stdout, stderr *os.File) (processHandle, error) {
		gotName = name
		gotArgs = append([]string(nil), args...)
		if stdout == nil || stderr == nil {
			t.Fatal("expected stdout and stderr log files")
		}
		if stdout.Name() != stderr.Name() {
			t.Fatalf("expected shared log file, got stdout=%q stderr=%q", stdout.Name(), stderr.Name())
		}
		gotLogPath = stdout.Name()
		if _, err := stdout.WriteString("started\n"); err != nil {
			t.Fatalf("write log output: %v", err)
		}
		return &fakeProcessHandle{
			pid: 4242,
			releaseFn: func() error {
				released = true
				return nil
			},
		}, nil
	}

	root := t.TempDir()
	plan := rtm.MachinePlan{
		Name:          "web",
		RuntimeDir:    filepath.Join(root, "instances", "web"),
		LogPath:       filepath.Join(root, "instances", "web", "vm.log"),
		MemoryMiB:     1024,
		CPUs:          1,
		SeedImagePath: filepath.Join(root, "instances", "web", "seed.iso"),
		Disk: rtm.DiskPlan{
			DiskPath: filepath.Join(root, "instances", "web", "disk.qcow2"),
		},
		ManagementNetwork: rtm.NetworkOptions{
			ManagementSSHPort: 2222,
		},
	}

	rt := NewRuntime()
	got, err := rt.Start(context.Background(), plan)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if gotName != qemuSystemBinary {
		t.Fatalf("unexpected binary: got %q want %q", gotName, qemuSystemBinary)
	}
	if len(gotArgs) == 0 {
		t.Fatal("expected command args")
	}
	if gotLogPath != plan.LogPath {
		t.Fatalf("unexpected log path: got %q want %q", gotLogPath, plan.LogPath)
	}
	if !released {
		t.Fatal("expected process handle to be released")
	}
	if got.PID != 4242 {
		t.Fatalf("unexpected pid: got %d want %d", got.PID, 4242)
	}
	content, err := os.ReadFile(plan.LogPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), "started") {
		t.Fatalf("expected log output, got %q", string(content))
	}
}

func TestRuntimeStopTerminatesProcess(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	rt := NewRuntime()
	instance := rtm.RuntimeInstance{
		PID: cmd.Process.Pid,
	}

	if err := rt.Stop(context.Background(), instance, 2*time.Second); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	running, err := processRunning(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("inspect process: %v", err)
	}
	if running {
		t.Fatal("expected process to be stopped")
	}
}

func TestRuntimeStopKillsAfterTimeout(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("sh", "-c", `trap "" TERM; while :; do sleep 1; done`)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start stubborn process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	rt := NewRuntime()
	instance := rtm.RuntimeInstance{
		PID: cmd.Process.Pid,
	}

	if err := rt.Stop(context.Background(), instance, 200*time.Millisecond); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	running, err := processRunning(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("inspect process: %v", err)
	}
	if running {
		t.Fatal("expected process to be killed after timeout")
	}
}

func TestRuntimeInspectReportsRunningAndStopped(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	rt := NewRuntime()
	instance := rtm.RuntimeInstance{
		PID:       cmd.Process.Pid,
		StartedAt: time.Now().UTC(),
	}

	info, err := rt.Inspect(context.Background(), instance)
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if info.State != rtm.ProcessStateRunning {
		t.Fatalf("unexpected running state: got %q", info.State)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill process: %v", err)
	}
	_, _ = cmd.Process.Wait()

	info, err = rt.Inspect(context.Background(), instance)
	if err != nil {
		t.Fatalf("Inspect returned error after stop: %v", err)
	}
	if info.State != rtm.ProcessStateStopped {
		t.Fatalf("unexpected stopped state: got %q", info.State)
	}
}

type fakeProcessHandle struct {
	pid       int
	releaseFn func() error
}

func (p *fakeProcessHandle) PID() int {
	return p.pid
}

func (p *fakeProcessHandle) Release() error {
	if p.releaseFn == nil {
		return nil
	}
	return p.releaseFn()
}
