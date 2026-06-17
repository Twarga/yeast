package ui

import (
	"io"
	"os"
	"strings"
)

// ShouldAnimate reports whether animated progress (spinners, transitions)
// should be shown for the given writer.
//
// Animation is disabled when:
//   - the writer is not a terminal
//   - NO_COLOR is set
//   - TERM=dumb
//   - YEAST_ANIMATION=0
//   - CI=true (common in GitHub Actions, CircleCI, etc.)
func ShouldAnimate(w io.Writer) bool {
	if os.Getenv("YEAST_ANIMATION") == "0" {
		return false
	}
	if os.Getenv("CI") == "true" || os.Getenv("CI") == "1" {
		return false
	}
	return StylingEnabled(w)
}

// ShouldColor reports whether colored output should be emitted for w.
// This is equivalent to StylingEnabled but named for clarity in callers
// that only care about color, not animation.
func ShouldColor(w io.Writer) bool {
	return StylingEnabled(w)
}

// IsCI returns true when running in a known CI environment.
func IsCI() bool {
	ci := os.Getenv("CI")
	return ci == "true" || ci == "1"
}

// IsDumbTerminal returns true when TERM=dumb or NO_COLOR is set.
func IsDumbTerminal() bool {
	return os.Getenv("NO_COLOR") != "" || strings.ToLower(os.Getenv("TERM")) == "dumb"
}
