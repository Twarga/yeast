package qemu

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

const qmpSocketName = "qmp.sock"

func qmpSocketPath(runtimeDir string) string {
	return fmt.Sprintf("%s/%s", runtimeDir, qmpSocketName)
}

type qmpClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

func newQMPClient(socketPath string, dialTimeout time.Duration) (*qmpClient, error) {
	deadline := time.Now().Add(dialTimeout)
	for {
		if _, err := os.Stat(socketPath); err == nil {
			conn, err := net.DialTimeout("unix", socketPath, dialTimeout)
			if err == nil {
				client := &qmpClient{conn: conn, reader: bufio.NewReader(conn)}
				if err := client.readGreeting(); err != nil {
					_ = conn.Close()
					return nil, err
				}
				if err := client.execute("qmp_capabilities", nil); err != nil {
					_ = conn.Close()
					return nil, err
				}
				return client, nil
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("qmp socket not available at %s", socketPath)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

const defaultQMPCommandTimeout = 30 * time.Second

func (c *qmpClient) setDeadline(d time.Duration) {
	if c.conn != nil && d > 0 {
		_ = c.conn.SetDeadline(time.Now().Add(d))
	}
}

func (c *qmpClient) readGreeting() error {
	c.setDeadline(defaultQMPCommandTimeout)
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("read qmp greeting: %w", err)
	}
	var greeting struct {
		QMP struct {
			Version json.RawMessage `json:"version"`
		} `json:"QMP"`
	}
	if err := json.Unmarshal(line, &greeting); err != nil {
		return fmt.Errorf("parse qmp greeting: %w", err)
	}
	if greeting.QMP.Version == nil {
		return fmt.Errorf("unexpected qmp greeting: %s", string(line))
	}
	return nil
}

func (c *qmpClient) execute(command string, args map[string]any) error {
	c.setDeadline(defaultQMPCommandTimeout)
	req := map[string]any{"execute": command}
	if args != nil {
		req["arguments"] = args
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write qmp command %s: %w", command, err)
	}

	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("read qmp response for %s: %w", command, err)
		}
		var resp struct {
			Return json.RawMessage `json:"return"`
			Error  *struct {
				Class string `json:"class"`
				Desc  string `json:"desc"`
			} `json:"error"`
			Event json.RawMessage `json:"event"`
		}
		if err := json.Unmarshal(line, &resp); err != nil {
			return fmt.Errorf("parse qmp response: %w", err)
		}
		if resp.Event != nil {
			// ignore async events
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("qmp command %s failed: %s", command, resp.Error.Desc)
		}
		return nil
	}
}

// executeQuery runs a command and returns the "return" payload for
// inspection (query-* commands).
func (c *qmpClient) executeQuery(command string, args map[string]any) (json.RawMessage, error) {
	c.setDeadline(defaultQMPCommandTimeout)
	req := map[string]any{"execute": command}
	if args != nil {
		req["arguments"] = args
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("write qmp command %s: %w", command, err)
	}
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("read qmp response for %s: %w", command, err)
		}
		var resp struct {
			Return json.RawMessage `json:"return"`
			Error  *struct {
				Class string `json:"class"`
				Desc  string `json:"desc"`
			} `json:"error"`
			Event json.RawMessage `json:"event"`
		}
		if err := json.Unmarshal(line, &resp); err != nil {
			return nil, fmt.Errorf("parse qmp response: %w", err)
		}
		if resp.Event != nil {
			continue
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("qmp command %s failed: %s", command, resp.Error.Desc)
		}
		return resp.Return, nil
	}
}

func (c *qmpClient) close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// screendump executes QEMU's screendump. format "" lets QEMU pick (PPM on
// old builds); "png" requires QEMU ≥ 7.1.
func (c *qmpClient) screendump(filename, format string) error {
	args := map[string]any{"filename": filename}
	if format != "" {
		args["format"] = format
	}
	return c.execute("screendump", args)
}

// sendKey presses a simultaneous chord of QKeyCode names. holdTimeMs adds a
// hold between press and release (QEMU clamps to its own limit).
func (c *qmpClient) sendKey(keys []string, holdTimeMs int) error {
	keyObjs := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		keyObjs = append(keyObjs, map[string]string{"type": "qcode", "data": key})
	}
	args := map[string]any{"keys": keyObjs}
	if holdTimeMs > 0 {
		args["hold-time"] = holdTimeMs
	}
	return c.execute("send-key", args)
}
