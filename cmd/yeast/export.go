package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/Twarga/yeast/internal/app"
)

func newExportCmd(service *app.Service) *cobra.Command {
	var (
		format   string
		output   string
		sysprep  bool
		skelUser string
		builder  string
	)
	cmd := &cobra.Command{
		Use:   "export <instance> --format <qcow2|rootfs|iso> -o <path>",
		Short: "Export a VM to a standalone qcow2, rootfs .tar.zst, or bootable live .iso",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Export(context.Background(), app.ExportOptions{
				GuestTargetOptions: app.GuestTargetOptions{Target: args[0]},
				Format:             format,
				Output:             output,
				Sysprep:            sysprep,
				SkelUser:           skelUser,
				Builder:            builder,
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "export", result)
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "Export format: qcow2, rootfs, or iso (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output path (required)")
	cmd.Flags().BoolVar(&sysprep, "sysprep", false, "Generalize a throwaway clone first (machine-id, ssh host keys, caches, fstrim)")
	cmd.Flags().StringVar(&skelUser, "skel-user", "", "iso: copy this user's dotfiles into /etc/skel")
	cmd.Flags().StringVar(&builder, "builder", "", "iso builder (default: debian-live)")
	_ = cmd.MarkFlagRequired("format")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func newBootISOCmd(service *app.Service) *cobra.Command {
	var (
		uefi     bool
		headless bool
		diskSize string
		wait     string
		shot     string
		remove   bool
	)
	cmd := &cobra.Command{
		Use:   "boot-iso <path.iso>",
		Short: "Boot a local .iso in a clean Yeast VM (boot-test OS builds)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			waitDur, _ := time.ParseDuration(wait)
			result, err := service.BootISO(context.Background(), app.BootISOOptions{
				ISOPath:    args[0],
				UEFI:       uefi,
				Headless:   headless,
				DiskSize:   diskSize,
				Wait:       waitDur,
				Screenshot: shot,
				Remove:     remove,
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "boot-iso", result)
		},
	}
	cmd.Flags().BoolVar(&uefi, "uefi", false, "Boot with UEFI firmware (OVMF)")
	cmd.Flags().BoolVar(&headless, "headless", false, "No VNC display")
	cmd.Flags().StringVar(&diskSize, "disk", "", "Create an empty scratch disk (e.g. 20G) as install target")
	cmd.Flags().StringVar(&wait, "wait", "", "Wait before screenshot/return (e.g. 30s)")
	cmd.Flags().StringVar(&shot, "screenshot", "", "Save a QMP framebuffer screenshot to this PNG path")
	cmd.Flags().BoolVar(&remove, "rm", false, "Destroy the VM when the command finishes")
	return cmd
}
