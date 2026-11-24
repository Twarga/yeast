package sprites

import "fmt"

// ColorDepth selects how many colors the terminal can display.
type ColorDepth int

const (
	// DepthNone disables sprite rendering entirely (no ANSI, no sprite lines).
	DepthNone ColorDepth = iota
	// Depth16 uses hand-picked ANSI-16 approximations of the Yeast palette.
	Depth16
	// Depth256 uses nearest xterm-256 indices, precomputed for the palette.
	Depth256
	// DepthTruecolor emits full 24-bit RGB sequences.
	DepthTruecolor
)

// ansi16Table is the hand-picked ANSI-16 approximation of the Yeast palette:
// outline → black, body → bright green, shade → green, highlight → bright
// cyan, cream → white, blush → bright magenta, nightcap → magenta.
var ansi16Table = map[string]uint8{
	"#": 0,
	"B": 10,
	"S": 2,
	"H": 14,
	"W": 15,
	"P": 13,
	"V": 5,
}

// xtermTable maps palette keys to the nearest xterm-256 color index.
// Computed once at init from the true hex values.
var xtermTable = func() map[string]uint8 {
	t := make(map[string]uint8, len(Palette))
	for key, hex := range Palette {
		t[key] = nearestXterm256(hex)
	}
	return t
}()

func hexRGB(hex string) (r, g, b int) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0
	}
	_, _ = fmt.Sscanf(hex[1:], "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// nearestXterm256 finds the closest xterm-256 index among the 6×6×6 color
// cube and the 24-step grayscale ramp using squared euclidean distance.
func nearestXterm256(hex string) uint8 {
	r, g, b := hexRGB(hex)
	best, bestDist := 0, 1<<62
	cube := func(i int) int {
		if i == 0 {
			return 0
		}
		return 55 + i*40
	}
	for ri := 0; ri < 6; ri++ {
		for gi := 0; gi < 6; gi++ {
			for bi := 0; bi < 6; bi++ {
				d := sq(cube(ri)-r) + sq(cube(gi)-g) + sq(cube(bi)-b)
				if d < bestDist {
					bestDist = d
					best = 16 + 36*ri + 6*gi + bi
				}
			}
		}
	}
	for i := 0; i < 24; i++ {
		v := 8 + i*10
		d := sq(v-r) + sq(v-g) + sq(v-b)
		if d < bestDist {
			bestDist = d
			best = 232 + i
		}
	}
	return uint8(best)
}

func sq(x int) int { return x * x }

// fg returns the ANSI sequence setting the foreground to a palette color.
func fg(depth ColorDepth, key string) string {
	switch depth {
	case DepthTruecolor:
		r, g, b := hexRGB(Palette[key])
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	case Depth256:
		return fmt.Sprintf("\x1b[38;5;%dm", xtermTable[key])
	case Depth16:
		idx := ansi16Table[key]
		if idx < 8 {
			return fmt.Sprintf("\x1b[3%dm", idx)
		}
		return fmt.Sprintf("\x1b[9%dm", idx-8)
	}
	return ""
}

// bg returns the ANSI sequence setting the background to a palette color.
func bg(depth ColorDepth, key string) string {
	switch depth {
	case DepthTruecolor:
		r, g, b := hexRGB(Palette[key])
		return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	case Depth256:
		return fmt.Sprintf("\x1b[48;5;%dm", xtermTable[key])
	case Depth16:
		idx := ansi16Table[key]
		if idx < 8 {
			return fmt.Sprintf("\x1b[4%dm", idx)
		}
		return fmt.Sprintf("\x1b[10%dm", idx-8)
	}
	return ""
}
