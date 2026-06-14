package guest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
)

const (
	DefaultPollInterval = 250 * time.Millisecond
	DefaultDialTimeout  = 1 * time.Second

	fastPollInterval = 100 * time.Millisecond
	fastPollDuration = 5 * time.Second
	slowPollInterval = 500 * time.Millisecond
	fastDialTimeout  = 200 * time.Millisecond
	slowDialTimeout  = 2 * time.Second
	fastDialAttempts = 10
)

type ReadinessOptions struct {
	Address      string
	Timeout      time.Duration
	PollInterval time.Duration
	DialTimeout  time.Duration
	Dial         DialFunc
}

type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)

func WaitForTCP(ctx context.Context, options ReadinessOptions) error {
	if options.Address == "" {
		return fmt.Errorf("address is required")
	}
	if options.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}

	dial := options.Dial
	if dial == nil {
		dial = defaultDialContext
	}

	deadline := time.Now().Add(options.Timeout)
	start := time.Now()
	var lastErr error
	attempts := 0

	for {
		attempts++

		// Adaptive dial timeout: fast early, slow later.
		dialTimeout := options.DialTimeout
		if dialTimeout <= 0 {
			if attempts <= fastDialAttempts {
				dialTimeout = fastDialTimeout
			} else {
				dialTimeout = slowDialTimeout
			}
		}

		attemptCtx, cancel := context.WithTimeout(ctx, dialTimeout)
		conn, err := dial(attemptCtx, "tcp", options.Address)
		cancel()
		if err == nil {
			_ = conn.Close()
			return nil
		}

		lastErr = err
		if !retryableDialError(err) {
			return fmt.Errorf("connect to %s: %w", options.Address, err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for tcp readiness on %s after %s: %w", options.Address, options.Timeout, lastErr)
		}

		// Adaptive poll interval: fast at start, slow later.
		pollInterval := options.PollInterval
		if pollInterval <= 0 {
			if time.Since(start) < fastPollDuration {
				pollInterval = fastPollInterval
			} else {
				pollInterval = slowPollInterval
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func defaultDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, network, address)
}

func retryableDialError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.EHOSTUNREACH) || errors.Is(err, syscall.ENETUNREACH) {
		return true
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return retryableDialError(opErr.Err)
	}

	return false
}
