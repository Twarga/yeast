package ui

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Twarga/yeast/internal/ui/sprites"
)

func TestLiveStageStaticNonTTY(t *testing.T) {
	var buf bytes.Buffer
	s := NewLiveStage(&buf, LiveOptions{})
	if s.Live() {
		t.Fatal("expected non-live stage for non-TTY writer")
	}
	s.Start(sprites.MoodWorking)
	s.SetStep("image", "Pull image", StepRunning, "ubuntu-22.04")
	s.SetStep("image", "Pull image", StepDone, "")
	s.Finish(sprites.MoodSuccess, "all done")

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("static output must contain zero ANSI escapes, got %q", out)
	}
	if !strings.Contains(out, "Pull image") || !strings.Contains(out, "all done") {
		t.Fatalf("static output missing steps/summary: %q", out)
	}
}

func TestLiveStageNarrowDegrades(t *testing.T) {
	var buf bytes.Buffer
	s := NewLiveStage(&buf, LiveOptions{Force: true, Width: 60, Depth: sprites.Depth16})
	if s.Live() {
		t.Fatal("width < 80 must degrade to static output")
	}
	s.Start(sprites.MoodWorking)
	s.SetStep("disk", "Create disk", StepDone, "")
	s.Finish(sprites.MoodSuccess, "ok")
	if strings.Contains(buf.String(), "\x1b[?25") || strings.Contains(buf.String(), "\x1b[1A") {
		t.Fatalf("narrow output must not contain cursor control: %q", buf.String())
	}
}

func TestLiveStageAnimatedFrame(t *testing.T) {
	var buf bytes.Buffer
	s := NewLiveStage(&buf, LiveOptions{Force: true, Width: 100, Depth: sprites.Depth16})
	if !s.Live() {
		t.Fatal("expected live stage")
	}
	s.Start(sprites.MoodWorking)
	s.SetStep("image", "Pull image", StepRunning, "")
	s.SetStep("disk", "Create disk", StepPending, "")
	s.Finish(sprites.MoodSuccess, "done!")

	out := buf.String()
	if !strings.Contains(out, "\x1b[?25l") {
		t.Fatal("missing cursor-hide sequence")
	}
	if !strings.Contains(out, "\x1b[?25h") {
		t.Fatal("missing cursor-restore sequence")
	}
	if strings.Index(out, "\x1b[?25l") > strings.LastIndex(out, "\x1b[?25h") {
		t.Fatal("cursor must be hidden before it is restored")
	}
	if !strings.Contains(out, "Pull image") || !strings.Contains(out, "done!") {
		t.Fatalf("frame missing step labels/summary: %q", out)
	}
	if !strings.Contains(out, "▀") && !strings.Contains(out, "▄") {
		t.Fatal("live frame should contain half-block sprite art")
	}
}

func TestLiveStageInPlaceRedraw(t *testing.T) {
	var buf bytes.Buffer
	s := NewLiveStage(&buf, LiveOptions{Force: true, Width: 100, Depth: sprites.Depth16})
	s.Start(sprites.MoodWorking)
	s.SetStep("image", "Pull image", StepRunning, "")
	time.Sleep(2 * frameInterval) // let ≥1 tick fire
	s.Stop()

	out := buf.String()
	if !strings.Contains(out, "A\x1b[J") {
		t.Fatalf("expected cursor-up + clear-line redraw, got %q", out)
	}
	if !strings.HasSuffix(out, "\x1b[?25h") {
		t.Fatalf("Stop must restore cursor last, got tail %q", out[len(out)-16:])
	}
}

func TestLiveStageConcurrentUpdates(t *testing.T) {
	var buf bytes.Buffer
	s := NewLiveStage(&buf, LiveOptions{Force: true, Width: 100, Depth: sprites.Depth256})
	s.Start(sprites.MoodWorking)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s.SetStep("s", "step", StepRunning, "")
				s.SetStep("s", "step", StepDone, "")
				s.SetMood(sprites.Mood(j % 6))
			}
		}(i)
	}
	wg.Wait()
	s.Finish(sprites.MoodSuccess, "done")
}

func TestLiveStageNoMotion(t *testing.T) {
	var buf bytes.Buffer
	s := NewLiveStage(&buf, LiveOptions{Force: true, Width: 100, Depth: sprites.Depth16, NoMotion: true})
	if s.Live() {
		t.Fatal("NoMotion must disable live animation")
	}
	s.Start(sprites.MoodWorking)
	s.SetStep("image", "Pull", StepDone, "")
	s.Finish(sprites.MoodSuccess, "done")
	if strings.Contains(buf.String(), "\x1b[?25") {
		t.Fatal("NoMotion output must not touch the cursor")
	}
}

func TestLiveStagePlain(t *testing.T) {
	var buf bytes.Buffer
	s := NewLiveStage(&buf, LiveOptions{Force: true, Width: 100, Depth: sprites.DepthTruecolor, Plain: true})
	if s.Live() {
		t.Fatal("Plain must disable live animation")
	}
	s.Start(sprites.MoodWorking)
	s.SetStep("image", "Pull", StepDone, "")
	s.Finish(sprites.MoodSuccess, "done")
	out := buf.String()
	if strings.Contains(out, "▀") || strings.Contains(out, "▄") {
		t.Fatal("Plain output must not contain sprite art")
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatal("Plain output must be ANSI-free")
	}
}
