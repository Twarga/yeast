package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type instanceBootProgress struct {
	name        string
	startedAt   time.Time
	interactive bool
	lastWidth   int
	done        chan struct{}
	once        sync.Once
}

func newInstanceBootProgress(name string) *instanceBootProgress {
	return &instanceBootProgress{
		name:        name,
		interactive: humanTTY(),
		done:        make(chan struct{}),
	}
}

func (p *instanceBootProgress) Start() {
	p.startedAt = time.Now()
	if !p.interactive {
		return
	}

	go func() {
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()

		frame := 0
		for {
			select {
			case <-ticker.C:
				line := fmt.Sprintf(
					"%s %s %s  %s  %s",
					humanStyle(progressFrames[frame%len(progressFrames)], ansiBlue),
					humanMuted("booting"),
					humanAccent(p.name),
					humanMuted(formatHumanDuration(time.Since(p.startedAt))),
					humanMuted("waiting for SSH readiness"),
				)
				padding := ""
				if p.lastWidth > len(line) {
					padding = strings.Repeat(" ", p.lastWidth-len(line))
				}
				fmt.Printf("\r%s%s", line, padding)
				p.lastWidth = len(line)
				frame++
			case <-p.done:
				p.clearLine()
				return
			}
		}
	}()
}

func (p *instanceBootProgress) Finish() time.Duration {
	elapsed := time.Since(p.startedAt)
	p.once.Do(func() {
		close(p.done)
	})
	return elapsed
}

func (p *instanceBootProgress) clearLine() {
	if !p.interactive || p.lastWidth == 0 {
		return
	}
	fmt.Printf("\r%s\r", strings.Repeat(" ", p.lastWidth))
	p.lastWidth = 0
}

var progressFrames = []string{"[|]", "[/]", "[-]", "[\\]"}
