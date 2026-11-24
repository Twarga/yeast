package qemu

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net"
	"os"
	"path/filepath"
	"testing"

	rtm "github.com/Twarga/yeast/internal/runtime"
)

type qmpRequest struct {
	Execute   string         `json:"execute"`
	Arguments map[string]any `json:"arguments"`
}

// startFakeQMP runs a scripted QMP server on a unix socket. The handler
// receives each request and returns either a {"return":{}} or an error.
func startFakeQMP(t *testing.T, socketPath string, handler func(req qmpRequest) map[string]any) {
	t.Helper()
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		writer := bufio.NewWriter(conn)
		greeting, _ := json.Marshal(map[string]any{"QMP": map[string]any{"version": map[string]any{"qemu": map[string]any{"major": 8}}}})
		_, _ = writer.Write(append(greeting, '\n'))
		_ = writer.Flush()
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			var req qmpRequest
			if err := json.Unmarshal(line, &req); err != nil {
				continue
			}
			resp := handler(req)
			respLine, _ := json.Marshal(resp)
			_, _ = writer.Write(append(respLine, '\n'))
			_ = writer.Flush()
		}
	}()
}

func okResp() map[string]any { return map[string]any{"return": map[string]any{}} }

func qmpErr(desc string) map[string]any {
	return map[string]any{"error": map[string]any{"class": "GenericError", "desc": desc}}
}

func TestScreenshotPNGDirect(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "qmp.sock")
	var gotFormat any
	var gotFile any
	startFakeQMP(t, sock, func(req qmpRequest) map[string]any {
		if req.Execute == "screendump" {
			gotFormat = req.Arguments["format"]
			gotFile = req.Arguments["filename"]
			// fake QEMU writing the PNG
			f, _ := os.Create(fmt.Sprint(req.Arguments["filename"]))
			defer f.Close()
			_ = png.Encode(f, image.NewRGBA(image.Rect(0, 0, 4, 4)))
		}
		return okResp()
	})

	r := &Runtime{}
	out := filepath.Join(dir, "shot.png")
	inst := rtm.RuntimeInstance{Name: "web", RuntimeDir: dir, PID: 1}
	if err := r.Screenshot(testCtx(), inst, out); err != nil {
		t.Fatal(err)
	}
	if gotFormat != "png" {
		t.Fatalf("format arg = %v", gotFormat)
	}
	if fmt.Sprint(gotFile) != out {
		t.Fatalf("filename = %v want %s", gotFile, out)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output not written: %v", err)
	}
}

func TestScreenshotPPMFallback(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "qmp.sock")
	var dumps int
	startFakeQMP(t, sock, func(req qmpRequest) map[string]any {
		if req.Execute != "screendump" {
			return okResp()
		}
		dumps++
		if _, hasFormat := req.Arguments["format"]; hasFormat {
			return qmpErr("Parameter 'format' is unexpected")
		}
		// write a 2x2 red PPM where QEMU would
		filename := fmt.Sprint(req.Arguments["filename"])
		ppm := "P6\n2 2\n255\n" + string([]byte{
			255, 0, 0, 255, 0, 0,
			255, 0, 0, 255, 0, 0,
		})
		_ = os.WriteFile(filename, []byte(ppm), 0o644)
		return okResp()
	})

	r := &Runtime{}
	out := filepath.Join(dir, "shot.png")
	inst := rtm.RuntimeInstance{Name: "web", RuntimeDir: dir, PID: 1}
	if err := r.Screenshot(testCtx(), inst, out); err != nil {
		t.Fatal(err)
	}
	if dumps != 2 {
		t.Fatalf("expected 2 screendump attempts (png+ppm), got %d", dumps)
	}
	cfg := decodePNGConfig(t, out)
	if cfg.Width != 2 || cfg.Height != 2 {
		t.Fatalf("converted png dims = %dx%d", cfg.Width, cfg.Height)
	}
	if _, err := os.Stat(out + ".ppm"); !os.IsNotExist(err) {
		t.Fatal("ppm temp file was not cleaned up")
	}
}

func TestSendKeysPayload(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "qmp.sock")
	var gotReq qmpRequest
	startFakeQMP(t, sock, func(req qmpRequest) map[string]any {
		if req.Execute == "send-key" {
			gotReq = req
		}
		return okResp()
	})
	r := &Runtime{}
	inst := rtm.RuntimeInstance{Name: "web", RuntimeDir: dir, PID: 1}
	if err := r.SendKeys(testCtx(), inst, []string{"meta_l", "ret"}, 80); err != nil {
		t.Fatal(err)
	}
	keys, ok := gotReq.Arguments["keys"].([]any)
	if !ok || len(keys) != 2 {
		t.Fatalf("keys arg = %#v", gotReq.Arguments["keys"])
	}
	first, _ := keys[0].(map[string]any)
	if first["type"] != "qcode" || first["data"] != "meta_l" {
		t.Fatalf("first key = %#v", first)
	}
	if gotReq.Arguments["hold-time"] != float64(80) {
		t.Fatalf("hold-time = %#v", gotReq.Arguments["hold-time"])
	}
}

func TestSendTextTranslatesChars(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "qmp.sock")
	var chords [][]string
	startFakeQMP(t, sock, func(req qmpRequest) map[string]any {
		if req.Execute == "send-key" {
			var chord []string
			for _, k := range req.Arguments["keys"].([]any) {
				kv, _ := k.(map[string]any)
				chord = append(chord, fmt.Sprint(kv["data"]))
			}
			chords = append(chords, chord)
		}
		return okResp()
	})
	r := &Runtime{}
	inst := rtm.RuntimeInstance{Name: "web", RuntimeDir: dir, PID: 1}
	if err := r.SendText(testCtx(), inst, "Hi\n"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"shift", "h"}, {"i"}, {"ret"}}
	if len(chords) != len(want) {
		t.Fatalf("chords = %v want %v", chords, want)
	}
	for i := range want {
		if len(chords[i]) != len(want[i]) {
			t.Fatalf("chord %d = %v want %v", i, chords[i], want[i])
		}
		for j := range want[i] {
			if chords[i][j] != want[i][j] {
				t.Fatalf("chord %d = %v want %v", i, chords[i], want[i])
			}
		}
	}
}

func TestQCodeForByteCoversPrintableASCII(t *testing.T) {
	for b := byte(32); b < 127; b++ {
		if _, err := qcodeForByte(b); err != nil {
			t.Fatalf("printable byte %q unmapped: %v", string(b), err)
		}
	}
	if _, err := qcodeForByte(0x07); err == nil {
		t.Fatal("control byte should be rejected")
	}
}

func TestScreenshotNoSocket(t *testing.T) {
	r := &Runtime{}
	err := r.Screenshot(testCtx(), rtm.RuntimeInstance{Name: "x", RuntimeDir: t.TempDir()}, "out.png")
	if err == nil {
		t.Fatal("expected socket-missing error")
	}
}

// --- tiny helpers ------------------------------------------------------------

func decodePNGConfig(t *testing.T, path string) (cfg pngConfig) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	return pngConfig{Width: c.Width, Height: c.Height}
}

type pngConfig struct{ Width, Height int }

func testCtx() context.Context { return context.Background() }
