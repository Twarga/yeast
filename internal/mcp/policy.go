package mcp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Policy enforces server-side agent guardrails from .yeast/agent-policy.yaml.
// Client-side MCP annotations are hints only — this file is authoritative.
type Policy struct {
	// Allow lists permitted tools; empty means all tools allowed.
	Allow []string `yaml:"allow"`
	// Deny lists forbidden tools (wins over Allow).
	Deny []string `yaml:"deny"`
	// MaxExecTimeout caps vm_exec timeouts regardless of the request.
	MaxExecTimeout time.Duration `yaml:"max_exec_timeout"`
	// SnapshotBeforeExec takes a checkpoint before every vm_exec.
	SnapshotBeforeExec bool `yaml:"snapshot_before_exec"`
}

const policyFileName = "agent-policy.yaml"

// LoadPolicy reads .yeast/agent-policy.yaml under root; a missing file means
// the permissive default (all tools allowed).
func LoadPolicy(root string) (*Policy, error) {
	path := filepath.Join(root, ".yeast", policyFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Policy{}, nil
		}
		return nil, err
	}
	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &p, nil
}

// Allows reports whether a tool call is permitted by policy.
func (p *Policy) Allows(tool string) bool {
	for _, d := range p.Deny {
		if d == tool || d == "*" {
			return false
		}
	}
	if len(p.Allow) == 0 {
		return true
	}
	for _, a := range p.Allow {
		if a == tool || a == "*" {
			return true
		}
	}
	return false
}

// ClampExecTimeout caps a requested duration to the policy maximum.
func (p *Policy) ClampExecTimeout(d time.Duration) time.Duration {
	if p.MaxExecTimeout <= 0 {
		return d
	}
	if d <= 0 || d > p.MaxExecTimeout {
		return p.MaxExecTimeout
	}
	return d
}

// newLineScanner returns a bufio.Scanner for newline-delimited JSON-RPC.
func newLineScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	return s
}
