package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rtm "github.com/Twarga/yeast/internal/runtime"
)

// fakeLiveQMP is a scripted QMP server modeling a live-save conversation:
// query-block returns one virtio disk, blockdev-snapshot-sync creates the
// file QEMU would create, migrate writes memory.bin then reports completed.
func startLiveFakeQMP(t *testing.T, runtimeDir, diskPath string, fails map[string]string) {
	t.Helper()
	sock := qmpSocketPath(runtimeDir)
	os.Remove(sock)
	startFakeQMP(t, sock, func(req qmpRequest) map[string]any {
		if fail, ok := fails[req.Execute]; ok {
			return qmpErr(fail)
		}
		switch req.Execute {
		case "query-migrate-capabilities":
			return map[string]any{"return": []any{
				map[string]any{"capability": "mapped-ram", "state": false},
				map[string]any{"capability": "multifd", "state": false},
			}}
		case "query-block":
			return map[string]any{"return": []any{
				map[string]any{"device": "virtio0", "inserted": map[string]any{
					"file": diskPath, "node-name": "#block100",
				}},
			}}
		case "blockdev-snapshot-sync":
			f := fmt.Sprint(req.Arguments["snapshot-file"])
			_ = os.MkdirAll(filepath.Dir(f), 0755)
			_ = os.WriteFile(f, []byte("overlay"), 0644)
			return okResp()
		case "migrate":
			uri := fmt.Sprint(req.Arguments["uri"])
			path := strings.TrimPrefix(uri, "file:")
			_ = os.WriteFile(path, make([]byte, 1024), 0644)
			return okResp()
		case "query-migrate":
			return map[string]any{"return": map[string]any{"status": "completed"}}
		case "query-status":
			return map[string]any{"return": map[string]any{"status": "paused"}}
		case "query-jobs":
			return map[string]any{"return": []any{}}
		default:
			return okResp()
		}
	})
}

func TestSaveLiveSnapshotHappyPath(t *testing.T) {
	rt := t.TempDir()
	snapDir := filepath.Join(rt, "snapshots", "snap1")
	disk := filepath.Join(rt, "disk.qcow2")
	if err := os.WriteFile(disk, []byte("diskdata"), 0644); err != nil {
		t.Fatal(err)
	}
	startLiveFakeQMP(t, rt, disk, nil)

	r := &Runtime{}
	info, err := r.SaveLiveSnapshot(context.Background(), rtm.LiveSnapshotPlan{
		Instance:    rtm.RuntimeInstance{Name: "desk", RuntimeDir: rt, PID: 4242},
		SnapshotDir: snapDir,
		MemoryMiB:   512,
		Plan:        rtm.MachinePlan{Name: "desk", RuntimeDir: rt, MemoryMiB: 512},
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.Method != "mapped-ram" {
		t.Fatalf("method = %q want mapped-ram", info.Method)
	}
	for _, f := range []string{"base.qcow2", "overlay.qcow2", "memory.bin", "plan.json"} {
		if _, err := os.Stat(filepath.Join(snapDir, f)); err != nil {
			t.Fatalf("missing %s: %v", f, err)
		}
	}
	// frozen base is read-only and holds the original disk content
	base, _ := os.ReadFile(filepath.Join(snapDir, "base.qcow2"))
	if string(base) != "diskdata" {
		t.Fatalf("base content = %q", base)
	}
	st, _ := os.Stat(filepath.Join(snapDir, "base.qcow2"))
	if st.Mode().Perm() != 0444 {
		t.Fatalf("base perms = %o", st.Mode().Perm())
	}
}

func TestSaveLiveSnapshotResumeOnFailure(t *testing.T) {
	rt := t.TempDir()
	disk := filepath.Join(rt, "disk.qcow2")
	os.WriteFile(disk, []byte("d"), 0644)
	// migrate fails → guest must still be resumed
	var sent []string
	sock := qmpSocketPath(rt)
	os.Remove(sock)
	startFakeQMP(t, sock, func(req qmpRequest) map[string]any {
		sent = append(sent, req.Execute)
		switch req.Execute {
		case "query-migrate-capabilities":
			return map[string]any{"return": []any{}}
		case "query-block":
			return map[string]any{"return": []any{
				map[string]any{"device": "virtio0", "inserted": map[string]any{"file": disk}},
			}}
		case "migrate":
			return qmpErr("io error writing")
		default:
			return okResp()
		}
	})
	r := &Runtime{}
	_, err := r.SaveLiveSnapshot(context.Background(), rtm.LiveSnapshotPlan{
		Instance:    rtm.RuntimeInstance{Name: "d", RuntimeDir: rt},
		SnapshotDir: filepath.Join(rt, "snapshots", "s"),
		MemoryMiB:   256,
	})
	if err == nil {
		t.Fatal("expected migrate failure")
	}
	// last command sent should be cont (resume after failure)
	if len(sent) == 0 || sent[len(sent)-1] != "cont" {
		t.Fatalf("guest not resumed after failure, sent=%v", sent)
	}
}

func TestSaveLiveSnapshotInsufficientSpace(t *testing.T) {
	rt := t.TempDir()
	disk := filepath.Join(rt, "disk.qcow2")
	os.WriteFile(disk, []byte("d"), 0644)
	startLiveFakeQMP(t, rt, disk, nil)

	old := statfsFreeFn
	statfsFreeFn = func(dir string) (uint64, error) { return 100 * 1024 * 1024, nil } // 100 MiB free
	defer func() { statfsFreeFn = old }()

	r := &Runtime{}
	_, err := r.SaveLiveSnapshot(context.Background(), rtm.LiveSnapshotPlan{
		Instance:    rtm.RuntimeInstance{Name: "d", RuntimeDir: rt},
		SnapshotDir: filepath.Join(rt, "snapshots", "s"),
		MemoryMiB:   4096, // needs 4096+512 MiB
	})
	if err == nil || !strings.Contains(err.Error(), "insufficient disk space") {
		t.Fatalf("expected space error, got %v", err)
	}
}

func TestSaveLiveSnapshotFallback(t *testing.T) {
	rt := t.TempDir()
	disk := filepath.Join(rt, "disk.qcow2")
	os.WriteFile(disk, []byte("d"), 0644)
	var cmds []string
	sock := qmpSocketPath(rt)
	os.Remove(sock)
	startFakeQMP(t, sock, func(req qmpRequest) map[string]any {
		cmds = append(cmds, req.Execute)
		switch req.Execute {
		case "query-migrate-capabilities":
			return map[string]any{"return": []any{}}
		case "query-block":
			return map[string]any{"return": []any{
				map[string]any{"device": "virtio0", "inserted": map[string]any{"file": disk}},
			}}
		case "blockdev-snapshot-sync":
			return qmpErr("The command blockdev-snapshot-sync has not been found")
		case "snapshot-save":
			return okResp()
		case "query-jobs":
			return map[string]any{"return": []any{}}
		default:
			return okResp()
		}
	})
	r := &Runtime{}
	_, err := r.SaveLiveSnapshot(context.Background(), rtm.LiveSnapshotPlan{
		Instance:    rtm.RuntimeInstance{Name: "d", RuntimeDir: rt},
		SnapshotDir: filepath.Join(rt, "snapshots", "s"),
		MemoryMiB:   256,
	})
	if err == nil {
		t.Fatal("expected error when blockdev-snapshot-sync is unsupported")
	}
	joined := strings.Join(cmds, ",")
	if !strings.Contains(joined, "cont") {
		t.Fatalf("expected VM to be resumed via cont on failure, cmds=%v", cmds)
	}
}

func TestRestoreLiveSnapshotHappyPath(t *testing.T) {
	rt := t.TempDir()
	snapDir := filepath.Join(rt, "snapshots", "snap1")
	base := filepath.Join(snapDir, "base.qcow2")
	mem := filepath.Join(snapDir, "memory.bin")
	makeQcow2(t, base, "")
	if err := os.WriteFile(mem, []byte("ram"), 0644); err != nil {
		t.Fatal(err)
	}
	// saved machine plan
	planData, _ := json.Marshal(rtm.MachinePlan{
		Name: "desk", MemoryMiB: 512, CPUs: 2,
		Disk:          rtm.DiskPlan{DiskPath: filepath.Join(rt, "disk.qcow2")},
		SeedImagePath: filepath.Join(rt, "seed.iso"),
		Networks:      rtm.NetworkPlan{Management: rtm.ManagementNetworkPlan{SSHPort: 2222, SSHHost: "127.0.0.1", InterfaceName: "eth0", MACAddress: "52:54:00:12:34:56"}},
	})
	os.WriteFile(filepath.Join(snapDir, "plan.json"), planData, 0644)

	// Fake QEMU spawn: the socket appears when QEMU "starts" (mirrors -qmp).
	sock := qmpSocketPath(rt)
	os.Remove(sock)
	var spawnedArgs []string
	var gotIncoming string
	oldStart := startProcess
	startProcess = func(ctx context.Context, name string, args []string, stdout, stderr *os.File) (processHandle, error) {
		spawnedArgs = args
		startFakeQMP(t, sock, func(req qmpRequest) map[string]any {
			switch req.Execute {
			case "query-migrate-capabilities":
				return map[string]any{"return": []any{map[string]any{"capability": "mapped-ram", "state": false}}}
			case "migrate-incoming":
				gotIncoming = fmt.Sprint(req.Arguments["uri"])
				return okResp()
			case "query-status":
				return map[string]any{"return": map[string]any{"status": "paused"}}
			case "query-migrate":
				return map[string]any{"return": map[string]any{"status": "completed"}}
			default:
				return okResp()
			}
		})
		return &fakeProc{pid: 7777}, nil
	}
	defer func() { startProcess = oldStart }()

	r := &Runtime{}
	inst, err := r.RestoreLiveSnapshot(context.Background(), rtm.LiveRestorePlan{
		Instance:    rtm.RuntimeInstance{Name: "desk", RuntimeDir: rt},
		SnapshotDir: snapDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.PID != 7777 {
		t.Fatalf("pid = %d", inst.PID)
	}
	joined := strings.Join(spawnedArgs, " ")
	if !strings.Contains(joined, "-incoming defer") {
		t.Fatalf("args missing -incoming defer:\n%s", joined)
	}
	if gotIncoming != "file:"+mem {
		t.Fatalf("migrate-incoming uri = %q", gotIncoming)
	}
	// fresh overlay must chain on the snapshot base
	got, err := qemuImgBackingFile(context.Background(), filepath.Join(rt, "disk.qcow2"))
	if err != nil || filepath.Clean(got) != filepath.Clean(base) {
		t.Fatalf("restore overlay backing = %q (err %v) want %s", got, err, base)
	}
}

type fakeProc struct{ pid int }

func (f *fakeProc) PID() int       { return f.pid }
func (f *fakeProc) Release() error { return nil }

func TestRestoreLiveSnapshotMissingMemory(t *testing.T) {
	rt := t.TempDir()
	snapDir := filepath.Join(rt, "snapshots", "snap1")
	makeQcow2(t, filepath.Join(snapDir, "base.qcow2"), "")
	r := &Runtime{}
	_, err := r.RestoreLiveSnapshot(context.Background(), rtm.LiveRestorePlan{
		Instance: rtm.RuntimeInstance{Name: "d", RuntimeDir: rt}, SnapshotDir: snapDir,
	})
	if err == nil || !strings.Contains(err.Error(), "memory image") {
		t.Fatalf("expected missing-memory error, got %v", err)
	}
}
