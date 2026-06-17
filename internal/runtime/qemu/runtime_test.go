package qemu

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	rtm "yeast/internal/runtime"

	"golang.org/x/sys/unix"
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
	var handle *fakeProcessHandle
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
		handle = &fakeProcessHandle{
			pid: 4242,
			releaseFn: func() error {
				released = true
				handle.pid = -1
				return nil
			},
		}
		return handle, nil
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
		Networks: rtm.NetworkPlan{
			Management: rtm.ManagementNetworkPlan{
				SSHHost:       "127.0.0.1",
				SSHPort:       2222,
				InterfaceName: "yeastmgmt0",
				MACAddress:    "52:54:00:11:22:33",
			},
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

func TestRuntimeStartRemovesStaleQMPSocket(t *testing.T) {
	previous := startProcess
	defer func() {
		startProcess = previous
	}()

	startProcess = func(_ context.Context, name string, args []string, stdout, stderr *os.File) (processHandle, error) {
		return &fakeProcessHandle{pid: 4242}, nil
	}

	root := t.TempDir()
	runtimeDir := filepath.Join(root, "instances", "web")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	socketPath := qmpSocketPath(runtimeDir)
	if err := os.WriteFile(socketPath, []byte("stale"), 0644); err != nil {
		t.Fatalf("write stale socket marker: %v", err)
	}

	plan := rtm.MachinePlan{
		Name:          "web",
		RuntimeDir:    runtimeDir,
		LogPath:       filepath.Join(runtimeDir, "vm.log"),
		MemoryMiB:     1024,
		CPUs:          1,
		SeedImagePath: filepath.Join(runtimeDir, "seed.iso"),
		Disk: rtm.DiskPlan{
			DiskPath: filepath.Join(runtimeDir, "disk.qcow2"),
		},
		Networks: rtm.NetworkPlan{
			Management: rtm.ManagementNetworkPlan{
				SSHHost:       "127.0.0.1",
				SSHPort:       2222,
				InterfaceName: "yeastmgmt0",
				MACAddress:    "52:54:00:11:22:33",
			},
		},
	}

	if _, err := NewRuntime().Start(context.Background(), plan); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale qmp socket to be removed, stat err=%v", err)
	}
}

func TestStartCommandContextDoesNotTieReleasedProcessToContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	logPath := filepath.Join(t.TempDir(), "process.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	handle, err := startCommandContext(ctx, "sleep", []string{"60"}, logFile, logFile)
	if err != nil {
		t.Fatalf("startCommandContext returned error: %v", err)
	}
	pid := handle.PID()
	defer func() {
		_ = signalProcess(pid, syscall.SIGKILL)
	}()

	parentSID, err := unix.Getsid(0)
	if err != nil {
		t.Fatalf("get parent session id: %v", err)
	}
	childSID, err := unix.Getsid(pid)
	if err != nil {
		t.Fatalf("get child session id: %v", err)
	}
	if childSID == parentSID {
		t.Fatal("expected released process to run in a separate session")
	}

	if err := handle.Release(); err != nil {
		t.Fatalf("release handle: %v", err)
	}

	cancel()
	time.Sleep(200 * time.Millisecond)

	running, err := processRunning(pid)
	if err != nil {
		t.Fatalf("inspect process: %v", err)
	}
	if !running {
		t.Fatal("expected released process to survive context cancellation")
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

func TestRuntimeStopReturnsErrorWhenProcessStillRunningAfterSIGKILL(t *testing.T) {
	previousProcessRunning := processRunningFn
	previousSignalProcess := signalProcessFn
	previousWaitForProcessExit := waitForProcessExitFn
	defer func() {
		processRunningFn = previousProcessRunning
		signalProcessFn = previousSignalProcess
		waitForProcessExitFn = previousWaitForProcessExit
	}()

	processRunningFn = func(pid int) (bool, error) {
		return true, nil
	}
	signalProcessFn = func(pid int, sig syscall.Signal) error {
		return nil
	}
	waitForProcessExitFn = func(ctx context.Context, pid int, timeout time.Duration) error {
		return context.DeadlineExceeded
	}

	err := NewRuntime().Stop(context.Background(), rtm.RuntimeInstance{PID: 4242}, 2*time.Second)
	if err == nil {
		t.Fatal("expected stop to fail when process survives SIGKILL")
	}
	if !strings.Contains(err.Error(), "after SIGKILL") {
		t.Fatalf("expected SIGKILL wait error, got %v", err)
	}
}

func TestRuntimeFindProcessesByRuntimeDir(t *testing.T) {
	t.Parallel()

	runtimeDir := filepath.Join(t.TempDir(), "instances", "web")
	socketArg := "unix:" + qmpSocketPath(runtimeDir) + ",server,nowait"

	cmd := exec.Command("sh", "-c", `while :; do sleep 1; done`, "sh", socketArg)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	results, err := NewRuntime().FindProcesses(context.Background(), []rtm.CleanupTarget{
		{Name: "web", RuntimeDir: runtimeDir},
	})
	if err != nil {
		t.Fatalf("FindProcesses returned error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected a matching process to be found")
	}

	found := false
	for _, result := range results {
		if result.Name == "web" && result.PID == cmd.Process.Pid {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pid %d in results, got %#v", cmd.Process.Pid, results)
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
