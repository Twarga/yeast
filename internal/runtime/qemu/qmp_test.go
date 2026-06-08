package qemu

import (
	"bufio"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeQMPPowerdownSendsSystemPowerdown(t *testing.T) {
	root := t.TempDir()
	socketPath := filepath.Join(root, "qmp.sock")

	// Start a fake QMP server
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on unix socket: %v", err)
	}
	defer listener.Close()

	commands := make(chan string, 10)
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		writer := bufio.NewWriter(conn)
		// Send greeting
		greeting := map[string]any{"QMP": map[string]any{"version": map[string]any{"qemu": map[string]any{"major": 8}}}}
		greetingLine, _ := json.Marshal(greeting)
		_, _ = writer.Write(append(greetingLine, '\n'))
		_ = writer.Flush()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			var req struct {
				Execute string `json:"execute"`
			}
			if err := json.Unmarshal(line, &req); err != nil {
				continue
			}
			commands <- req.Execute

			resp := map[string]any{"return": map[string]any{}}
			respLine, _ := json.Marshal(resp)
			_, _ = writer.Write(append(respLine, '\n'))
			_ = writer.Flush()
		}
	}()

	client, err := newQMPClient(socketPath, 2*time.Second)
	if err != nil {
		t.Fatalf("newQMPClient returned error: %v", err)
	}
	defer client.close()

	if err := client.execute("system_powerdown", nil); err != nil {
		t.Fatalf("execute system_powerdown returned error: %v", err)
	}

	close(commands)
	var got []string
	for cmd := range commands {
		got = append(got, cmd)
	}

	want := []string{"qmp_capabilities", "system_powerdown"}
	if len(got) != len(want) {
		t.Fatalf("expected %d commands, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected command %q, got %q", want[i], got[i])
		}
	}
}

func TestQMPClientRetriesUntilSocketExists(t *testing.T) {
	root := t.TempDir()
	socketPath := filepath.Join(root, "qmp.sock")

	// Start server in background after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			return
		}
		defer listener.Close()

		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		writer := bufio.NewWriter(conn)
		greeting := map[string]any{"QMP": map[string]any{"version": map[string]any{"qemu": map[string]any{"major": 8}}}}
		greetingLine, _ := json.Marshal(greeting)
		_, _ = writer.Write(append(greetingLine, '\n'))
		_ = writer.Flush()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			var req struct {
				Execute string `json:"execute"`
			}
			if err := json.Unmarshal(line, &req); err != nil {
				continue
			}
			resp := map[string]any{"return": map[string]any{}}
			respLine, _ := json.Marshal(resp)
			_, _ = writer.Write(append(respLine, '\n'))
			_ = writer.Flush()
		}
	}()

	client, err := newQMPClient(socketPath, 2*time.Second)
	if err != nil {
		t.Fatalf("newQMPClient returned error: %v", err)
	}
	client.close()
}
