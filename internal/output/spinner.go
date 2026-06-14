package output

import (
	"fmt"
	"io"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner renders an animated braille-pattern spinner on a single terminal line.
// Safe for concurrent use.
type Spinner struct {
	w       io.Writer
	mu      sync.Mutex
	stopped bool
	done    chan struct{}
	frame   int
	message string
}

// NewSpinner creates a spinner that writes to w. Call Start to begin animation.
func NewSpinner(w io.Writer) *Spinner {
	return &Spinner{w: w}
}

// Start begins the spinner animation. If a spinner is already running, it is
// stopped first (without printing a final line). The spinner only animates on
// TTY writers; on non-TTY writers, Start is a no-op (use PrintLine for output).
func (s *Spinner) Start(message string) {
	s.mu.Lock()
	if s.done != nil {
		s.mu.Unlock()
		s.Stop("", "")
		s.mu.Lock()
	}
	s.message = message
	s.stopped = false
	s.frame = 0
	s.done = make(chan struct{})
	go s.spin()
	s.mu.Unlock()
}

// Update changes the spinner text without stopping the animation.
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.message = message
}

// Stop halts the animation and prints a final line with the given symbol and
// message. Pass "" for message to use the last displayed message.
func (s *Spinner) Stop(symbol, message string) {
	s.mu.Lock()
	if s.done == nil || s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	close(s.done)
	done := s.done
	s.done = nil
	s.mu.Unlock()

	// Wait for goroutine to exit so it doesn't race with our print.
	<-done

	if message == "" {
		message = s.message
	}
	// Clear the spinner line, then print the final result.
	fmt.Fprintf(s.w, "\r\033[K  %s %s\n", symbol, message)
}

// PrintLine writes a standalone line (no spinner animation). Useful for
// non-TTY output or for lines that should not be animated.
func (s *Spinner) PrintLine(symbol, message string) {
	fmt.Fprintf(s.w, "  %s %s\n", symbol, message)
}

func (s *Spinner) spin() {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.stopped {
				s.mu.Unlock()
				return
			}
			frame := spinnerFrames[s.frame%len(spinnerFrames)]
			msg := s.message
			s.frame++
			s.mu.Unlock()
			fmt.Fprintf(s.w, "\r\033[K  %s %s", frame, msg)
		}
	}
}
