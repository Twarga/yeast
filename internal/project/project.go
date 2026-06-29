package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const MetadataSchema = "yeast.project.v1"
const MetadataDirName = ".yeast"
const MetadataFileName = "project.json"

var ErrMetadataNotFound = errors.New("project metadata not found")

type LocalProjectCleanupResult struct {
	Removed []string
}

type Metadata struct {
	Schema    string    `json:"schema"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func NewMetadata(id string, createdAt time.Time) Metadata {
	return Metadata{
		Schema:    MetadataSchema,
		ID:        id,
		CreatedAt: createdAt.UTC(),
	}
}

func MetadataPath(root string) string {
	return filepath.Join(root, MetadataDirName, MetadataFileName)
}

func EnsureMetadata(root string, now time.Time) (Metadata, error) {
	metadata, err := LoadMetadata(root)
	if err == nil {
		return metadata, nil
	}
	if !errors.Is(err, ErrMetadataNotFound) {
		return Metadata{}, err
	}

	id, err := GenerateID()
	if err != nil {
		return Metadata{}, err
	}

	metadata = NewMetadata(id, now)
	if err := SaveMetadata(root, metadata); err != nil {
		return Metadata{}, err
	}
	return metadata, nil
}

func LoadMetadata(root string) (Metadata, error) {
	path := MetadataPath(root)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Metadata{}, fmt.Errorf("%w: %s", ErrMetadataNotFound, path)
		}
		return Metadata{}, fmt.Errorf("read project metadata %s: %w", path, err)
	}

	var metadata Metadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return Metadata{}, fmt.Errorf("parse project metadata %s: %w", path, err)
	}
	if err := ValidateMetadata(metadata); err != nil {
		return Metadata{}, fmt.Errorf("invalid project metadata %s: %w", path, err)
	}
	return metadata, nil
}

func SaveMetadata(root string, metadata Metadata) error {
	if err := ValidateMetadata(metadata); err != nil {
		return fmt.Errorf("invalid project metadata: %w", err)
	}

	dir := filepath.Join(root, MetadataDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create project metadata directory %s: %w", dir, err)
	}

	path := MetadataPath(root)
	tmpPath := path + ".tmp"
	raw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode project metadata: %w", err)
	}
	raw = append(raw, '\n')

	if err := os.WriteFile(tmpPath, raw, 0644); err != nil {
		return fmt.Errorf("write project metadata temp file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save project metadata %s: %w", path, err)
	}
	return nil
}

func ValidateMetadata(metadata Metadata) error {
	if metadata.Schema != MetadataSchema {
		return fmt.Errorf("unsupported schema %q, expected %q", metadata.Schema, MetadataSchema)
	}
	if !IsValidID(metadata.ID) {
		return fmt.Errorf("invalid project id %q", metadata.ID)
	}
	if metadata.CreatedAt.IsZero() {
		return errors.New("created_at is required")
	}
	return nil
}

func RemoveLocalProjectFiles(root string) (LocalProjectCleanupResult, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return LocalProjectCleanupResult{}, fmt.Errorf("resolve project root: %w", err)
	}

	targets := []string{
		filepath.Join(cleanRoot, MetadataDirName),
		filepath.Join(cleanRoot, "yeast.yaml"),
	}
	result := LocalProjectCleanupResult{
		Removed: make([]string, 0, len(targets)),
	}

	for _, target := range targets {
		cleanTarget, err := filepath.Abs(target)
		if err != nil {
			return result, fmt.Errorf("resolve local project file %s: %w", target, err)
		}
		if cleanTarget != filepath.Join(cleanRoot, "yeast.yaml") && cleanTarget != filepath.Join(cleanRoot, MetadataDirName) {
			return result, fmt.Errorf("refusing to remove unexpected local project path: %s", cleanTarget)
		}
		if _, err := os.Stat(cleanTarget); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return result, fmt.Errorf("stat local project file %s: %w", cleanTarget, err)
		}
		if err := os.RemoveAll(cleanTarget); err != nil {
			return result, fmt.Errorf("remove local project file %s: %w", cleanTarget, err)
		}
		result.Removed = append(result.Removed, cleanTarget)
	}

	return result, nil
}
