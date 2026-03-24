package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// Lock strategy:
//   - We use an exclusive lock file (<state>.lock) created via O_CREATE|O_EXCL.
//   - Lock metadata stores owner PID + creation time.
//   - On contention, we detect stale locks when the owner PID is no longer running
//     (or metadata is malformed and older than StaleAfter), then remove and retry.
type LockOptions struct {
	AcquireTimeout time.Duration
	RetryInterval  time.Duration
	StaleAfter     time.Duration
}

type FileLock struct {
	path string
}

type lockInfo struct {
	PID       int       `json:"pid"`
	CreatedAt time.Time `json:"created_at"`
}

func DefaultLockOptions() LockOptions {
	return LockOptions{
		AcquireTimeout: 30 * time.Second,
		RetryInterval:  100 * time.Millisecond,
		StaleAfter:     2 * time.Minute,
	}
}

func AcquireFileLock(stateFilename string, opts LockOptions) (*FileLock, error) {
	if opts.AcquireTimeout <= 0 {
		opts.AcquireTimeout = 30 * time.Second
	}
	if opts.RetryInterval <= 0 {
		opts.RetryInterval = 100 * time.Millisecond
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = 2 * time.Minute
	}

	lockPath := lockPathForState(stateFilename)
	deadline := time.Now().Add(opts.AcquireTimeout)
	for {
		if err := tryCreateLock(lockPath); err == nil {
			return &FileLock{path: lockPath}, nil
		} else if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("failed to create lock file %s: %w", lockPath, err)
		}

		stale, owner, staleReason := isStaleLock(lockPath, opts.StaleAfter)
		if stale {
			if err := os.Remove(lockPath); err == nil || os.IsNotExist(err) {
				// Try again immediately after stale lock cleanup.
				continue
			}
		}

		if time.Now().After(deadline) {
			msg := fmt.Sprintf("timed out waiting for state lock %s", lockPath)
			if owner != "" {
				msg += " (" + owner + ")"
			}
			if staleReason != "" {
				msg += " [" + staleReason + "]"
			}
			return nil, errors.New(msg)
		}

		time.Sleep(opts.RetryInterval)
	}
}

func (l *FileLock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	err := os.Remove(l.path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("failed to remove state lock %s: %w", l.path, err)
}

func WithLockedState(filename string, opts LockOptions, fn func(*State) error) (retErr error) {
	mu := inProcessMutexFor(filename)
	mu.Lock()
	defer mu.Unlock()

	lock, err := AcquireFileLock(filename, opts)
	if err != nil {
		return err
	}
	defer func() {
		if releaseErr := lock.Release(); releaseErr != nil && retErr == nil {
			retErr = releaseErr
		}
	}()

	s, err := Load(filename)
	if err != nil {
		return err
	}

	if err := fn(s); err != nil {
		return err
	}

	return s.Save(filename)
}

func lockPathForState(stateFilename string) string {
	return stateFilename + ".lock"
}

var inProcessStateLocks sync.Map

func inProcessMutexFor(stateFilename string) *sync.Mutex {
	if v, ok := inProcessStateLocks.Load(stateFilename); ok {
		return v.(*sync.Mutex)
	}
	mu := &sync.Mutex{}
	actual, _ := inProcessStateLocks.LoadOrStore(stateFilename, mu)
	return actual.(*sync.Mutex)
}

func tryCreateLock(lockPath string) error {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	info := lockInfo{
		PID:       os.Getpid(),
		CreatedAt: time.Now().UTC(),
	}
	return json.NewEncoder(f).Encode(info)
}

func isStaleLock(lockPath string, staleAfter time.Duration) (bool, string, string) {
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "", "lock file disappeared"
		}
		return false, "", ""
	}

	var info lockInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		fileInfo, statErr := os.Stat(lockPath)
		if statErr == nil && time.Since(fileInfo.ModTime()) > staleAfter {
			return true, "", "malformed stale lock metadata"
		}
		return false, "lock owner unknown", ""
	}

	owner := fmt.Sprintf("held by pid %d since %s", info.PID, info.CreatedAt.Format(time.RFC3339))
	if info.PID <= 0 {
		if time.Since(info.CreatedAt) > staleAfter {
			return true, owner, "invalid PID in stale lock metadata"
		}
		return false, owner, ""
	}

	if !IsProcessRunning(info.PID) {
		return true, owner, "owner process is not running"
	}

	// Active process still running; treat lock as valid.
	return false, owner, ""
}
