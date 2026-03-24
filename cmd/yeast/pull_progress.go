package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"yeast/pkg/images"
)

type pullProgressPrinter struct {
	imageName   string
	startedAt   time.Time
	lastLogAt   time.Time
	lastWidth   int
	lastTotal   int64
	lastBytes   int64
	active      bool
	interactive bool
	mu          sync.Mutex
}

func newPullProgressPrinter(imageName string) *pullProgressPrinter {
	return &pullProgressPrinter{
		imageName:   imageName,
		interactive: humanTTY(),
	}
}

func (p *pullProgressPrinter) AttemptStarted(info images.DownloadAttemptInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.clearProgressLine()
	humanInfof("Downloading %s (attempt %d/%d)", humanAccent(p.imageName), info.Attempt, info.TotalAttempts)
	p.startedAt = time.Now()
	p.lastLogAt = time.Time{}
	p.lastTotal = info.TotalBytes
	p.lastBytes = 0
	p.active = true
}

func (p *pullProgressPrinter) BytesTransferred(update images.DownloadProgressUpdate) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastTotal = update.TotalBytes
	p.lastBytes = update.DownloadedBytes
	if !p.active {
		return
	}

	if p.interactive {
		line := p.renderProgressLine(update.DownloadedBytes, update.TotalBytes)
		padding := ""
		if p.lastWidth > len(line) {
			padding = strings.Repeat(" ", p.lastWidth-len(line))
		}
		fmt.Printf("\r%s%s", line, padding)
		p.lastWidth = len(line)
		return
	}

	now := time.Now()
	if p.lastLogAt.IsZero() || now.Sub(p.lastLogAt) >= 2*time.Second || update.TotalBytes > 0 && update.DownloadedBytes >= update.TotalBytes {
		humanInfof("%s", p.renderProgressLine(update.DownloadedBytes, update.TotalBytes))
		p.lastLogAt = now
	}
}

func (p *pullProgressPrinter) RetryScheduled(info images.DownloadRetryInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.finishProgressLine()
	humanWarnf("Attempt %d/%d failed: %v", info.Attempt, info.TotalAttempts, info.Err)
	humanInfof("Retrying in %s", info.Wait.Round(time.Second))
}

func (p *pullProgressPrinter) AttemptFinished(_ images.DownloadAttemptInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.finishProgressLine()
	p.active = false
}

func (p *pullProgressPrinter) FinishSuccess() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.finishProgressLine()
}

func (p *pullProgressPrinter) FinishWithError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.finishProgressLine()
	humanErrorf("Download failed: %v", err)
}

func (p *pullProgressPrinter) renderProgressLine(downloaded, total int64) string {
	elapsed := time.Since(p.startedAt).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	speed := int64(float64(downloaded) / elapsed)
	if total > 0 {
		percent := float64(downloaded) / float64(total)
		if percent > 1 {
			percent = 1
		}
		return fmt.Sprintf(
			"%s %s %5.1f%%  %s / %s  %s/s",
			humanStyle(progressBar(percent, 26), ansiBlue),
			humanMuted("download"),
			percent*100,
			formatBytes(downloaded),
			formatBytes(total),
			formatBytes(speed),
		)
	}
	return fmt.Sprintf(
		"%s %s  %s  %s/s",
		humanStyle(progressSpinnerGlyph(downloaded), ansiBlue),
		humanMuted("download"),
		formatBytes(downloaded),
		formatBytes(speed),
	)
}

func (p *pullProgressPrinter) clearProgressLine() {
	if !p.interactive || p.lastWidth == 0 {
		return
	}
	fmt.Printf("\r%s\r", strings.Repeat(" ", p.lastWidth))
	p.lastWidth = 0
}

func (p *pullProgressPrinter) finishProgressLine() {
	if !p.interactive || p.lastWidth == 0 {
		return
	}
	fmt.Print("\n")
	p.lastWidth = 0
}

func progressBar(percent float64, width int) string {
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", width-filled) + "]"
}

func progressSpinnerGlyph(downloaded int64) string {
	frames := []string{"[|]", "[/]", "[-]", "[\\]"}
	if len(frames) == 0 {
		return "[ ]"
	}
	return frames[(downloaded/131072)%int64(len(frames))]
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(n)
	unit := "B"
	for _, u := range units {
		value /= 1024
		unit = u
		if value < 1024 {
			break
		}
	}
	if unit == "B" {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}
