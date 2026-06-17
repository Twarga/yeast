package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveDestroyKeepFilesModeAcceptsKeepFilesChoice(t *testing.T) {
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
		return "n\n", nil
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
		t.Fatal("expected keep-files mode to be selected")
	}
	if !strings.Contains(stderr.String(), "Delete VM runtime files too?") {
		t.Fatalf("expected prompt to be shown, got %q", stderr.String())
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
