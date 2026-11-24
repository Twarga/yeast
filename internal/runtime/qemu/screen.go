package qemu

import (
	"bufio"
	"context"
	"fmt"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const qmpDisplayTimeout = 5 * time.Second

// Screenshot dumps the VM framebuffer to outputPath as PNG via QMP
// screendump. PNG format is requested first (QEMU ≥ 7.1); when the build
// rejects the format argument the code falls back to a PPM dump converted
// to PNG in-process.
func (r *Runtime) Screenshot(ctx context.Context, instance rtm.RuntimeInstance, outputPath string) error {
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("output path is required")
	}
	client, err := connectDisplayQMP(instance)
	if err != nil {
		return err
	}
	defer client.close()

	abs, err := filepath.Abs(outputPath)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := client.screendump(abs, "png"); err == nil {
		return nil
	}

	// Fallback: dump PPM (always supported) and convert to PNG in Go.
	ppm := abs + ".ppm"
	if err := client.screendump(ppm, ""); err != nil {
		return fmt.Errorf("qmp screendump failed (png and ppm): %w", err)
	}
	defer os.Remove(ppm)
	if err := convertPPMToPNG(ppm, abs); err != nil {
		return fmt.Errorf("convert screendump ppm to png: %w", err)
	}
	return nil
}

// SendKeys presses one simultaneous QKeyCode chord (e.g. meta_l+ret).
func (r *Runtime) SendKeys(ctx context.Context, instance rtm.RuntimeInstance, keys []string, holdTimeMs int) error {
	if len(keys) == 0 {
		return fmt.Errorf("at least one key is required")
	}
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("key name cannot be empty")
		}
	}
	client, err := connectDisplayQMP(instance)
	if err != nil {
		return err
	}
	defer client.close()
	return client.sendKey(keys, holdTimeMs)
}

const textKeyHoldMs = 60

// SendText types printable ASCII through send-key chords — one chord per
// character (with shift for capitals/symbols).
func (r *Runtime) SendText(ctx context.Context, instance rtm.RuntimeInstance, text string) error {
	if text == "" {
		return fmt.Errorf("text is required")
	}
	client, err := connectDisplayQMP(instance)
	if err != nil {
		return err
	}
	defer client.close()

	unescaped := unescapeInputText(text)
	interKeyDelay := time.Duration(textKeyHoldMs+15) * time.Millisecond
	for i, ch := range unescaped {
		if ch > 127 {
			return fmt.Errorf("input text supports ASCII only (found %q)", string(ch))
		}
		keys, err := qcodeForByte(byte(ch))
		if err != nil {
			return err
		}
		if err := client.sendKey(keys, textKeyHoldMs); err != nil {
			return fmt.Errorf("send-key for %q: %w", string(ch), err)
		}
		if i < len(unescaped)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interKeyDelay):
			}
		}
	}
	return nil
}

func unescapeInputText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
				continue
			case 't':
				b.WriteByte('\t')
				i++
				continue
			case 'r':
				b.WriteByte('\r')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func connectDisplayQMP(instance rtm.RuntimeInstance) (*qmpClient, error) {
	if instance.RuntimeDir == "" {
		return nil, fmt.Errorf("runtime directory is required")
	}
	socketPath := qmpSocketPath(instance.RuntimeDir)
	if _, err := os.Stat(socketPath); err != nil {
		return nil, fmt.Errorf("qmp socket not available at %s (is the instance running?): %w", socketPath, err)
	}
	return newQMPClient(socketPath, qmpDisplayTimeout)
}

// qcodeForByte maps one ASCII byte to a send-key chord. Chords are ordered
// [modifiers..., key] exactly like QEMU expects.
func qcodeForByte(b byte) ([]string, error) {
	if b >= 'a' && b <= 'z' {
		return []string{string(b)}, nil
	}
	if b >= 'A' && b <= 'Z' {
		return []string{"shift", string(b + 32)}, nil
	}
	if b >= '0' && b <= '9' {
		return []string{string(b)}, nil
	}
	switch b {
	case ' ':
		return []string{"spc"}, nil
	case '\n', '\r':
		return []string{"ret"}, nil
	case '\t':
		return []string{"tab"}, nil
	case '-':
		return []string{"minus"}, nil
	case '=':
		return []string{"equal"}, nil
	case '[':
		return []string{"bracket_left"}, nil
	case ']':
		return []string{"bracket_right"}, nil
	case ';':
		return []string{"semicolon"}, nil
	case '\'':
		return []string{"apostrophe"}, nil
	case '`':
		return []string{"grave_accent"}, nil
	case '\\':
		return []string{"backslash"}, nil
	case ',':
		return []string{"comma"}, nil
	case '.':
		return []string{"dot"}, nil
	case '/':
		return []string{"slash"}, nil

	// shifted punctuation
	case '!':
		return []string{"shift", "1"}, nil
	case '@':
		return []string{"shift", "2"}, nil
	case '#':
		return []string{"shift", "3"}, nil
	case '$':
		return []string{"shift", "4"}, nil
	case '%':
		return []string{"shift", "5"}, nil
	case '^':
		return []string{"shift", "6"}, nil
	case '&':
		return []string{"shift", "7"}, nil
	case '*':
		return []string{"shift", "8"}, nil
	case '(':
		return []string{"shift", "9"}, nil
	case ')':
		return []string{"shift", "0"}, nil
	case '_':
		return []string{"shift", "minus"}, nil
	case '+':
		return []string{"shift", "equal"}, nil
	case '{':
		return []string{"shift", "bracket_left"}, nil
	case '}':
		return []string{"shift", "bracket_right"}, nil
	case ':':
		return []string{"shift", "semicolon"}, nil
	case '"':
		return []string{"shift", "apostrophe"}, nil
	case '~':
		return []string{"shift", "grave_accent"}, nil
	case '|':
		return []string{"shift", "backslash"}, nil
	case '<':
		return []string{"shift", "comma"}, nil
	case '>':
		return []string{"shift", "dot"}, nil
	case '?':
		return []string{"shift", "slash"}, nil
	}
	return nil, fmt.Errorf("no QKeyCode mapping for byte 0x%02x", b)
}

// convertPPMToPNG decodes a binary P6 PPM (as written by QEMU screendump)
// and re-encodes it as PNG.
func convertPPMToPNG(ppmPath, pngPath string) error {
	f, err := os.Open(ppmPath)
	if err != nil {
		return err
	}
	defer f.Close()
	img, err := decodePPM(f)
	if err != nil {
		return err
	}
	out, err := os.Create(pngPath)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

func decodePPM(f *os.File) (image.Image, error) {
	br := bufio.NewReader(f)
	magic, err := ppmToken(br)
	if err != nil || magic != "P6" {
		return nil, fmt.Errorf("not a binary PPM (magic %q)", magic)
	}
	wStr, err := ppmToken(br)
	if err != nil {
		return nil, err
	}
	hStr, err := ppmToken(br)
	if err != nil {
		return nil, err
	}
	maxStr, err := ppmToken(br)
	if err != nil {
		return nil, err
	}
	w, err1 := strconv.Atoi(wStr)
	h, err2 := strconv.Atoi(hStr)
	maxv, err3 := strconv.Atoi(maxStr)
	if err1 != nil || err2 != nil || err3 != nil || w <= 0 || h <= 0 || maxv <= 0 || maxv > 255 {
		return nil, fmt.Errorf("invalid ppm header %s %s %s", wStr, hStr, maxStr)
	}
	// P6 PPM spec: exactly one whitespace character separates maxval from raw RGB pixels.
	sep, err := br.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read ppm header separator: %w", err)
	}
	if sep != ' ' && sep != '\t' && sep != '\r' && sep != '\n' {
		return nil, fmt.Errorf("expected whitespace after ppm maxval, got 0x%02x", sep)
	}
	pix := make([]byte, w*h*3)
	if _, err := io.ReadFull(br, pix); err != nil {
		return nil, fmt.Errorf("short ppm pixel data: %w", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		off := i * 4
		src := i * 3
		img.Pix[off] = scalePPM(pix[src], maxv)
		img.Pix[off+1] = scalePPM(pix[src+1], maxv)
		img.Pix[off+2] = scalePPM(pix[src+2], maxv)
		img.Pix[off+3] = 0xff
	}
	return img, nil
}

func scalePPM(v byte, maxv int) byte {
	if maxv == 255 {
		return v
	}
	return byte(int(v) * 255 / maxv)
}

// ppmToken reads one whitespace-delimited token, skipping '#' comments.
func ppmToken(br *bufio.Reader) (string, error) {
	for {
		if err := ppmSkipWhitespace(br); err != nil {
			return "", err
		}
		c, err := br.ReadByte()
		if err != nil {
			return "", err
		}
		if c == '#' {
			if _, err := br.ReadBytes('\n'); err != nil {
				return "", err
			}
			continue
		}
		tok := []byte{c}
		for {
			c, err := br.ReadByte()
			if err != nil {
				return string(tok), nil
			}
			if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
				_ = br.UnreadByte()
				return string(tok), nil
			}
			tok = append(tok, c)
		}
	}
}

func ppmSkipWhitespace(br *bufio.Reader) error {
	for {
		c, err := br.ReadByte()
		if err != nil {
			return err
		}
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			return br.UnreadByte()
		}
	}
}
