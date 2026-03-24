package main

import (
	"fmt"
	"os"
	"yeast/pkg/state"
)

func releaseStateLock(lock *state.FileLock) {
	if lock == nil {
		return
	}
	if err := lock.Release(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to release state lock: %v\n", err)
	}
}
