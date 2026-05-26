package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"yeast/internal/config"
)

func TestLabsBackeryExampleMaterializesValidYeastProject(t *testing.T) {
	t.Parallel()

	source := filepath.Join("..", "..", "examples", "labsbackery-attacker-target-basic")
	template, err := LoadLocal(source)
	if err != nil {
		t.Fatalf("LoadLocal returned error: %v", err)
	}
	if template.Metadata.Name != "attacker-target-basic" {
		t.Fatalf("unexpected template name %q", template.Metadata.Name)
	}

	destination := t.TempDir()
	result, err := Materialize(template, MaterializeOptions{Destination: destination})
	if err != nil {
		t.Fatalf("Materialize returned error: %v", err)
	}
	if len(result.Files) != len(template.Metadata.Files) {
		t.Fatalf("expected %d files, got %#v", len(template.Metadata.Files), result.Files)
	}
	if _, err := config.Load(filepath.Join(destination, "yeast.yaml")); err != nil {
		t.Fatalf("generated invalid yeast.yaml: %v", err)
	}

	for _, file := range []string{
		"lab.yaml",
		"files/target/flag.txt",
		"scenario/instructions.md",
		"scenario/checks.yaml",
	} {
		if _, err := os.Stat(filepath.Join(destination, filepath.FromSlash(file))); err != nil {
			t.Fatalf("expected materialized file %s: %v", file, err)
		}
	}

	checks, err := os.ReadFile(filepath.Join(destination, "scenario", "checks.yaml"))
	if err != nil {
		t.Fatalf("read checks: %v", err)
	}
	for _, want := range []string{"attacker-reaches-target-ssh", "target-marker-file"} {
		if strings.Contains(string(checks), want) {
			continue
		}
		t.Fatalf("expected checks to include %q, got:\n%s", want, string(checks))
	}

	readme, err := os.ReadFile(filepath.Join(destination, "README.md"))
	if err != nil {
		t.Fatalf("read readme: %v", err)
	}
	if !strings.Contains(string(readme), "yeast exec attacker --json") {
		t.Fatalf("expected README to document check execution, got:\n%s", string(readme))
	}
}
