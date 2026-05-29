package guest

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestWaitForTCPSucceedsAgainstListeningServer(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	err = WaitForTCP(context.Background(), ReadinessOptions{
		Address: listener.Addr().String(),
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("WaitForTCP returned error: %v", err)
	}
}

func TestWaitForTCPTimeout(t *testing.T) {
	t.Parallel()

	err := WaitForTCP(context.Background(), ReadinessOptions{
		Address:      "127.0.0.1:2222",
		Timeout:      300 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
		DialTimeout:  10 * time.Millisecond,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return nil, context.DeadlineExceeded
		},
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded in wrapped error, got %v", err)
	}
}

func TestWaitForTCPRetriesConnectionRefusedThenSucceeds(t *testing.T) {
	t.Parallel()

	attempts := 0

	err := WaitForTCP(context.Background(), ReadinessOptions{
		Address:      "127.0.0.1:2222",
		Timeout:      2 * time.Second,
		PollInterval: 10 * time.Millisecond,
		DialTimeout:  10 * time.Millisecond,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			attempts++
			if attempts < 3 {
				return nil, &net.OpError{Err: syscall.ECONNREFUSED}
			}

			server, client := net.Pipe()
			go func() {
				<-ctx.Done()
				_ = server.Close()
			}()
			_ = server.Close()
			return client, nil
		},
	})
	if err != nil {
		t.Fatalf("WaitForTCP returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 dial attempts, got %d", attempts)
	}
}
