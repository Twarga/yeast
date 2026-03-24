package images

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DownloadOptions struct {
	Retries  int
	Timeout  time.Duration
	Progress DownloadProgressSink
}

type retryableError struct {
	err error
}

type DownloadAttemptInfo struct {
	Attempt       int
	TotalAttempts int
	URL           string
	TotalBytes    int64
}

type DownloadProgressUpdate struct {
	Attempt         int
	TotalAttempts   int
	DownloadedBytes int64
	TotalBytes      int64
}

type DownloadRetryInfo struct {
	Attempt       int
	TotalAttempts int
	Err           error
	Wait          time.Duration
}

type DownloadProgressSink interface {
	AttemptStarted(info DownloadAttemptInfo)
	BytesTransferred(update DownloadProgressUpdate)
	RetryScheduled(info DownloadRetryInfo)
	AttemptFinished(info DownloadAttemptInfo)
}

func (e retryableError) Error() string {
	return e.err.Error()
}

func (e retryableError) Unwrap() error {
	return e.err
}

func markRetryable(err error) error {
	return retryableError{err: err}
}

func IsRetryable(err error) bool {
	var re retryableError
	return errors.As(err, &re)
}

func DefaultDownloadOptions() DownloadOptions {
	return DownloadOptions{
		Retries: 3,
		Timeout: 30 * time.Minute,
	}
}

func DownloadAndVerify(spec TrustedImage, destPath string, opts DownloadOptions) error {
	if spec.URL == "" || spec.SHA256 == "" {
		return fmt.Errorf("manifest entry %q is missing URL or SHA256", spec.Name)
	}
	if opts.Retries <= 0 {
		opts.Retries = 1
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Minute
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	expected := strings.ToLower(spec.SHA256)
	var lastErr error
	for attempt := 1; attempt <= opts.Retries; attempt++ {
		lastErr = downloadAttempt(spec.URL, expected, destPath, opts.Timeout, attempt, opts.Retries, opts.Progress)
		if lastErr == nil {
			return nil
		}
		if !IsRetryable(lastErr) || attempt == opts.Retries {
			return lastErr
		}
		wait := backoff(attempt)
		if opts.Progress != nil {
			opts.Progress.RetryScheduled(DownloadRetryInfo{
				Attempt:       attempt,
				TotalAttempts: opts.Retries,
				Err:           lastErr,
				Wait:          wait,
			})
		}
		time.Sleep(wait)
	}
	return lastErr
}

func VerifyFileSHA256(path string, expected string) error {
	sum, err := FileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(sum, expected) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", strings.ToLower(expected), strings.ToLower(sum))
	}
	return nil
}

func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read file for checksum: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func downloadAttempt(url, expectedSHA, destPath string, timeout time.Duration, attempt, totalAttempts int, progress DownloadProgressSink) error {
	tmpPath := fmt.Sprintf("%s.part-%d-%d", destPath, os.Getpid(), time.Now().UnixNano())
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return markRetryable(fmt.Errorf("download request failed: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		msg := fmt.Sprintf("unexpected HTTP status %d from %s: %s", resp.StatusCode, url, strings.TrimSpace(string(body)))
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			return markRetryable(errors.New(msg))
		}
		return errors.New(msg)
	}

	totalBytes := resp.ContentLength
	if totalBytes < 0 {
		totalBytes = 0
	}
	attemptInfo := DownloadAttemptInfo{
		Attempt:       attempt,
		TotalAttempts: totalAttempts,
		URL:           url,
		TotalBytes:    totalBytes,
	}
	if progress != nil {
		progress.AttemptStarted(attemptInfo)
	}

	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	h := sha256.New()
	reader := &progressReader{
		r:             resp.Body,
		progress:      progress,
		attempt:       attempt,
		totalAttempts: totalAttempts,
		totalBytes:    totalBytes,
		lastReport:    time.Now(),
	}
	if _, err := io.Copy(io.MultiWriter(out, h), reader); err != nil {
		_ = out.Close()
		return markRetryable(fmt.Errorf("failed while downloading image: %w", err))
	}
	reader.flush()
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	gotSHA := hex.EncodeToString(h.Sum(nil))
	if gotSHA != expectedSHA {
		return fmt.Errorf("checksum mismatch after download: expected %s, got %s", expectedSHA, gotSHA)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to move downloaded image into place: %w", err)
	}
	if progress != nil {
		progress.AttemptFinished(attemptInfo)
	}
	success = true
	return nil
}

type progressReader struct {
	r             io.Reader
	progress      DownloadProgressSink
	attempt       int
	totalAttempts int
	totalBytes    int64
	downloaded    int64
	lastReport    time.Time
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.downloaded += int64(n)
		now := time.Now()
		if now.Sub(r.lastReport) >= 120*time.Millisecond {
			r.flush()
			r.lastReport = now
		}
	}
	if err == io.EOF {
		r.flush()
	}
	return n, err
}

func (r *progressReader) flush() {
	if r.progress == nil {
		return
	}
	r.progress.BytesTransferred(DownloadProgressUpdate{
		Attempt:         r.attempt,
		TotalAttempts:   r.totalAttempts,
		DownloadedBytes: r.downloaded,
		TotalBytes:      r.totalBytes,
	})
}

func backoff(attempt int) time.Duration {
	if attempt <= 1 {
		return time.Second
	}
	// 2s, 4s, 8s capped to 10s
	d := time.Duration(1<<attempt) * time.Second
	if d > 10*time.Second {
		return 10 * time.Second
	}
	return d
}
