package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"yeast/internal/app"
)

func TestInitListTemplatesJSON(t *testing.T) {
	previous := outputJSON
	outputJSON = false
	defer func() {
		outputJSON = previous
	}()

	root := newRootCmd(app.NewService())
	root.SetArgs([]string{"init", "--list-templates", "--json"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\noutput: %s", err, buf.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			Templates []app.TemplateSummary `json:"Templates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal rendered json: %v\npayload: %s", err, buf.String())
	}
	if !payload.OK {
		t.Fatalf("expected ok=true, got %#v", payload)
	}
	if payload.Command != "init" {
		t.Fatalf("expected command init, got %q", payload.Command)
	}
	if len(payload.Data.Templates) != 3 {
		t.Fatalf("expected 3 templates, got %#v", payload.Data.Templates)
	}
	if payload.Data.Templates[0].Name != "caddy-single-vm" {
		t.Fatalf("expected sorted templates, got %#v", payload.Data.Templates)
	}
}

func TestInitTemplateCreatesProject(t *testing.T) {
	previous := outputJSON
	outputJSON = false
	defer func() {
		outputJSON = previous
	}()

	projectRoot := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(previousDir); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir project root: %v", err)
	}

	root := newRootCmd(app.NewService())
	root.SetArgs([]string{"init", "--template", "caddy-single-vm"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\noutput: %s", err, buf.String())
	}

	for _, file := range []string{"yeast.yaml", "README.md", "site/Caddyfile", "site/index.html", ".yeast/project.json"} {
		if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(file))); err != nil {
			t.Fatalf("expected %s to exist: %v", file, err)
		}
	}
}
