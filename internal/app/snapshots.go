package app

import (
	"fmt"
	"yeast/internal/state"
)

func listInstanceSnapshots(currentState state.State, target string) ([]state.SnapshotState, error) {
	if target == "" {
		return nil, WrapError(ErrorCodeInvalidArgument, "snapshot target instance is required", nil)
	}

	instance, ok := currentState.Instances[target]
	if !ok {
		return nil, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", target), nil)
	}

	return state.SortedSnapshots(instance), nil
}
