package templates

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ErrOutputExists = errors.New("template output already exists")

type MaterializeOptions struct {
	Destination string
}

type MaterializeResult struct {
	Template    Metadata `json:"Template"`
	Destination string   `json:"Destination"`
	Files       []string `json:"Files"`
}

func Materialize(template Template, options MaterializeOptions) (MaterializeResult, error) {
	if err := ValidateMetadata(template.Metadata); err != nil {
		return MaterializeResult{}, err
	}
	destination := options.Destination
	if destination == "" {
		destination = "."
	}
	absoluteDestination, err := filepath.Abs(destination)
	if err != nil {
		return MaterializeResult{}, fmt.Errorf("resolve destination: %w", err)
	}
	info, err := os.Stat(absoluteDestination)
	if err != nil {
		return MaterializeResult{}, fmt.Errorf("inspect destination %s: %w", absoluteDestination, err)
	}
	if !info.IsDir() {
		return MaterializeResult{}, fmt.Errorf("destination %s is not a directory", absoluteDestination)
	}

	result := MaterializeResult{
		Template:    template.Metadata,
		Destination: absoluteDestination,
		Files:       make([]string, 0, len(template.Metadata.Files)),
	}
	for _, file := range template.Metadata.Files {
		cleanFile, err := cleanTemplateFile(file)
		if err != nil {
			return result, fmt.Errorf("prepare template file %q: %w", file, err)
		}
		content, err := readTemplateFile(template, cleanFile)
		if err != nil {
			return result, err
		}
		if err := writeMaterializedFile(absoluteDestination, cleanFile, content); err != nil {
			return result, err
		}
		result.Files = append(result.Files, cleanFile)
	}
	return result, nil
}

func readTemplateFile(template Template, file string) ([]byte, error) {
	switch template.Source {
	case SourceBuiltin:
		path := filepath.ToSlash(filepath.Join("builtin", template.Metadata.Name, file))
		content, err := fs.ReadFile(builtinFS, path)
		if err != nil {
			return nil, fmt.Errorf("read built-in template file %s: %w", path, err)
		}
		return content, nil
	case SourceLocal:
		path := filepath.Join(template.Path, filepath.FromSlash(file))
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read local template file %s: %w", path, err)
		}
		return content, nil
	default:
		return nil, fmt.Errorf("unsupported template source %q", template.Source)
	}
}

func writeMaterializedFile(destination, relativePath string, content []byte) error {
	target := filepath.Join(destination, filepath.FromSlash(relativePath))
	if err := ensureInside(destination, target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("create template file directory %s: %w", filepath.Dir(target), err)
	}

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s", ErrOutputExists, target)
		}
		return fmt.Errorf("create template output file %s: %w", target, err)
	}
	defer file.Close()

	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("write template output file %s: %w", target, err)
	}
	return nil
}

func ensureInside(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return fmt.Errorf("check template output path %s: %w", target, err)
	}
	normalized := filepath.ToSlash(relative)
	if normalized == ".." || strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("template output path %s escapes destination %s", target, root)
	}
	return nil
}
