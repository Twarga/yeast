package main

import (
	"fmt"
	"os"
	"yeast/pkg/config"

	"github.com/spf13/cobra"
)

var (
	initName   string
	initImage  string
	initMemory int
	initCPUs   int
	initUser   string
	initSudo   string
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
		fmt.Printf("Created yeast.yaml for instance %s (%s, %dMB RAM, %d CPU).\n", initName, initImage, initMemory, initCPUs)
		fmt.Printf("Next steps: run 'yeast pull %s' and then 'yeast up'.\n", initImage)
		return nil
	},
}

func renderInitConfig() string {
	return renderInitInstanceConfig(config.Instance{
		Name:   initName,
		Image:  initImage,
		Memory: initMemory,
		CPUs:   initCPUs,
		User:   initUser,
		Sudo:   initSudo,
	})
}

func renderInitInstanceConfig(instance config.Instance) string {
	return fmt.Sprintf("version: 1\n"+
		"instances:\n"+
		"  - name: %s\n"+
		"    image: %s\n"+
		"    memory: %d\n"+
		"    cpus: %d\n"+
		"    user: %s\n"+
		"    sudo: %s\n",
		instance.Name,
		instance.Image,
		instance.Memory,
		instance.CPUs,
		instance.User,
		instance.Sudo,
	)
}

func init() {
	initCmd.Flags().StringVar(&initName, "name", "web", "Initial instance name")
	initCmd.Flags().StringVar(&initImage, "image", "ubuntu-22.04", "Initial instance image key")
	initCmd.Flags().IntVar(&initMemory, "memory", 1024, "Initial instance memory in MB")
	initCmd.Flags().IntVar(&initCPUs, "cpus", 1, "Initial instance CPU count")
	initCmd.Flags().StringVar(&initUser, "user", "yeast", "Initial bootstrap username")
	initCmd.Flags().StringVar(&initSudo, "sudo", "none", "Initial sudo policy: none | password | nopasswd")
	rootCmd.AddCommand(initCmd)
}
