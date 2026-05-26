package main

import (
	"bytes"
	"strings"
	"testing"
	"yeast/internal/app"
)

func TestCommandHelpRendersForV1Surface(t *testing.T) {
	commands := map[string]string{
		"doctor":          "Usage:\n  yeast doctor",
		"init":            "Usage:\n  yeast init",
		"pull":            "Usage:\n  yeast pull [image]",
		"up":              "Usage:\n  yeast up",
		"provision":       "Usage:\n  yeast provision [instance]",
		"snapshot":        "Usage:\n  yeast snapshot <instance> <name>",
		"restore":         "Usage:\n  yeast restore <instance> <name>",
		"snapshots":       "Usage:\n  yeast snapshots <instance>",
		"delete-snapshot": "Usage:\n  yeast delete-snapshot <instance> <name>",
		"status":          "Usage:\n  yeast status",
		"exec":            "Usage:\n  yeast exec [instance] -- <command...>",
		"copy":            "Usage:\n  yeast copy <instance> [--to-guest|--from-guest] <source> <destination>",
		"logs":            "Usage:\n  yeast logs <instance>",
		"inspect":         "Usage:\n  yeast inspect <instance>",
		"ssh":             "Usage:\n  yeast ssh [instance]",
		"down":            "Usage:\n  yeast down",
		"destroy":         "Usage:\n  yeast destroy",
		"version":         "Usage:\n  yeast version",
		"docs":            "Usage:\n  yeast docs [topic]",
	}

	for command, usage := range commands {
		command := command
		usage := usage
		t.Run(command, func(t *testing.T) {
			root := newRootCmd(app.NewService())
			root.SetArgs([]string{command, "--help"})

			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)

			if err := root.Execute(); err != nil {
				t.Fatalf("help returned error: %v\noutput: %s", err, buf.String())
			}
			output := buf.String()
			if !strings.Contains(output, usage) {
				t.Fatalf("help output missing usage %q:\n%s", usage, output)
			}
			if !strings.Contains(output, "Flags:") {
				t.Fatalf("help output missing flags section:\n%s", output)
			}
		})
	}
}
