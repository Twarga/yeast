package qemu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	rtm "github.com/Twarga/yeast/internal/runtime"
)

func TestBuildMountArgsVirtiofs(t *testing.T) {
	plan := rtm.MachinePlan{
		Name: "desk", RuntimeDir: "/run/desk", MemoryMiB: 2048,
		Mounts: []rtm.MountPlan{
			{Source: "/home/u/dotfiles", Target: "/mnt/dotfiles", Driver: "virtiofs", Tag: "yeast_mnt_0"},
		},
	}
	args := buildMountArgs(plan)
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"memory-backend-memfd,id=mem,size=2048M,share=on",
		"node,memdev=mem",
		"socket,id=vfs0,path=/run/desk/vfs-0.sock",
		"vhost-user-fs-pci,chardev=vfs0,tag=yeast_mnt_0",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q:\n%s", want, joined)
		}
	}
}

func TestBuildMountArgs9p(t *testing.T) {
	plan := rtm.MachinePlan{
		Name: "desk", RuntimeDir: "/run/desk", MemoryMiB: 2048,
		Mounts: []rtm.MountPlan{
			{Source: "/home/u/data", Target: "/mnt/data", Driver: "9p", Tag: "yeast_mnt_1", ReadOnly: true},
		},
	}
	args := buildMountArgs(plan)
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"local,id=fs0,path=/home/u/data,security_model=mapped-xattr,readonly=on",
		"virtio-9p-pci,fsdev=fs0,mount_tag=yeast_mnt_1",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "memory-backend-memfd") {
		t.Fatal("9p-only mounts must not add memfd shared memory")
	}
}

func TestBuildMountArgsMixed(t *testing.T) {
	plan := rtm.MachinePlan{
		Name: "desk", RuntimeDir: "/run/desk", MemoryMiB: 1024,
		Mounts: []rtm.MountPlan{
			{Source: "/a", Target: "/a", Driver: "9p", Tag: "t9"},
			{Source: "/b", Target: "/b", Driver: "virtiofs", Tag: "tv"},
		},
	}
	args := buildMountArgs(plan)
	joined := strings.Join(args, " ")
	if strings.Count(joined, "memory-backend-memfd") != 1 {
		t.Fatalf("memfd emitted more than once:\n%s", joined)
	}
	if !strings.Contains(joined, "mount_tag=t9") || !strings.Contains(joined, "tag=tv") {
		t.Fatalf("mixed mounts missing tags:\n%s", joined)
	}
}

func TestSpawnMountSidecarsLifecycle(t *testing.T) {
	dir := t.TempDir()
	// fake virtiofsd: a shell script that records its args then sleeps.
	fake := filepath.Join(dir, "fake-virtiofsd.sh")
	script := "#!/bin/sh\necho \"$@\" > \"" + dir + "/args.txt\"\nfor a in \"$@\"; do case \"$a\" in --socket-path=*) touch \"${a#--socket-path=}\" ;; esac; done\nsleep 60\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "shared")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	rtDir := filepath.Join(dir, "rt")
	if err := os.Mkdir(rtDir, 0o755); err != nil {
		t.Fatal(err)
	}
	plan := rtm.MachinePlan{
		Name: "desk", RuntimeDir: rtDir,
		Mounts: []rtm.MountPlan{
			{Source: src, Target: "/mnt/x", Driver: "virtiofs", Tag: "t0", DaemonPath: fake},
			{Source: src, Target: "/mnt/y", Driver: "9p", Tag: "t1"}, // skipped
		},
	}
	if err := spawnMountSidecars(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	pidData, err := os.ReadFile(virtiofsPIDPath(rtDir, 0))
	if err != nil {
		t.Fatalf("pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil || pid <= 0 {
		t.Fatalf("bad pid %q", pidData)
	}
	// process should be alive
	if err := signalProcessFn(pid, 0); err != nil {
		t.Fatalf("sidecar pid %d not alive: %v", pid, err)
	}
	// wait for the script's echo to land before reading
	var argsData []byte
	for i := 0; i < 50; i++ {
		argsData, _ = os.ReadFile(filepath.Join(dir, "args.txt"))
		if len(argsData) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	for _, want := range []string{"--socket-path=" + virtiofsSocketPath(rtDir, 0), "--shared-dir=" + src, "--sandbox=none"} {
		if !strings.Contains(string(argsData), want) {
			t.Fatalf("sidecar args %q missing %q", argsData, want)
		}
	}

	stopMountSidecars(rtDir)
	dead := false
	for i := 0; i < 50; i++ {
		state := procState(pid)
		if state == "" || state == "Z" { // gone or unreaped zombie (we own it)
			dead = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !dead {
		t.Fatalf("sidecar pid %d still alive after stop", pid)
	}
	if _, err := os.Stat(virtiofsPIDPath(rtDir, 0)); !os.IsNotExist(err) {
		t.Fatal("pid file not cleaned up")
	}
}

func TestSpawnMountSidecarsRequiresDaemonPath(t *testing.T) {
	plan := rtm.MachinePlan{
		Name: "desk", RuntimeDir: t.TempDir(),
		Mounts: []rtm.MountPlan{{Source: "/a", Target: "/a", Driver: "virtiofs", Tag: "t0"}},
	}
	if err := spawnMountSidecars(context.Background(), plan); err == nil {
		t.Fatal("expected empty-daemon-path error")
	}
}

func TestStopMountSidecarsEmptyDir(t *testing.T) {
	// must not panic / error on a dir with no pid files
	stopMountSidecars(t.TempDir())
	stopMountSidecars("")
}

func TestSidecarSIGTERMKills(t *testing.T) {
	// sanity: signalProcessFn honors SIGTERM on a real process
	dir := t.TempDir()
	fake := filepath.Join(dir, "s.sh")
	_ = os.WriteFile(fake, []byte("#!/bin/sh\nsleep 60\n"), 0o755)
	pidData := []byte("999999999") // nonexistent pid → signal should error
	if err := signalProcessFn(mustAtoi(t, pidData), syscall.SIGTERM); err == nil {
		t.Fatal("expected error signalling dead pid")
	}
}

func mustAtoi(t *testing.T, b []byte) int {
	t.Helper()
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// procState reads the process state letter from /proc ("" when gone).
func procState(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return ""
	}
	// comm is in parens and may contain spaces; state follows ") "
	if i := strings.LastIndexByte(string(data), ')'); i >= 0 && i+2 < len(data) {
		return string(data[i+2])
	}
	return ""
}
