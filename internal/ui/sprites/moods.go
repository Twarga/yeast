package sprites

// Mood selects the Yeast Cell's animation choreography. Each mood is a list of
// Steps — a Step is one rendered tick expressed as a set of frame placements
// on the shared 22×16 pixel canvas (22×8 terminal rows).
type Mood int

const (
	MoodIdle Mood = iota
	MoodWorking
	MoodSuccess
	MoodError
	MoodSleep
	MoodQuirk
)

func (m Mood) String() string {
	switch m {
	case MoodWorking:
		return "working"
	case MoodSuccess:
		return "success"
	case MoodError:
		return "error"
	case MoodSleep:
		return "sleep"
	case MoodQuirk:
		return "quirk"
	default:
		return "idle"
	}
}

// CanvasW and CanvasH are the mascot stage dimensions in pixels (24×9 terminal rows).
// The Yeast Cell (16×15) sits at (cellOX=2, cellBaseY=3), leaving 3px of headroom
// for the sleep nightcap + rising bubbles and 6px on the right for budding VMs.
const (
	CanvasW   = 24
	CanvasH   = 18
	cellOX    = 2
	cellBaseY = 3
)

// Placement draws one named frame at pixel offset (OX, OY) on the canvas.
type Placement struct {
	Frame string
	OX    int
	OY    int
}

// Step is a single animation frame: the full set of placements composing it.
type Step []Placement

func p(frame string, ox, oy int) Placement { return Placement{Frame: frame, OX: ox, OY: oy} }

// cellStep builds a Step whose base is the yeast cell at (ox, oy) plus the
// given overlay placements. Overlay frames (eyes, mouth, blush) are authored
// relative to the cell origin, so callers pass cell-relative names and the
// helper anchors them at the same offset.
func cellStep(ox, oy int, overlays ...string) Step {
	s := Step{p("cell", ox, oy)}
	for _, o := range overlays {
		s = append(s, p(o, ox, oy))
	}
	return s
}

// Steps returns the animation loop for a mood. Looping the returned slice at
// the animator's tick rate (~12 FPS) yields the choreography described per
// mood below.
func Steps(m Mood) []Step {
	switch m {
	case MoodWorking:
		return workingSteps
	case MoodSuccess:
		return successSteps
	case MoodError:
		return errorSteps
	case MoodSleep:
		return sleepSteps
	case MoodQuirk:
		return quirkSteps
	default:
		return idleSteps
	}
}

// Still returns the crisp settle frame for a mood — used for --no-motion,
// reduced-motion, narrow terminals, and settle states.
func Still(m Mood) Step {
	switch m {
	case MoodWorking:
		return workingSteps[len(workingSteps)-1]
	case MoodSuccess:
		return successSteps[len(successSteps)-1]
	case MoodError:
		return errorSteps[len(errorSteps)-1]
	case MoodSleep:
		return sleepSteps[4]
	case MoodQuirk:
		return quirkSteps[4]
	default:
		return idleSteps[0]
	}
}

// Compose renders a Step onto a fresh 24×18 canvas.
func Compose(step Step, depth ColorDepth) []string {
	c := NewCanvas(CanvasW, CanvasH)
	for _, pl := range step {
		if frame, ok := Frames[pl.Frame]; ok {
			c.Blit(frame, pl.OX, pl.OY)
		}
	}
	return c.Render(depth)
}

// RenderMood renders mood m at loop index step (mod the loop length).
func RenderMood(m Mood, step int, depth ColorDepth) []string {
	steps := Steps(m)
	return Compose(steps[step%len(steps)], depth)
}

// RenderStill renders a mood's settle frame.
func RenderStill(m Mood, depth ColorDepth) []string {
	return Compose(Still(m), depth)
}

// idleSteps: calm organic breathing (12-tick inhale/exhale hold, never strobing),
// a natural 3-tick blink every ~4 seconds, and delicate fizz bubbles floating
// smoothly upward and popping in the headroom above the cell.
var idleSteps = func() []Step {
	const loop = 48
	steps := make([]Step, loop)
	bubblePath := []Placement{
		p("bubble", 18, 12),
		p("bubble", 19, 10),
		p("bubble", 20, 8),
		p("bubble", 20, 6),
		p("bubble", 21, 4),
		p("pop", 21, 1),
		{}, {},
	}
	for i := 0; i < loop; i++ {
		// Gentle 2-second breathing cycle: 12 ticks resting, 12 ticks lifted.
		oy := cellBaseY
		if (i/12)%2 == 1 {
			oy = cellBaseY - 1
		}
		eyes := "eyes_open"
		if i >= 34 && i <= 36 {
			eyes = "eyes_closed" // natural ~250ms blink
		}
		s := cellStep(cellOX, oy, eyes, "mouth_smile")
		// Bubble advances smoothly every 2 ticks.
		if b := bubblePath[(i/2)%len(bubblePath)]; b.Frame != "" {
			s = append(s, b)
		}
		steps[i] = s
	}
	return steps
}()

// workingSteps: rich ~2.0s budding choreography (24 ticks at 12 FPS) —
// cell leans left, sprouts a tiny bud, swells it, stretches joyfully as a
// baby VM pops off into midair, lands softly beside the parent, and fizzes.
var workingSteps = func() []Step {
	hold := func(n int, st Step) []Step {
		out := make([]Step, n)
		for i := range out {
			out[i] = st
		}
		return out
	}
	var steps []Step
	// Ready stance
	steps = append(steps, hold(3, cellStep(cellOX, cellBaseY, "eyes_open", "mouth_smile"))...)
	// Lean left & sprout tiny bud
	steps = append(steps, hold(3, append(cellStep(cellOX-1, cellBaseY, "eyes_open", "mouth_smile"), p("bud_s", 17, 14)))...)
	// Push & swell medium bud + first fizz bubble
	steps = append(steps, hold(3, append(cellStep(cellOX-1, cellBaseY-1, "eyes_open", "mouth_smile"), p("bud_m", 17, 12), p("bubble", 20, 8)))...)
	// Peak stretch as bud pinches off
	steps = append(steps, hold(3, append(cellStep(cellOX, cellBaseY-1, "eyes_happy", "mouth_open"), p("bud_m", 18, 11), p("bubble", 21, 5)))...)
	// Pop! Baby VM launches upward with celebratory fizz
	steps = append(steps, hold(2, append(cellStep(cellOX, cellBaseY-1, "eyes_happy", "mouth_open"), p("baby", 17, 9), p("bubble", 21, 3)))...)
	steps = append(steps, hold(2, append(cellStep(cellOX, cellBaseY, "eyes_happy", "mouth_smile"), p("baby", 17, 11), p("pop", 21, 1)))...)
	// Baby VM lands softly beside the parent
	steps = append(steps, hold(4, append(cellStep(cellOX, cellBaseY, "eyes_open", "mouth_smile"), p("baby", 17, 12)))...)
	// Proud moment together
	steps = append(steps, hold(4, append(cellStep(cellOX, cellBaseY-1, "eyes_happy", "mouth_smile"), p("baby", 17, 12)))...)
	return steps
}()

// successSteps: joyful celebratory hop + twin rising fizz bubbles that pop,
// settling onto a warm proud smile with rosy cheeks.
var successSteps = []Step{
	append(cellStep(cellOX, cellBaseY, "eyes_happy", "mouth_open", "blush_big"), p("bubble", 19, 8)),
	append(cellStep(cellOX, cellBaseY-1, "eyes_happy", "mouth_open", "blush_big"), p("bubble", 20, 6), p("bubble", 0, 8)),
	append(cellStep(cellOX, cellBaseY-2, "eyes_happy", "mouth_open", "blush_big"), p("bubble", 21, 4), p("bubble", 0, 5)),
	append(cellStep(cellOX, cellBaseY-2, "eyes_happy", "mouth_open", "blush_big"), p("pop", 21, 1), p("bubble", 1, 3)),
	append(cellStep(cellOX, cellBaseY-1, "eyes_happy", "mouth_smile", "blush_big"), p("pop", 0, 1), p("baby", 17, 12)),
	append(cellStep(cellOX, cellBaseY, "eyes_happy", "mouth_smile"), p("baby", 17, 12)),
}

// errorSteps: crisp micro-shiver (x ±1), strained eyes, frown, a stuck bud,
// and a burst bubble — settling calmly so the error card beside it reads cleanly.
var errorSteps = []Step{
	append(cellStep(cellOX-1, cellBaseY, "eyes_strain", "mouth_frown"), p("bud_m", 18, 13)),
	append(cellStep(cellOX+1, cellBaseY, "eyes_strain", "mouth_frown"), p("bud_m", 18, 13), p("pop", 20, 6)),
	append(cellStep(cellOX-1, cellBaseY, "eyes_strain", "mouth_frown"), p("bud_m", 18, 13)),
	append(cellStep(cellOX+1, cellBaseY, "eyes_strain", "mouth_frown"), p("bud_m", 18, 13)),
	append(cellStep(cellOX, cellBaseY, "eyes_strain", "mouth_frown"), p("bud_m", 18, 13)),
	append(cellStep(cellOX, cellBaseY, "eyes_strain", "mouth_frown"), p("bud_m", 18, 13)),
}

// sleepSteps: peaceful slow breathing with an unclipped purple nightcap + pom-pom
// (cap at oy-2 = 0..1, always in bounds) and a 'z' drifting smoothly upward.
var sleepSteps = func() []Step {
	const loop = 24
	steps := make([]Step, loop)
	zPath := []Placement{
		p("z", 18, 10),
		p("z", 19, 8),
		p("z", 20, 6),
		p("z", 20, 4),
		p("z", 20, 2),
		{}, // brief pause before next z rises
	}
	for i := 0; i < loop; i++ {
		oy := cellBaseY
		if i >= 12 {
			oy = cellBaseY - 1
		}
		s := append(cellStep(cellOX, oy, "eyes_closed", "mouth_smile"), p("cap", cellOX+1, oy-2))
		if z := zPath[(i/4)%len(zPath)]; z.Frame != "" {
			s = append(s, z)
		}
		steps[i] = s
	}
	return steps
}()

// quirkSteps: playful fizzing easter egg — double-hiccup bounce, big blush,
// and twin buds fizzing off both sides like shaken soda.
var quirkSteps = []Step{
	append(cellStep(cellOX, cellBaseY, "eyes_open", "mouth_smile", "blush_big"), p("bud_s", 0, 14)),
	append(cellStep(cellOX, cellBaseY, "eyes_open", "mouth_smile", "blush_big"), p("bud_s", 0, 14)),
	append(cellStep(cellOX, cellBaseY-1, "eyes_open", "mouth_smile", "blush_big"), p("bud_s", 0, 12), p("bud_s", 20, 14)),
	append(cellStep(cellOX, cellBaseY-1, "eyes_happy", "mouth_open", "blush_big"), p("bud_m", 0, 10), p("bud_s", 20, 12)),
	append(cellStep(cellOX, cellBaseY-2, "eyes_happy", "mouth_open", "blush_big"), p("bud_m", 1, 8), p("bud_s", 21, 10)),
	append(cellStep(cellOX, cellBaseY-1, "eyes_open", "mouth_smile", "blush_big"), p("bubble", 0, 5), p("bud_s", 21, 8)),
	append(cellStep(cellOX, cellBaseY, "eyes_happy", "mouth_smile", "blush_big"), p("pop", 0, 2), p("pop", 21, 4)),
	cellStep(cellOX, cellBaseY, "eyes_happy", "mouth_smile", "blush_big"),
}
