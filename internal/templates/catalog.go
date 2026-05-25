package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed builtin/*/template.yaml builtin/*/yeast.yaml builtin/*/README.md builtin/*/site/*
var builtinFS embed.FS

type SourceType string

const (
	SourceBuiltin SourceType = "builtin"
	SourceLocal   SourceType = "local"
)

type Template struct {
	Metadata Metadata   `json:"Metadata"`
	Source   SourceType `json:"Source"`
	Path     string     `json:"Path,omitempty"`
}

func Builtins() ([]Template, error) {
	entries, err := fs.ReadDir(builtinFS, "builtin")
	if err != nil {
		return nil, fmt.Errorf("read built-in templates: %w", err)
	}

	templates := make([]Template, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		template, err := readBuiltin(entry.Name())
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}
	sortTemplates(templates)
	return templates, nil
}

func BuiltinNames() ([]string, error) {
	templates, err := Builtins()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(templates))
	for _, template := range templates {
		names = append(names, template.Metadata.Name)
	}
	return names, nil
}

func LookupBuiltin(name string) (Template, bool, error) {
	templates, err := Builtins()
	if err != nil {
		return Template{}, false, err
	}
	for _, template := range templates {
		if template.Metadata.Name == name {
			return template, true, nil
		}
	}
	return Template{}, false, nil
}

func readBuiltin(name string) (Template, error) {
	file, err := builtinFS.Open("builtin/" + name + "/" + MetadataFileName)
	if err != nil {
		return Template{}, fmt.Errorf("read built-in template %q metadata: %w", name, err)
	}
	defer file.Close()

	metadata, err := DecodeMetadata(file)
	if err != nil {
		return Template{}, fmt.Errorf("load built-in template %q metadata: %w", name, err)
	}
	if metadata.Name != name {
		return Template{}, fmt.Errorf("built-in template directory %q contains metadata for %q", name, metadata.Name)
	}
	return Template{
		Metadata: metadata,
		Source:   SourceBuiltin,
		Path:     "builtin/" + name,
	}, nil
}

func sortTemplates(templates []Template) {
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Metadata.Name < templates[j].Metadata.Name
	})
}
