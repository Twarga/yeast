package output

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
	"yeast/internal/app"
)

func NewProgressSink(w io.Writer) app.EventSink {
	styled := progressStylingEnabled(w)
	tty := isTerminal(w)

	if tty {
		return newTTYProgressSink(w, styled)
	}
	return newPlainProgressSink(w, styled)
}

func newPlainProgressSink(w io.Writer, styled bool) app.EventSink {
	mu := &sync.Mutex{}
	return func(event app.Event) {
		line := formatProgressEvent(event, styled)
		if line == "" {
			return
		}
		mu.Lock()
		fmt.Fprintln(w, line)
		mu.Unlock()
	}
}

func newTTYProgressSink(w io.Writer, styled bool) app.EventSink {
	mu := &sync.Mutex{}
	spinner := NewSpinner(w)

	return func(event app.Event) {
		symbol, text := progressEventParts(event, styled)
		if text == "" && symbol == "" {
			return
		}

		mu.Lock()
		defer mu.Unlock()

		switch event.Name {
		case app.EventProjectLoaded, app.EventConfigValidated,
			app.EventCloudInitGenerated, app.EventWorkflowCompleted:
			// Silent events — nothing to show.

		case app.EventWorkflowFailed:
			spinner.Stop(fail(styled), text)

		case app.EventImagePulling, app.EventVMStarting,
			app.EventSSHWaiting, app.EventProvisionStarted:
			// Transition events — show as animated spinner.
			prefix := instancePrefix(event.Instance)
			spinner.Start(prefix + text)

		default:
			// Completion / status events — stop spinner and print result line.
			prefix := instancePrefix(event.Instance)
			spinner.Stop(symbol, prefix+text)
		}
	}
}

func progressEventParts(event app.Event, styled bool) (symbol, text string) {
	switch event.Name {
	case app.EventProjectLoaded, app.EventConfigValidated,
		app.EventCloudInitGenerated, app.EventWorkflowCompleted:
		return "", ""

	case app.EventImagePulling:
		return progress(styled), "Pulling image..."
	case app.EventImageReady:
		return done(styled), "Image ready"
	case app.EventDiskReady:
		return done(styled), "Disk ready"
	case app.EventVMStarting:
		return progress(styled), "Starting VM..."
	case app.EventSSHWaiting:
		return progress(styled), "Waiting for SSH..."
	case app.EventSSHReady:
		return done(styled), "SSH ready"
	case app.EventProvisionStarted:
		return progress(styled), "Provisioning..."
	case app.EventProvisionFinished:
		return done(styled), "Provisioned"
	case app.EventProvisionSkipped:
		return skip(styled), "Provisioning skipped (unchanged)"
	case app.EventInstanceReady:
		return done(styled), "Ready"
	case app.EventInstanceStopped:
		return done(styled), "Stopped"
	case app.EventInstanceDestroyed:
		return done(styled), "Destroyed"
	case app.EventWorkflowFailed:
		text := "Failed"
		if event.Message != "" {
			text = event.Message
		}
		return fail(styled), text
	default:
		if event.Message != "" {
			return progress(styled), event.Message
		}
		return "", ""
	}
}

func formatProgressEvent(event app.Event, styled bool) string {
	symbol, text := progressEventParts(event, styled)
	if text == "" {
		return ""
	}
	prefix := instancePrefix(event.Instance)
	return fmt.Sprintf("  %s %s%s", symbol, prefix, text)
}

func instancePrefix(instance string) string {
	if instance != "" {
		return "[" + instance + "] "
	}
	return ""
}

func progress(styled bool) string {
	if styled {
		return "\033[33m●\033[0m"
	}
	return "*"
}

func done(styled bool) string {
	if styled {
		return "\033[32m✓\033[0m"
	}
	return "+"
}

func fail(styled bool) string {
	if styled {
		return "\033[31m✗\033[0m"
	}
	return "!"
}

func skip(styled bool) string {
	if styled {
		return "\033[33m⊘\033[0m"
	}
	return "-"
}

func isTerminal(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func progressStylingEnabled(w io.Writer) bool {
	return isTerminal(w)
}
