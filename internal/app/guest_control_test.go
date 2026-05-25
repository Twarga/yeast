package app

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestCommandStringJoinsPartsForStableResultShape(t *testing.T) {
	t.Parallel()

	got := commandString([]string{"bash", "-lc", "echo hi"})
	if got != "bash -lc echo hi" {
		t.Fatalf("unexpected command string: %q", got)
	}
}

func TestCleanLocalPathNormalizesLocalCopyPaths(t *testing.T) {
	t.Parallel()

	got := cleanLocalPath("./artifacts/../artifacts/out.txt")
	want := filepath.Clean("./artifacts/../artifacts/out.txt")
	if got != want {
		t.Fatalf("unexpected cleaned path: got %q want %q", got, want)
	}
}

func TestGuestControlResultShapesExposeExpectedFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	run := GuestCommandResult{
		Command:    "whoami",
		ExitCode:   0,
		Stdout:     "yeast\n",
		Stderr:     "",
		StartedAt:  now,
		FinishedAt: now.Add(200 * time.Millisecond),
		Duration:   200 * time.Millisecond,
		TimedOut:   false,
	}
	execResult := ExecResult{
		ProjectID: "proj_0123456789abcdef01234567",
		Instance:  "web",
		Run:       run,
	}
	copyResult := CopyResult{
		ProjectID:   "proj_0123456789abcdef01234567",
		Instance:    "web",
		Direction:   CopyToGuest,
		Source:      "/tmp/local.txt",
		Destination: "/home/yeast/local.txt",
		StartedAt:   now,
		FinishedAt:  now.Add(time.Second),
		Duration:    time.Second,
	}
	inspectResult := InspectResult{
		ProjectID: "proj_0123456789abcdef01234567",
		Instance: StatusInstanceResult{
			Name:         "web",
			Status:       "running",
			ManagementIP: "127.0.0.1",
			SSHPort:      2205,
			LabIP:        "10.10.10.10",
			RuntimeDir:   "/tmp/runtime",
		},
		SnapshotNames: []string{"clean", "post"},
		SnapshotCount: 2,
	}
	logsResult := LogsResult{
		ProjectID: "proj_0123456789abcdef01234567",
		Instance:  "web",
		LogPath:   "/tmp/runtime/vm.log",
		Content:   "booted\n",
	}

	if execResult.Run.Command != "whoami" || execResult.Run.ExitCode != 0 {
		t.Fatalf("unexpected exec result: %#v", execResult)
	}
	if copyResult.Direction != CopyToGuest {
		t.Fatalf("unexpected copy direction: %#v", copyResult)
	}
	if !reflect.DeepEqual(inspectResult.SnapshotNames, []string{"clean", "post"}) {
		t.Fatalf("unexpected inspect snapshot names: %#v", inspectResult)
	}
	if logsResult.LogPath == "" || logsResult.Content == "" {
		t.Fatalf("unexpected logs result: %#v", logsResult)
	}
}
