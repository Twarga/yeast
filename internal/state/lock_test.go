package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAcquireRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")

	lock, err := Acquire(path, testLockOptions())
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected lock file to exist: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release returned error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected lock file to be removed, stat err=%v", err)
	}
}

func TestAcquireCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "nested", "state.lock")

	lock, err := Acquire(path, testLockOptions())
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	defer func() { _ = lock.Release() }()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected lock file to exist: %v", err)
	}
}

func TestDoubleAcquireTimesOut(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	first, err := Acquire(path, testLockOptions())
	if err != nil {
		t.Fatalf("first Acquire returned error: %v", err)
	}
	defer func() {
		if err := first.Release(); err != nil {
			t.Fatalf("first Release returned error: %v", err)
		}
	}()

	_, err = Acquire(path, LockOptions{
		AcquireTimeout: 50 * time.Millisecond,
		RetryInterval:  10 * time.Millisecond,
		StaleAfter:     time.Minute,
		Now:            time.Now,
		IsProcessAlive: func(pid int) bool { return true },
	})
	if err == nil {
		t.Fatal("expected second acquire to time out")
	}
	if !strings.Contains(err.Error(), "timed out waiting for state lock") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func TestStaleLockRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	err := tryCreateLock(path, LockInfo{
		PID:       99999,
		CreatedAt: time.Now().Add(-10 * time.Minute).UTC(),
	})
	if err != nil {
		t.Fatalf("tryCreateLock returned error: %v", err)
	}

	lock, err := Acquire(path, LockOptions{
		AcquireTimeout: time.Second,
		RetryInterval:  10 * time.Millisecond,
		StaleAfter:     time.Second,
		Now:            time.Now,
		IsProcessAlive: func(pid int) bool { return false },
	})
	if err != nil {
		t.Fatalf("Acquire after stale lock returned error: %v", err)
	}
	_ = lock.Release()
}

func TestMalformedStaleLockRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	if err := os.WriteFile(path, []byte("{bad json"), 0600); err != nil {
		t.Fatalf("write malformed lock: %v", err)
	}
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("set lock mtime: %v", err)
	}

	lock, err := Acquire(path, LockOptions{
		AcquireTimeout: time.Second,
		RetryInterval:  10 * time.Millisecond,
		StaleAfter:     time.Second,
		Now:            time.Now,
		IsProcessAlive: func(pid int) bool { return false },
	})
	if err != nil {
		t.Fatalf("Acquire after malformed stale lock returned error: %v", err)
	}
	_ = lock.Release()
}

// TestDeadPIDLockReclaimedImmediately writes a lock whose recorded PID can
// never exist (above Linux's maximum pid value) with a fresh timestamp, then
// verifies Acquire reclaims it instantly instead of waiting out StaleAfter.
func TestDeadPIDLockReclaimedImmediately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")

	// kernel pid_max caps at 2^22, so this PID is guaranteed dead.
	deadPID := (1 << 22) + 1
	raw, err := json.Marshal(LockInfo{
		PID:       deadPID,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal lock info: %v", err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("write lock file: %v", err)
	}

	start := time.Now()
	lock, err := Acquire(path, LockOptions{
		AcquireTimeout: 5 * time.Second,
		RetryInterval:  10 * time.Millisecond,
		StaleAfter:     2 * time.Minute,
		Now:            time.Now,
		IsProcessAlive: processAlive,
	})
	if err != nil {
		t.Fatalf("Acquire after dead-pid lock returned error: %v", err)
	}
	defer func() { _ = lock.Release() }()

	if elapsed := time.Since(start); elapsed >= 50*time.Millisecond {
		t.Fatalf("expected immediate lock reclaim (<50ms), took %s", elapsed)
	}
}

// TestForeignHostLockNotReclaimedByPIDCheck ensures a lock written by another
// host is not reclaimed via the (unreliable cross-host) PID liveness check —
// it must fall back to the timestamp heuristic and keep waiting.
func TestForeignHostLockNotReclaimedByPIDCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname: %v", err)
	}
	foreign := "definitely-not-" + hostname
	raw, err := json.Marshal(LockInfo{
		PID:       (1 << 22) + 1,
		CreatedAt: time.Now().UTC(),
		Hostname:  foreign,
	})
	if err != nil {
		t.Fatalf("marshal lock info: %v", err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("write lock file: %v", err)
	}

	_, err = Acquire(path, LockOptions{
		AcquireTimeout: 120 * time.Millisecond,
		RetryInterval:  20 * time.Millisecond,
		StaleAfter:     time.Hour,
		Now:            time.Now,
		IsProcessAlive: func(pid int) bool { return false },
	})
	if err == nil {
		t.Fatal("expected foreign-host lock to keep waiting, got immediate acquire")
	}
	if !strings.Contains(err.Error(), "timed out waiting for state lock") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testLockOptions() LockOptions {
	return LockOptions{
		AcquireTimeout: time.Second,
		RetryInterval:  10 * time.Millisecond,
		StaleAfter:     time.Minute,
		Now:            time.Now,
		IsProcessAlive: func(pid int) bool { return true },
	}
}
