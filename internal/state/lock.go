package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LockOptions struct {
	AcquireTimeout time.Duration
	RetryInterval  time.Duration
	StaleAfter     time.Duration
	Now            func() time.Time
	IsProcessAlive func(pid int) bool
}

type FileLock struct {
	Path string
}

type LockInfo struct {
	PID       int       `json:"pid"`
	CreatedAt time.Time `json:"created_at"`
}

func DefaultLockOptions() LockOptions {
	return LockOptions{
		AcquireTimeout: 30 * time.Second,
		RetryInterval:  100 * time.Millisecond,
		StaleAfter:     2 * time.Minute,
		Now:            time.Now,
		IsProcessAlive: processAlive,
	}
}

func Acquire(path string, options LockOptions) (*FileLock, error) {
	opts := normalizeLockOptions(options)
	deadline := opts.Now().Add(opts.AcquireTimeout)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create lock directory %s: %w", filepath.Dir(path), err)
	}

	for {
		err := tryCreateLock(path, LockInfo{
			PID:       os.Getpid(),
			CreatedAt: opts.Now().UTC(),
		})
		if err == nil {
			return &FileLock{Path: path}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create lock file %s: %w", path, err)
		}

		stale, staleErr := removeIfStale(path, opts)
		if staleErr != nil {
			return nil, staleErr
		}
		if stale {
			continue
		}

		if !opts.Now().Before(deadline) {
			return nil, fmt.Errorf("timed out waiting for state lock %s", path)
		}
		time.Sleep(opts.RetryInterval)
	}
}

func (l *FileLock) Release() error {
	if l == nil || l.Path == "" {
		return nil
	}
	if err := os.Remove(l.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("release lock file %s: %w", l.Path, err)
	}
	return nil
}

func normalizeLockOptions(options LockOptions) LockOptions {
	opts := options
	if opts.AcquireTimeout <= 0 {
		opts.AcquireTimeout = 30 * time.Second
	}
	if opts.RetryInterval <= 0 {
		opts.RetryInterval = 100 * time.Millisecond
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = 2 * time.Minute
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.IsProcessAlive == nil {
		opts.IsProcessAlive = processAlive
	}
	return opts
}

func tryCreateLock(path string, info LockInfo) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(info)
}

func removeIfStale(path string, options LockOptions) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, fmt.Errorf("read lock file %s: %w", path, err)
	}

	var info LockInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		fileInfo, statErr := os.Stat(path)
		if statErr == nil && options.Now().Sub(fileInfo.ModTime()) > options.StaleAfter {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return false, fmt.Errorf("remove stale malformed lock %s: %w", path, err)
			}
			return true, nil
		}
		return false, nil
	}

	if info.PID > 0 && options.IsProcessAlive(info.PID) {
		return false, nil
	}
	if info.CreatedAt.IsZero() {
		fileInfo, statErr := os.Stat(path)
		if statErr == nil && options.Now().Sub(fileInfo.ModTime()) <= options.StaleAfter {
			return false, nil
		}
	}
	if !info.CreatedAt.IsZero() && options.Now().Sub(info.CreatedAt) <= options.StaleAfter && info.PID > 0 {
		return false, nil
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("remove stale lock %s: %w", path, err)
	}
	return true, nil
}
