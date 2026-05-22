package cloudinit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNoISOBuilder = errors.New("no supported ISO builder found")

type SeedInput struct {
	InstanceName  string
	RuntimeDir    string
	UserData      string
	MetaData      string
	NetworkConfig string
}

type SeedResult struct {
	UserDataPath      string
	MetaDataPath      string
	NetworkConfigPath string
	ISOPath           string
	Builder           string
}

type isoBuilder struct {
	Name      string
	BuildArgs func(outputPath, userDataPath, metaDataPath, networkConfigPath string) []string
}

var (
	lookPath      = exec.LookPath
	runISOCommand = runISOCommandContext
)

func CreateSeedISO(ctx context.Context, input SeedInput) (SeedResult, error) {
	if strings.TrimSpace(input.InstanceName) == "" {
		return SeedResult{}, fmt.Errorf("instance name is required")
	}
	if strings.TrimSpace(input.RuntimeDir) == "" {
		return SeedResult{}, fmt.Errorf("runtime directory is required")
	}
	if strings.TrimSpace(input.UserData) == "" {
		return SeedResult{}, fmt.Errorf("user-data is required")
	}
	if strings.TrimSpace(input.MetaData) == "" {
		return SeedResult{}, fmt.Errorf("meta-data is required")
	}

	if err := os.MkdirAll(input.RuntimeDir, 0755); err != nil {
		return SeedResult{}, fmt.Errorf("create runtime directory %s: %w", input.RuntimeDir, err)
	}

	userDataPath := filepath.Join(input.RuntimeDir, "user-data")
	metaDataPath := filepath.Join(input.RuntimeDir, "meta-data")
	networkConfigPath := filepath.Join(input.RuntimeDir, "network-config")
	isoPath := filepath.Join(input.RuntimeDir, "seed.iso")

	if err := writeFileAtomic(userDataPath, []byte(ensureTrailingNewline(input.UserData))); err != nil {
		return SeedResult{}, fmt.Errorf("write user-data: %w", err)
	}
	if err := writeFileAtomic(metaDataPath, []byte(ensureTrailingNewline(input.MetaData))); err != nil {
		return SeedResult{}, fmt.Errorf("write meta-data: %w", err)
	}
	if strings.TrimSpace(input.NetworkConfig) != "" {
		if err := writeFileAtomic(networkConfigPath, []byte(ensureTrailingNewline(input.NetworkConfig))); err != nil {
			return SeedResult{}, fmt.Errorf("write network-config: %w", err)
		}
	} else {
		_ = os.Remove(networkConfigPath)
	}

	builder, err := discoverISOBuilder()
	if err != nil {
		return SeedResult{
			UserDataPath:      userDataPath,
			MetaDataPath:      metaDataPath,
			NetworkConfigPath: networkConfigPath,
			ISOPath:           isoPath,
		}, err
	}

	args := builder.BuildArgs(isoPath, userDataPath, metaDataPath, networkConfigPath)
	if err := runISOCommand(ctx, builder.Name, args...); err != nil {
		return SeedResult{}, fmt.Errorf("create seed ISO with %s: %w", builder.Name, err)
	}

	return SeedResult{
		UserDataPath:      userDataPath,
		MetaDataPath:      metaDataPath,
		NetworkConfigPath: networkConfigPath,
		ISOPath:           isoPath,
		Builder:           builder.Name,
	}, nil
}

func discoverISOBuilder() (isoBuilder, error) {
	builders := []isoBuilder{
		{
			Name: "genisoimage",
			BuildArgs: func(outputPath, userDataPath, metaDataPath, networkConfigPath string) []string {
				args := []string{
					"-output", outputPath,
					"-volid", "cidata",
					"-joliet",
					"-rock",
					userDataPath,
					metaDataPath,
				}
				if strings.TrimSpace(networkConfigPath) != "" {
					if _, err := os.Stat(networkConfigPath); err == nil {
						args = append(args, networkConfigPath)
					}
				}
				return args
			},
		},
		{
			Name: "mkisofs",
			BuildArgs: func(outputPath, userDataPath, metaDataPath, networkConfigPath string) []string {
				args := []string{
					"-output", outputPath,
					"-volid", "cidata",
					"-joliet",
					"-rock",
					userDataPath,
					metaDataPath,
				}
				if strings.TrimSpace(networkConfigPath) != "" {
					if _, err := os.Stat(networkConfigPath); err == nil {
						args = append(args, networkConfigPath)
					}
				}
				return args
			},
		},
	}

	for _, builder := range builders {
		if _, err := lookPath(builder.Name); err == nil {
			return builder, nil
		}
	}

	return isoBuilder{}, fmt.Errorf("%w: install genisoimage or mkisofs", ErrNoISOBuilder)
}

func ensureTrailingNewline(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}

func runISOCommandContext(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, trimmed)
	}
	return nil
}

func writeFileAtomic(path string, content []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save file %s: %w", path, err)
	}
	return nil
}
