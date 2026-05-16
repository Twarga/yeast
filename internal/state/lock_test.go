package state

import (
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

func TestDoubleAcquireTimesOut(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	first, err := Acquire(path, testLockOptions())
	if err != nil {
		t.Fatalf("first Acquire returned error: %v", err)
	}
	defer first.Release()

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

func testLockOptions() LockOptions {
	return LockOptions{
		AcquireTimeout: time.Second,
		RetryInterval:  10 * time.Millisecond,
		StaleAfter:     time.Minute,
		Now:            time.Now,
		IsProcessAlive: func(pid int) bool { return true },
	}
}
