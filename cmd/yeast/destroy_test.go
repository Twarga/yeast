package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDestroyPromptChoices(t *testing.T) {
	tests := []struct {
		name          string
		answer        string
		wantKeepFiles bool
	}{
		{name: "yes short", answer: "y\n", wantKeepFiles: false},
		{name: "yes long", answer: "yes\n", wantKeepFiles: false},
		{name: "default preserve", answer: "\n", wantKeepFiles: true},
		{name: "no preserve", answer: "n\n", wantKeepFiles: true},
		{name: "no long preserve", answer: "no\n", wantKeepFiles: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousPromptReader := destroyPromptReader
			previousTerminalCheck := destroyTerminalCheck
			previousJSON := outputJSON
			defer func() {
				destroyPromptReader = previousPromptReader
				destroyTerminalCheck = previousTerminalCheck
				outputJSON = previousJSON
			}()

			outputJSON = false
			destroyTerminalCheck = func(value any) bool { return true }
			destroyPromptReader = func(r io.Reader) (string, error) {
				return tt.answer, nil
			}

			cmd := &cobra.Command{}
			cmd.SetIn(bytes.NewBuffer(nil))
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			keepFiles, err := resolveDestroyKeepFilesMode(cmd, false, false)
			if err != nil {
				t.Fatalf("resolveDestroyKeepFilesMode returned error: %v", err)
			}
			if keepFiles != tt.wantKeepFiles {
				t.Fatalf("unexpected keep-files mode: got %v want %v", keepFiles, tt.wantKeepFiles)
			}
			if !strings.Contains(stderr.String(), "Delete local Yeast files for this project? [y/N]: ") {
				t.Fatalf("expected updated prompt to be shown, got %q", stderr.String())
			}
		})
	}
}

func TestResolveDestroyKeepFilesModeDefaultsToPreservingRuntimeFiles(t *testing.T) {
	previousPromptReader := destroyPromptReader
	previousTerminalCheck := destroyTerminalCheck
	previousJSON := outputJSON
	defer func() {
		destroyPromptReader = previousPromptReader
		destroyTerminalCheck = previousTerminalCheck
		outputJSON = previousJSON
	}()

	outputJSON = false
	destroyTerminalCheck = func(value any) bool { return true }
	destroyPromptReader = func(r io.Reader) (string, error) {
		return "\n", nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	keepFiles, err := resolveDestroyKeepFilesMode(cmd, false, false)
	if err != nil {
		t.Fatalf("resolveDestroyKeepFilesMode returned error: %v", err)
	}
	if !keepFiles {
		t.Fatal("expected empty prompt answer to preserve runtime files")
	}
	if !strings.Contains(stderr.String(), "Delete local Yeast files for this project? [y/N]: ") {
		t.Fatalf("expected safe default prompt, got %q", stderr.String())
	}
}

func TestResolveDestroyKeepFilesModeRejectsInvalidPromptAnswer(t *testing.T) {
	previousPromptReader := destroyPromptReader
	previousTerminalCheck := destroyTerminalCheck
	previousJSON := outputJSON
	defer func() {
		destroyPromptReader = previousPromptReader
		destroyTerminalCheck = previousTerminalCheck
		outputJSON = previousJSON
	}()

	outputJSON = false
	destroyTerminalCheck = func(value any) bool { return true }
	destroyPromptReader = func(r io.Reader) (string, error) {
		return "maybe\n", nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := resolveDestroyKeepFilesMode(cmd, false, false)
	if err == nil {
		t.Fatal("expected invalid prompt answer to return an error")
	}
	if !strings.Contains(err.Error(), "please answer y or n") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveDestroyKeepFilesModeSkipsPromptWhenFlagProvided(t *testing.T) {
	previousPromptReader := destroyPromptReader
	previousTerminalCheck := destroyTerminalCheck
	previousJSON := outputJSON
	defer func() {
		destroyPromptReader = previousPromptReader
		destroyTerminalCheck = previousTerminalCheck
		outputJSON = previousJSON
	}()

	outputJSON = false
	destroyTerminalCheck = func(value any) bool { return true }
	promptCalls := 0
	destroyPromptReader = func(r io.Reader) (string, error) {
		promptCalls++
		return "", nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	keepFiles, err := resolveDestroyKeepFilesMode(cmd, true, false)
	if err != nil {
		t.Fatalf("resolveDestroyKeepFilesMode returned error: %v", err)
	}
	if !keepFiles {
		t.Fatal("expected explicit keep-files flag to win")
	}
	if promptCalls != 0 {
		t.Fatalf("expected no prompt reads, got %d", promptCalls)
	}
}
