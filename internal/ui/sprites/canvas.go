package sprites

import "strings"

// Canvas is a pixel buffer rendered to the terminal with Unicode half-blocks.
// Each terminal cell packs two vertical pixels: the top pixel drives the
// foreground of '▀' or '▄', the bottom pixel the background — so a W×H pixel
// canvas occupies W columns × H/2 terminal rows.
//
// The default sprite canvas is 22×16 px → 22×8 terminal rows, leaving room
// beside the 16×15 Yeast Cell for budding babies and drifting bubbles.
type Canvas struct {
	W, H int
	px   [][]byte // px[y][x]; 0 = transparent, else a Palette key byte
}

// NewCanvas allocates a transparent W×H pixel canvas. H must be even.
func NewCanvas(w, h int) *Canvas {
	if h%2 != 0 {
		h++
	}
	px := make([][]byte, h)
	for y := range px {
		px[y] = make([]byte, w)
	}
	return &Canvas{W: w, H: h, px: px}
}

// Clear resets every pixel to transparent.
func (c *Canvas) Clear() {
	for y := range c.px {
		for x := range c.px[y] {
			c.px[y][x] = 0
		}
	}
}

// Blit draws a frame's non-transparent pixels at pixel offset (ox, oy).
// '.' pixels are skipped; out-of-bounds pixels are clipped silently.
func (c *Canvas) Blit(frame []string, ox, oy int) {
	for fy, row := range frame {
		y := oy + fy
		if y < 0 || y >= c.H {
			continue
		}
		for fx := 0; fx < len(row); fx++ {
			p := row[fx]
			if p == '.' {
				continue
			}
			x := ox + fx
			if x < 0 || x >= c.W {
				continue
			}
			c.px[y][x] = p
		}
	}
}

// Render converts the canvas into terminal lines at the given color depth.
// DepthNone returns nil (no sprite output at all). Each output line ends with
// a reset so the user's own terminal background shows through transparent
// pixels cleanly.
func (c *Canvas) Render(depth ColorDepth) []string {
	if depth == DepthNone {
		return nil
	}
	lines := make([]string, 0, c.H/2)
	for y := 0; y < c.H; y += 2 {
		var sb strings.Builder
		lastFG, lastBG := byte(0), byte(0)
		dirty := false // whether any SGR state is active
		for x := 0; x < c.W; x++ {
			top, bot := c.px[y][x], c.px[y+1][x]
			switch {
			case top != 0 && bot != 0:
				// Two opaque pixels: '▀' fg=top bg=bottom.
				if !dirty || lastFG != top {
					sb.WriteString(fg(depth, string(top)))
				}
				if !dirty || lastBG != bot {
					sb.WriteString(bg(depth, string(bot)))
				}
				sb.WriteRune('▀')
				lastFG, lastBG, dirty = top, bot, true
			case top != 0:
				// Top opaque only: '▀' fg=top on default background.
				if !dirty || lastFG != top {
					sb.WriteString(fg(depth, string(top)))
				}
				if dirty && lastBG != 0 {
					sb.WriteString("\x1b[49m")
				}
				sb.WriteRune('▀')
				lastFG, lastBG, dirty = top, 0, true
			case bot != 0:
				// Bottom opaque only: '▄' fg=bottom on default background.
				if !dirty || lastFG != bot {
					sb.WriteString(fg(depth, string(bot)))
				}
				if dirty && lastBG != 0 {
					sb.WriteString("\x1b[49m")
				}
				sb.WriteRune('▄')
				lastFG, lastBG, dirty = bot, 0, true
			default:
				// Fully transparent: plain space on default colors.
				if dirty {
					sb.WriteString("\x1b[0m")
					dirty = false
				}
				sb.WriteByte(' ')
				lastFG, lastBG = 0, 0
			}
		}
		if dirty {
			sb.WriteString("\x1b[0m")
		}
		lines = append(lines, sb.String())
	}
	return lines
}
