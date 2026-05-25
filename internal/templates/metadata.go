package templates

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const MetadataFileName = "template.yaml"

var templateNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Metadata struct {
	Name        string   `yaml:"name" json:"Name"`
	Title       string   `yaml:"title" json:"Title"`
	Description string   `yaml:"description" json:"Description"`
	Category    string   `yaml:"category" json:"Category"`
	Version     string   `yaml:"version" json:"Version"`
	Files       []string `yaml:"files" json:"Files"`
}

func DecodeMetadata(r io.Reader) (Metadata, error) {
	var metadata Metadata
	if err := yaml.NewDecoder(r).Decode(&metadata); err != nil {
		return Metadata{}, fmt.Errorf("parse template metadata: %w", err)
	}
	if err := ValidateMetadata(metadata); err != nil {
		return Metadata{}, err
	}
	return metadata, nil
}

func ValidateMetadata(metadata Metadata) error {
	if !isValidTemplateName(metadata.Name) {
		return fmt.Errorf("template name %q is invalid", metadata.Name)
	}
	if strings.TrimSpace(metadata.Title) == "" {
		return fmt.Errorf("template %s title is required", metadata.Name)
	}
	if strings.TrimSpace(metadata.Description) == "" {
		return fmt.Errorf("template %s description is required", metadata.Name)
	}
	if strings.TrimSpace(metadata.Category) == "" {
		return fmt.Errorf("template %s category is required", metadata.Name)
	}
	if strings.TrimSpace(metadata.Version) == "" {
		return fmt.Errorf("template %s version is required", metadata.Name)
	}
	if len(metadata.Files) == 0 {
		return fmt.Errorf("template %s must list at least one file", metadata.Name)
	}
	seen := make(map[string]struct{}, len(metadata.Files))
	for _, file := range metadata.Files {
		clean, err := cleanTemplateFile(file)
		if err != nil {
			return fmt.Errorf("template %s file %q is invalid: %w", metadata.Name, file, err)
		}
		if _, ok := seen[clean]; ok {
			return fmt.Errorf("template %s file %q is listed more than once", metadata.Name, clean)
		}
		seen[clean] = struct{}{}
	}
	return nil
}

func isValidTemplateName(name string) bool {
	if !templateNamePattern.MatchString(name) {
		return false
	}
	return !strings.Contains(name, "..")
}

func cleanTemplateFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", fmt.Errorf("path must stay inside the template")
	}
	return clean, nil
}
