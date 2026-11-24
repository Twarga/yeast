package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	rtm "github.com/Twarga/yeast/internal/runtime"
)

type InputOptions struct {
	ProjectRoot string
	Target      string
	Keys        []string // simultaneous chord, e.g. ["meta_l","ret"]
	Text        string
	HoldTimeMs  int
}

type InputResult struct {
	Instance string   `json:"instance"`
	Keys     []string `json:"keys,omitempty"`
	Text     string   `json:"text,omitempty"`
	Chars    int      `json:"chars,omitempty"`
}

// Input injects keystrokes into a display-enabled instance via QMP — no
// guest agent needed. Exactly one of Keys or Text is required.
func (s *Service) Input(ctx context.Context, options InputOptions) (result InputResult, err error) {
	started := time.Now()
	hasKeys := len(options.Keys) > 0
	hasText := options.Text != ""
	if hasKeys == hasText {
		return InputResult{}, WrapError(ErrorCodeInvalidArgument, "exactly one of --key or --text is required", nil)
	}
	dc, err := s.resolveDisplayInstance(ctx, options.ProjectRoot, options.Target)
	if err != nil {
		return InputResult{}, err
	}
	defer func() {
		root := options.ProjectRoot
		if root == "" {
			root = "."
		}
		if abs, absErr := filepath.Abs(root); absErr == nil {
			s.auditCall(ctx, abs, "input", map[string]any{"instance": options.Target}, started, err)
		}
	}()
	controller, ok := s.runtime.(rtm.DisplayController)
	if !ok {
		return InputResult{}, WrapError(ErrorCodePrecondition, "runtime does not support display control", nil)
	}

	if hasKeys {
		if err := controller.SendKeys(ctx, dc.runtime, options.Keys, options.HoldTimeMs); err != nil {
			return InputResult{}, WrapError(ErrorCodeRuntime, fmt.Sprintf("send keys to %s: %v", dc.instanceName, err), err)
		}
		return InputResult{Instance: dc.instanceName, Keys: options.Keys}, nil
	}
	if err := controller.SendText(ctx, dc.runtime, options.Text); err != nil {
		return InputResult{}, WrapError(ErrorCodeRuntime, fmt.Sprintf("type text into %s: %v", dc.instanceName, err), err)
	}
	return InputResult{Instance: dc.instanceName, Text: options.Text, Chars: len(options.Text)}, nil
}
