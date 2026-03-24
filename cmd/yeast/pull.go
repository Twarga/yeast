package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"yeast/pkg/images"

	"github.com/spf13/cobra"
)

var (
	pullRetries int
	pullTimeout time.Duration
	pullForce   bool
	pullList    bool
)

var pullCmd = &cobra.Command{
	Use:   "pull [image]",
	Short: "Download a trusted base image and verify checksum",
	Args: func(cmd *cobra.Command, args []string) error {
		if pullList {
			if len(args) != 0 {
				return fmt.Errorf("--list does not accept an image name")
			}
			return nil
		}
		if len(args) != 1 {
			return cobra.ExactArgs(1)(cmd, args)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if pullList {
			return runPullList()
		}

		imageName := args[0]
		spec, ok := images.GetTrustedImage(imageName)
		if !ok {
			return jsonCommandError("pull", "unsupported_image", fmt.Errorf("unsupported image %q; supported images: %v; run `yeast pull --list` for details", imageName, images.SupportedImageNames()))
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return jsonCommandError("pull", "resolve_home_failed", fmt.Errorf("failed to resolve home directory: %w", err))
		}

		cacheDir := filepath.Join(home, ".yeast", "cache")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return jsonCommandError("pull", "cache_dir_create_failed", fmt.Errorf("failed to create cache dir %s: %w", cacheDir, err))
		}

		destPath := filepath.Join(cacheDir, imageName+".img")
		data := pullCommandData{
			Schema:      "yeast.pull.v1",
			Image:       imageName,
			SourceURL:   spec.URL,
			SHA256:      spec.SHA256,
			Destination: destPath,
			Action:      "downloaded",
		}

		if _, err := os.Stat(destPath); err == nil {
			if !pullForce {
				if err := images.VerifyFileSHA256(destPath, spec.SHA256); err == nil {
					data.Action = "already_present"
					if outputJSON {
						return jsonCommandSuccess("pull", data)
					}
					fmt.Printf("Image %s already exists and checksum is valid: %s\n", imageName, destPath)
					return nil
				}
				return jsonCommandErrorWithData("pull", "checksum_mismatch", fmt.Errorf("image %s already exists but checksum does not match manifest; rerun with --force to replace it", destPath), data)
			}
			data.Action = "replaced"
			if !outputJSON {
				fmt.Printf("Replacing existing image at %s (--force)\n", destPath)
			}
		}

		if !outputJSON {
			fmt.Printf("Downloading trusted image %s\n", imageName)
			fmt.Printf("Source URL: %s\n", spec.URL)
			fmt.Printf("Expected SHA256: %s\n", spec.SHA256)
		}

		opts := images.DownloadOptions{
			Retries: pullRetries,
			Timeout: pullTimeout,
		}

		if err := images.DownloadAndVerify(spec, destPath, opts); err != nil {
			return jsonCommandErrorWithData("pull", "download_or_verify_failed", fmt.Errorf("pull failed: %w", err), data)
		}

		if outputJSON {
			return jsonCommandSuccess("pull", data)
		}

		fmt.Printf("Image saved and verified: %s\n", destPath)
		return nil
	},
}

func runPullList() error {
	names := images.SupportedImageNames()
	items := make([]pullListImage, 0, len(names))
	for _, name := range names {
		spec, ok := images.GetTrustedImage(name)
		if !ok {
			continue
		}
		items = append(items, pullListImage{
			Name:      spec.Name,
			SourceURL: spec.URL,
			SHA256:    spec.SHA256,
		})
	}

	data := pullListCommandData{
		Schema: "yeast.pull.list.v1",
		Count:  len(items),
		Images: items,
	}

	if outputJSON {
		return jsonCommandSuccess("pull", data)
	}

	fmt.Println("Supported trusted images:")
	for _, item := range items {
		fmt.Printf("- %s\n", item.Name)
		fmt.Printf("  URL: %s\n", item.SourceURL)
		fmt.Printf("  SHA256: %s\n", item.SHA256)
	}
	return nil
}

func init() {
	pullCmd.Flags().BoolVar(&pullList, "list", false, "List supported trusted images")
	pullCmd.Flags().IntVar(&pullRetries, "retries", 3, "Number of download retries for transient failures")
	pullCmd.Flags().DurationVar(&pullTimeout, "timeout", 30*time.Minute, "Per-attempt download timeout")
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "Replace existing local image even if checksum mismatch")
	rootCmd.AddCommand(pullCmd)
}
