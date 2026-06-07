package images

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type DownloadOptions struct {
	Timeout  time.Duration
	Client   *http.Client
	Progress DownloadProgressSink
}

type DownloadProgressSink func(DownloadProgress)

type DownloadProgress struct {
	Stage      string
	Downloaded int64
	Total      int64
}

func Download(image TrustedImage, destination string, options DownloadOptions) error {
	if image.URL == "" {
		return fmt.Errorf("image URL is required")
	}
	if image.SHA256 == "" {
		return fmt.Errorf("image SHA256 is required")
	}
	if destination == "" {
		return fmt.Errorf("destination path is required")
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	client := options.Client
	if client == nil {
		client = http.DefaultClient
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("create image cache directory for %s: %w", destination, err)
	}

	tempPath := destination + ".part"
	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale temp file %s: %w", tempPath, err)
	}

	success := false
	defer func() {
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, image.URL, nil)
	if err != nil {
		return fmt.Errorf("build download request for %s: %w", image.URL, err)
	}

	emitDownloadProgress(options.Progress, DownloadProgress{Stage: "requesting"})
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download image %s: %w", image.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download image %s: unexpected HTTP status %d", image.URL, resp.StatusCode)
	}

	total := resp.ContentLength
	emitDownloadProgress(options.Progress, DownloadProgress{Stage: "downloading", Total: total})
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create temp image file %s: %w", tempPath, err)
	}

	if _, err := io.Copy(file, &progressReader{
		reader:   resp.Body,
		total:    total,
		progress: options.Progress,
	}); err != nil {
		_ = file.Close()
		return fmt.Errorf("write temp image file %s: %w", tempPath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temp image file %s: %w", tempPath, err)
	}

	emitDownloadProgress(options.Progress, DownloadProgress{Stage: "verifying", Downloaded: total, Total: total})
	if err := VerifySHA256(tempPath, image.SHA256); err != nil {
		return err
	}

	if err := os.Rename(tempPath, destination); err != nil {
		return fmt.Errorf("move verified image into place %s: %w", destination, err)
	}

	emitDownloadProgress(options.Progress, DownloadProgress{Stage: "complete", Downloaded: total, Total: total})
	success = true
	return nil
}

type progressReader struct {
	reader     io.Reader
	downloaded int64
	total      int64
	progress   DownloadProgressSink
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.downloaded += int64(n)
		emitDownloadProgress(r.progress, DownloadProgress{
			Stage:      "downloading",
			Downloaded: r.downloaded,
			Total:      r.total,
		})
	}
	return n, err
}

func emitDownloadProgress(sink DownloadProgressSink, progress DownloadProgress) {
	if sink != nil {
		sink(progress)
	}
}
