package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "Valid Full Config",
			yamlContent: `
project_name: test-project
machines:
  - name: web
    image: ubuntu.img
    template: web.yaml
    specs:
      cpus: 2
      memory: 2G
`,
			wantErr: false,
		},
		{
			name: "Valid Config With Defaults",
			yamlContent: `
project_name: test-project-defaults
machines:
  - name: db
    image: alpine.img
`,
			wantErr: false,
		},
		{
			name: "Missing Project Name",
			yamlContent: `
machines:
  - name: web
    image: ubuntu.img
`,
			wantErr: true,
			errMsg:  "project_name is required",
		},
		{
			name: "Duplicate Machine Name",
			yamlContent: `
project_name: dup-test
machines:
  - name: web
    image: ubuntu.img
  - name: web
    image: ubuntu2.img
`,
			wantErr: true,
			errMsg:  "duplicate machine name: web",
		},
		{
			name: "Missing Image",
			yamlContent: `
project_name: no-image-test
machines:
  - name: web
`,
			wantErr: true,
			errMsg:  "machine 'web' is missing an image",
		},
		{
			name:    "Empty File",
			yamlContent: "",
			wantErr: true,
			errMsg: "project_name is required", // Or generic parse error, but empty struct fails validation
		},
	}

	tmpDir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, "yeast.yaml")
			err := os.WriteFile(filePath, []byte(tt.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			cfg, err := ParseConfig(filePath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error msg '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cfg == nil {
					t.Errorf("Config should not be nil")
				}
				
				// Specific check for defaults
				if tt.name == "Valid Config With Defaults" {
					if cfg.Machines[0].Specs.CPUs != 1 {
						t.Errorf("Expected default CPUs to be 1, got %d", cfg.Machines[0].Specs.CPUs)
					}
					if cfg.Machines[0].Specs.Memory != "1G" {
						t.Errorf("Expected default Memory to be 1G, got %s", cfg.Machines[0].Specs.Memory)
					}
				}
			}
		})
	}
}

func TestParseConfig_FileNotFound(t *testing.T) {
	_, err := ParseConfig("non-existent-file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
