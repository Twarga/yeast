package state

import "sort"

func SortedSnapshots(instance InstanceState) []SnapshotState {
	if len(instance.Snapshots) == 0 {
		return nil
	}

	snapshots := make([]SnapshotState, 0, len(instance.Snapshots))
	for _, snapshot := range instance.Snapshots {
		snapshots = append(snapshots, snapshot)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].CreatedAt.Equal(snapshots[j].CreatedAt) {
			return snapshots[i].Name < snapshots[j].Name
		}
		return snapshots[i].CreatedAt.Before(snapshots[j].CreatedAt)
	})

	return snapshots
}
