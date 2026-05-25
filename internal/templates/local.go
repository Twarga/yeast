package templates

import (
	"fmt"
	"os"
	"path/filepath"
)

func LoadLocal(dir string) (Template, error) {
	if dir == "" {
		return Template{}, fmt.Errorf("template directory is required")
	}
	cleanDir := filepath.Clean(dir)
	info, err := os.Stat(cleanDir)
	if err != nil {
		return Template{}, fmt.Errorf("inspect template directory %s: %w", cleanDir, err)
	}
	if !info.IsDir() {
		return Template{}, fmt.Errorf("template path %s is not a directory", cleanDir)
	}

	metadataPath := filepath.Join(cleanDir, MetadataFileName)
	file, err := os.Open(metadataPath)
	if err != nil {
		return Template{}, fmt.Errorf("read template metadata %s: %w", metadataPath, err)
	}
	defer file.Close()

	metadata, err := DecodeMetadata(file)
	if err != nil {
		return Template{}, fmt.Errorf("load template metadata %s: %w", metadataPath, err)
	}
	return Template{
		Metadata: metadata,
		Source:   SourceLocal,
		Path:     cleanDir,
	}, nil
}
