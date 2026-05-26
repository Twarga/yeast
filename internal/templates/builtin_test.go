package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"yeast/internal/config"
)

func TestOfficialBuiltinsMaterializeValidProjects(t *testing.T) {
	t.Parallel()

	builtins, err := Builtins()
	if err != nil {
		t.Fatalf("Builtins returned error: %v", err)
	}
	if len(builtins) != 3 {
		t.Fatalf("expected 3 official built-ins, got %d", len(builtins))
	}

	for _, template := range builtins {
		template := template
		t.Run(template.Metadata.Name, func(t *testing.T) {
			t.Parallel()

			destination := t.TempDir()
			result, err := Materialize(template, MaterializeOptions{Destination: destination})
			if err != nil {
				t.Fatalf("Materialize returned error: %v", err)
			}
			if len(result.Files) != len(template.Metadata.Files) {
				t.Fatalf("expected %d files, got %#v", len(template.Metadata.Files), result.Files)
			}

			configPath := filepath.Join(destination, "yeast.yaml")
			if _, err := config.Load(configPath); err != nil {
				t.Fatalf("template generated invalid yeast.yaml: %v", err)
			}

			readmePath := filepath.Join(destination, "README.md")
			raw, err := os.ReadFile(readmePath)
			if err != nil {
				t.Fatalf("read generated README: %v", err)
			}
			readme := string(raw)
			if !strings.Contains(readme, "yeast init --template "+template.Metadata.Name) {
				t.Fatalf("README should document template init command, got:\n%s", readme)
			}
			for _, forbidden := range []string{
				"/path/to/yeast/examples",
				"cp /path/to/yeast",
				"Yeast MCP",
				"Twarga Cloud",
			} {
				if strings.Contains(readme, forbidden) {
					t.Fatalf("README contains future/manual-copy language %q:\n%s", forbidden, readme)
				}
			}
		})
	}
}

func TestOfficialBuiltinsUseSupportedScope(t *testing.T) {
	t.Parallel()

	builtins, err := Builtins()
	if err != nil {
		t.Fatalf("Builtins returned error: %v", err)
	}
	for _, template := range builtins {
		for _, file := range template.Metadata.Files {
			forbidden := []string{
				"snapshot.yaml",
				"events.yaml",
				"mcp.yaml",
				"cloud.yaml",
				"labsbackery.yaml",
			}
			for _, value := range forbidden {
				if strings.EqualFold(file, value) || strings.Contains(strings.ToLower(file), strings.TrimSuffix(value, ".yaml")) {
					t.Fatalf("template %s includes future-scope file %q", template.Metadata.Name, file)
				}
			}
		}
	}
}
