package ui

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Twarga/yeast/internal/ui/sprites"
)

// frameInterval is the animator tick: 85ms ≈ 11.7 FPS.
const frameInterval = 85 * time.Millisecond

// minLiveWidth is the terminal width required for the side-by-side
// sprite + status layout; below it the stage degrades to static lines.
const minLiveWidth = 80

// StepState is the lifecycle of one live status step.
type StepState string

const (
	StepPending StepState = "pending"
	StepRunning StepState = "running"
	StepDone    StepState = "done"
	StepFailed  StepState = "failed"
	StepSkipped StepState = "skipped"
)

type liveStep struct {
	id     string
	label  string
	detail string
	state  StepState
}

// LiveStage draws the animated Yeast Cell beside live step progress for
// long-running commands. It redraws in place (never the alt-screen) so
// scrollback stays intact, always restores the cursor, and degrades to
// plain line output whenever animation is unsafe.
//
// Zero added latency: Finish renders the final settle frame immediately —
// it never sleeps for animation aesthetics.
type LiveStage struct {
	w        io.Writer
	theme    Theme
	depth    sprites.ColorDepth
	utf8     bool
	width    int
	live     bool // true = animated in-place redraw; false = static line mode
	static   bool // true = plain text only (no sprite, minimal symbols)
	noMotion bool // true = show still sprite without tick loop / cursor control

	mu            sync.Mutex
	wmu           sync.Mutex // serializes every write to w
	mood          sprites.Mood
	title         string
	steps         []*liveStep
	order         []string
	renderedLines int
	tick          int
	startedAt     time.Time
	done          chan struct{}
	loopDone      chan struct{} // closed when the render goroutine exits
	loopStarted   bool
	finished      bool
	sigCh         chan os.Signal
}

// LiveOptions tunes a LiveStage; zero values mean "auto-detect".
type LiveOptions struct {
	Width    int                // 0 = detect via WidthFor
	Depth    sprites.ColorDepth // 0 = detect via DepthFor (DepthNone disables sprite art but not the stage)
	Title    string             // panel heading; empty = mood-derived default
	NoMotion bool               // force a single still frame, no tick loop
	Plain    bool               // strip sprite art and symbols entirely
	Force    bool               // animate even when w is not a TTY (tests, captured PTY)
}

// NewLiveStage creates a stage bound to w. Call Start to begin drawing.
func NewLiveStage(w io.Writer, opts LiveOptions) *LiveStage {
	depth := opts.Depth
	if depth == 0 {
		depth = DepthFor(w)
	}
	width := opts.Width
	if width <= 0 {
		width = WidthFor(w)
	}
	s := &LiveStage{
		w:         w,
		theme:     NewTheme(w),
		depth:     depth,
		utf8:      UTF8For(w),
		width:     width,
		mood:      sprites.MoodIdle,
		title:     opts.Title,
		startedAt: time.Now(),
		done:      make(chan struct{}),
		static:    opts.Plain,
		noMotion:  opts.NoMotion,
	}
	animated := (opts.Force || ShouldAnimate(w)) && !opts.NoMotion && !opts.Plain && width >= minLiveWidth
	s.live = animated
	return s
}

// Live reports whether the stage is animating in place.
func (s *LiveStage) Live() bool { return s.live }

// Start begins the render loop. Safe to call once; further calls are no-ops.
func (s *LiveStage) Start(mood sprites.Mood) {
	s.mu.Lock()
	s.mood = mood
	s.startedAt = time.Now()
	live := s.live
	s.mu.Unlock()

	if !live {
		s.renderStaticHeader()
		return
	}
	s.write("\x1b[?25l") // hide cursor
	s.sigCh = make(chan os.Signal, 1)
	signal.Notify(s.sigCh, os.Interrupt, syscall.SIGTERM)
	s.mu.Lock()
	s.loopDone = make(chan struct{})
	s.loopStarted = true
	s.mu.Unlock()
	go s.loop()
}

// SetStep registers or updates a step. First use of an id appends it in
// display order.
func (s *LiveStage) SetStep(id, label string, state StepState, detail string) {
	s.mu.Lock()
	step, ok := s.find(id)
	if !ok {
		step = &liveStep{id: id, label: label}
		s.steps = append(s.steps, step)
		s.order = append(s.order, id)
	}
	prev := step.state
	step.label = label
	step.state = state
	step.detail = detail
	live := s.live
	finished := s.finished
	s.mu.Unlock()

	if !live && !finished && state != prev {
		s.printStaticStep(step, state)
	}
}

// SetMood switches the mascot choreography (working → success, …).
func (s *LiveStage) SetMood(m sprites.Mood) {
	s.mu.Lock()
	s.mood = m
	s.mu.Unlock()
}

// Finish draws the final settle frame, restores the cursor, and stops the
// loop. It returns immediately — never sleeps for animation aesthetics.
func (s *LiveStage) Finish(finalMood sprites.Mood, summary string) {
	s.mu.Lock()
	if s.finished {
		s.mu.Unlock()
		return
	}
	s.finished = true
	s.mood = finalMood
	live := s.live
	showStill := !live && s.noMotion && !s.static && s.depth != sprites.DepthNone && s.width >= minLiveWidth
	s.mu.Unlock()

	if live {
		s.redraw(true)
	} else if showStill {
		// Render a single crisp still sprite beside the panel without touching cursor control.
		s.redraw(true)
	}
	s.shutdown()
	if summary != "" {
		s.write(s.staticText(summary) + "\n")
	}
}

// Stop halts the stage and restores the cursor without a final frame.
func (s *LiveStage) Stop() {
	s.mu.Lock()
	if s.finished {
		s.mu.Unlock()
		return
	}
	s.finished = true
	s.mu.Unlock()
	s.shutdown()
}

func (s *LiveStage) find(id string) (*liveStep, bool) {
	for _, st := range s.steps {
		if st.id == id {
			return st, true
		}
	}
	return nil, false
}

func (s *LiveStage) shutdown() {
	if s.sigCh != nil {
		signal.Stop(s.sigCh)
	}
	s.mu.Lock()
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	started := s.loopStarted
	s.mu.Unlock()
	if started {
		<-s.loopDone // wait for the render goroutine so restore is truly last
	}
	if s.live {
		s.write("\x1b[?25h") // restore cursor — guaranteed
	}
}

// write serializes output to the underlying writer.
func (s *LiveStage) write(str string) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	fmt.Fprint(s.w, str)
}

// loop ticks at ~12 FPS and redraws the composed frame in place.
func (s *LiveStage) loop() {
	defer close(s.loopDone)
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.redraw(false)
		case sig := <-s.sigCh:
			// Ctrl-C/SIGTERM: restore the cursor ourselves (we are the loop —
			// Stop would deadlock waiting on us), then re-raise so the process
			// exits with the right status.
			s.write("\x1b[?25h")
			signal.Reset(sig)
			if p, err := os.FindProcess(os.Getpid()); err == nil {
				_ = p.Signal(sig)
			}
			return
		}
	}
}

// redraw composes sprite + steps side-by-side and paints over the previous
// frame region. Still = settle frame (mood's final step) instead of tick loop.
func (s *LiveStage) redraw(still bool) {
	s.mu.Lock()
	if s.finished && !still {
		s.mu.Unlock()
		return
	}
	var spriteLines []string
	if s.static || s.depth == sprites.DepthNone {
		spriteLines = nil
	} else if still {
		spriteLines = sprites.RenderStill(s.mood, s.depth)
	} else {
		spriteLines = sprites.RenderMood(s.mood, s.tick, s.depth)
	}
	tick := s.tick
	s.tick++
	panel := s.renderPanel(tick)
	prev := s.renderedLines
	s.mu.Unlock()

	var sprite string
	if len(spriteLines) > 0 {
		sprite = strings.Join(spriteLines, "\n")
	}
	var frame string
	if sprite != "" {
		frame = lipgloss.JoinHorizontal(lipgloss.Top, sprite, "   ", panel)
	} else {
		frame = panel
	}

	var b strings.Builder
	if prev > 0 {
		fmt.Fprintf(&b, "\x1b[%dA\x1b[J", prev)
	}
	b.WriteString(frame)
	b.WriteByte('\n')
	s.write(b.String())

	s.mu.Lock()
	s.renderedLines = lipgloss.Height(frame)
	s.mu.Unlock()
}

var brailleSpinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// renderPanel builds the right-hand status block: heading + subtle rule + step rows.
// Clamps line widths to the remaining terminal columns so redraws never wrap.
func (s *LiveStage) renderPanel(tick int) string {
	maxW := 0
	if s.width > sprites.CanvasW+6 {
		maxW = s.width - sprites.CanvasW - 5
	}
	elapsed := ""
	if d := time.Since(s.startedAt); d >= 100*time.Millisecond {
		elapsed = "  " + s.theme.Muted.Render(fmt.Sprintf("%.1fs", d.Seconds()))
	}
	title := s.stepTitle()
	header := s.theme.Header.Render(title) + elapsed
	ruleLen := len(title) + 8
	if ruleLen < 24 {
		ruleLen = 24
	}
	if maxW > 0 && ruleLen > maxW {
		ruleLen = maxW
	}
	ruleChar := "-"
	if s.utf8 && !s.static {
		ruleChar = "─"
	}
	lines := []string{
		header,
		s.theme.Border.Render(strings.Repeat(ruleChar, ruleLen)),
	}
	for _, st := range s.steps {
		label := st.label
		detail := st.detail
		if maxW > 8 {
			avail := maxW - 5 // leading "  " + icon + " "
			if len(label) > avail {
				label = truncateEllipsis(label, avail)
				detail = ""
			} else if detail != "" {
				rem := avail - len(label) - 4 // "  (" + ")"
				if rem <= 3 {
					detail = ""
				} else if len(detail) > rem {
					detail = truncateEllipsis(detail, rem)
				}
			}
		}
		row := "  " + s.stepIcon(st.state, tick) + " " + s.theme.Text.Render(label)
		if detail != "" {
			row += "  " + s.theme.Muted.Render("("+detail+")")
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func truncateEllipsis(s string, maxLen int) string {
	if maxLen <= 1 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func (s *LiveStage) stepTitle() string {
	if s.title != "" && s.mood != sprites.MoodSuccess && s.mood != sprites.MoodError {
		return s.title
	}
	switch s.mood {
	case sprites.MoodWorking:
		return "Working…"
	case sprites.MoodSuccess:
		return "Done"
	case sprites.MoodError:
		return "Something went wrong"
	case sprites.MoodSleep:
		return "Resting"
	default:
		return "Yeast"
	}
}

func (s *LiveStage) stepIcon(state StepState, tick int) string {
	utf8 := s.utf8 && !s.static
	switch state {
	case StepRunning:
		if utf8 {
			return s.theme.Accent.Render(brailleSpinner[tick%len(brailleSpinner)])
		}
		return s.theme.Label.Render(">")
	case StepDone:
		if utf8 {
			return s.theme.Success.Render("✓")
		}
		return s.theme.Success.Render("[ok]")
	case StepFailed:
		if utf8 {
			return s.theme.Blocker.Render("✗")
		}
		return s.theme.Blocker.Render("[x]")
	case StepSkipped:
		if utf8 {
			return s.theme.Muted.Render("–")
		}
		return s.theme.Muted.Render("[-]")
	default:
		if utf8 {
			return s.theme.Muted.Render("○")
		}
		return s.theme.Muted.Render("[ ]")
	}
}

// --- static (non-live) rendering --------------------------------------------

func (s *LiveStage) renderStaticHeader() {
	if !s.static {
		s.write(s.staticText(s.stepTitle()) + "\n")
	}
}

func (s *LiveStage) printStaticStep(st *liveStep, state StepState) {
	switch state {
	case StepRunning:
		s.write(s.staticText("… "+st.label+" "+st.detail) + "\n")
	case StepDone:
		s.write(s.staticText(s.iconGlyph("ok")+" "+st.label) + "\n")
	case StepFailed:
		s.write(s.staticText(s.iconGlyph("fail")+" "+st.label+" "+st.detail) + "\n")
	case StepSkipped:
		s.write(s.staticText(s.iconGlyph("skip")+" "+st.label) + "\n")
	}
}

func (s *LiveStage) iconGlyph(kind string) string {
	if s.utf8 && !s.static {
		switch kind {
		case "ok":
			return "✓"
		case "fail":
			return "✗"
		case "skip":
			return "–"
		}
	}
	switch kind {
	case "ok":
		return "[ok]"
	case "fail":
		return "[x]"
	case "skip":
		return "[-]"
	}
	return ""
}

func (s *LiveStage) staticText(t string) string {
	return strings.TrimRight(t, " ")
}
