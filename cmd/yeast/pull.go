package main

import (
	"errors"
	"fmt"
	"io"
	"time"
	"yeast/internal/app"
	"yeast/internal/images"

	"github.com/spf13/cobra"
)

func newPullCmd(service *app.Service) *cobra.Command {
	var list bool

	cmd := &cobra.Command{
		Use:   "pull [image]",
		Short: "List or download trusted base images",
		Args: func(cmd *cobra.Command, args []string) error {
			if list {
				if len(args) != 0 {
					return fmt.Errorf("--list does not accept an image name")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("expected exactly one image name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options := app.PullOptions{List: list}
			if !list {
				options.ImageName = args[0]
				if !outputJSON {
					options.Progress = humanDownloadProgress(cmd.ErrOrStderr())
				}
			}

			result, err := service.Pull(options)
			if err != nil {
				if errors.Is(err, app.ErrUnsupportedImage) {
					return app.WrapError(app.ErrorCodeInvalidArgument, fmt.Sprintf("%v. use `yeast pull --list` to see supported images", err), err)
				}
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "pull", result)
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List supported images")
	return cmd
}

func humanDownloadProgress(w io.Writer) images.DownloadProgressSink {
	started := time.Now()
	lastStage := ""
	lastPrint := time.Time{}
	return func(progress images.DownloadProgress) {
		now := time.Now()
		if progress.Stage == "downloading" && progress.Downloaded > 0 && !lastPrint.IsZero() && now.Sub(lastPrint) < time.Second {
			return
		}
		if progress.Stage != lastStage || progress.Stage != "downloading" || progress.Downloaded > 0 {
			lastStage = progress.Stage
			lastPrint = now
			_, _ = fmt.Fprintf(w, "yeast pull: %s %s\n", progress.Stage, formatDownloadProgress(progress, now.Sub(started)))
		}
	}
}

func formatDownloadProgress(progress images.DownloadProgress, elapsed time.Duration) string {
	elapsedText := fmt.Sprintf("(%s)", elapsed.Round(time.Second))
	if progress.Total > 0 && progress.Downloaded > 0 {
		percent := float64(progress.Downloaded) / float64(progress.Total) * 100
		return fmt.Sprintf("%s/%s %.0f%% %s", formatBytes(progress.Downloaded), formatBytes(progress.Total), percent, elapsedText)
	}
	if progress.Downloaded > 0 {
		return fmt.Sprintf("%s %s", formatBytes(progress.Downloaded), elapsedText)
	}
	return elapsedText
}

func formatBytes(value int64) string {
	if value < 0 {
		return "-"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	size := float64(value)
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d%s", value, units[unit])
	}
	return fmt.Sprintf("%.1f%s", size, units[unit])
}
