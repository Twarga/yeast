package guest

import (
	"reflect"
	"testing"
)

func TestSSHAddress(t *testing.T) {
	t.Parallel()

	address, err := SSHAddress("127.0.0.1", 2222)
	if err != nil {
		t.Fatalf("SSHAddress returned error: %v", err)
	}
	if address != "127.0.0.1:2222" {
		t.Fatalf("unexpected address: got %q", address)
	}
}

func TestBuildSSHArgs(t *testing.T) {
	t.Parallel()

	got, err := BuildSSHArgs("yeast", "127.0.0.1", 2222)
	if err != nil {
		t.Fatalf("BuildSSHArgs returned error: %v", err)
	}

	want := []string{
		"-p", "2222",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"yeast@127.0.0.1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected ssh args:\n got: %#v\nwant: %#v", got, want)
	}
}
