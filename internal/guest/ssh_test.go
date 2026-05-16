package guest

import "testing"

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
