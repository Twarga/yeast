package main

import (
	"fmt"
	"os"
	"yeast/pkg/config"

	"github.com/spf13/cobra"
)

var (
	initName     string
	initImage    string
	initMemory   int
	initCPUs     int
	initDiskSize string
	initUser     string
	initSudo     string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Yeast project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat("yeast.yaml"); err == nil {
			return jsonCommandErrorWithData("init", "config_exists", fmt.Errorf("yeast.yaml already exists"), initCommandData{
				Schema:     "yeast.init.v1",
				ConfigPath: "yeast.yaml",
				Created:    false,
			})
		}

		content := renderInitConfig()
		tmpPath := "yeast.yaml.tmp"
		if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
			return jsonCommandErrorWithData("init", "config_write_failed", fmt.Errorf("error writing file: %w", err), initCommandData{
				Schema:     "yeast.init.v1",
				ConfigPath: "yeast.yaml",
				Created:    false,
			})
		}

		cfg, err := config.Load(tmpPath)
		if err != nil {
			_ = os.Remove(tmpPath)
			return jsonCommandErrorWithData("init", "config_invalid", fmt.Errorf("invalid init options: %w", err), initCommandData{
				Schema:     "yeast.init.v1",
				ConfigPath: "yeast.yaml",
				Created:    false,
			})
		}
		if err := os.WriteFile(tmpPath, []byte(renderInitInstanceConfig(cfg.Instances[0])), 0644); err != nil {
			_ = os.Remove(tmpPath)
			return jsonCommandErrorWithData("init", "config_write_failed", fmt.Errorf("error normalizing file: %w", err), initCommandData{
				Schema:     "yeast.init.v1",
				ConfigPath: "yeast.yaml",
				Created:    false,
			})
		}

		if err := os.Rename(tmpPath, "yeast.yaml"); err != nil {
			_ = os.Remove(tmpPath)
			return jsonCommandErrorWithData("init", "config_write_failed", fmt.Errorf("error moving file into place: %w", err), initCommandData{
				Schema:     "yeast.init.v1",
				ConfigPath: "yeast.yaml",
				Created:    false,
			})
		}

		if outputJSON {
			return jsonCommandSuccess("init", initCommandData{
				Schema:     "yeast.init.v1",
				ConfigPath: "yeast.yaml",
				Created:    true,
			})
		}
		humanSection("Project Initialized")
		humanSuccessf("Created %s", humanAccent("yeast.yaml"))
		humanKeyValue("Instance", humanAccent(initName))
		humanKeyValue("Image", initImage)
		humanKeyValue("Memory", fmt.Sprintf("%d MB", initMemory))
		humanKeyValue("CPUs", fmt.Sprintf("%d", initCPUs))
		if initDiskSize != "" {
			humanKeyValue("Disk", initDiskSize)
		}
		humanKeyValue("User", initUser)
		humanKeyValue("Sudo", initSudo)
		fmt.Println()
		humanInfof("Next: run %s", humanAccent(fmt.Sprintf("yeast pull %s", initImage)))
		humanInfof("Then: run %s", humanAccent("yeast up"))
		return nil
	},
}

func renderInitConfig() string {
	return renderInitInstanceConfig(config.Instance{
		Name:     initName,
		Image:    initImage,
		Memory:   initMemory,
		CPUs:     initCPUs,
		DiskSize: initDiskSize,
		User:     initUser,
		Sudo:     initSudo,
	})
}

func renderInitInstanceConfig(instance config.Instance) string {
	diskLine := ""
	if instance.DiskSize != "" {
		diskLine = fmt.Sprintf("    disk_size: %s\n", instance.DiskSize)
	}

	return fmt.Sprintf("version: 1\n"+
		"instances:\n"+
		"  - name: %s\n"+
		"    image: %s\n"+
		"    memory: %d\n"+
		"    cpus: %d\n"+
		"%s"+
		"    user: %s\n"+
		"    sudo: %s\n",
		instance.Name,
		instance.Image,
		instance.Memory,
		instance.CPUs,
		diskLine,
		instance.User,
		instance.Sudo,
	)
}

func init() {
	initCmd.Flags().StringVar(&initName, "name", "web", "Initial instance name")
	initCmd.Flags().StringVar(&initImage, "image", "ubuntu-22.04", "Initial instance image key")
	initCmd.Flags().IntVar(&initMemory, "memory", 1024, "Initial instance memory in MB")
	initCmd.Flags().IntVar(&initCPUs, "cpus", 1, "Initial instance CPU count")
	initCmd.Flags().StringVar(&initDiskSize, "disk-size", "", "Initial instance disk size (examples: 20G, 10240M)")
	initCmd.Flags().StringVar(&initUser, "user", "yeast", "Initial bootstrap username")
	initCmd.Flags().StringVar(&initSudo, "sudo", "none", "Initial sudo policy: none | password | nopasswd")
	rootCmd.AddCommand(initCmd)
}
