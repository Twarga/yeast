package provision

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"yeast/internal/config"
)

const FingerprintSchemaVersion = "v1"

type FingerprintInput struct {
	SchemaVersion string            `json:"schema_version"`
	User          string            `json:"user,omitempty"`
	Sudo          string            `json:"sudo,omitempty"`
	Packages      []string          `json:"packages,omitempty"`
	Files         []FingerprintFile `json:"files,omitempty"`
	Shell         []string          `json:"shell,omitempty"`
}

type FingerprintFile struct {
	SourceDigest string `json:"source_digest"`
	Destination  string `json:"destination"`
	Permissions  string `json:"permissions,omitempty"`
}

func Fingerprint(projectRoot string, instance config.Instance, cfg *config.Config) (string, error) {
	plan := BuildPlan(instance, cfg.Provision)
	resolved, err := resolveFingerprintPlan(projectRoot, plan)
	if err != nil {
		return "", err
	}

	input := FingerprintInput{
		SchemaVersion: FingerprintSchemaVersion,
		User:          instance.User,
		Sudo:          instance.Sudo,
	}

	pkgs := make([]string, len(resolved.Packages))
	for i, p := range resolved.Packages {
		pkgs[i] = p.Name
	}
	sort.Strings(pkgs)
	input.Packages = pkgs

	input.Files = make([]FingerprintFile, len(resolved.Files))
	for i, f := range resolved.Files {
		digest, err := fileDigest(f.Source)
		if err != nil {
			return "", err
		}
		input.Files[i] = FingerprintFile{
			SourceDigest: digest,
			Destination:  f.Destination,
			Permissions:  f.Permissions,
		}
	}

	input.Shell = make([]string, len(resolved.Shell))
	for i, s := range resolved.Shell {
		input.Shell[i] = s.Command
	}

	canonical, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}

func resolveFingerprintPlan(projectRoot string, plan Plan) (Plan, error) {
	if plan.Empty() {
		return plan, nil
	}

	resolved := plan
	resolved.Files = make([]FileStep, 0, len(plan.Files))
	for _, step := range plan.Files {
		source := step.Source
		if !filepath.IsAbs(source) {
			source = filepath.Join(projectRoot, source)
		}
		source = filepath.Clean(source)
		if _, err := os.Stat(source); err != nil {
			return Plan{}, err
		}
		resolved.Files = append(resolved.Files, FileStep{
			Source:      source,
			Destination: step.Destination,
			Permissions: step.Permissions,
			Origin:      step.Origin,
		})
	}

	return resolved, nil
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
