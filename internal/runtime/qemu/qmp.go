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

func (c *qmpClient) readGreeting() error {
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

func (c *qmpClient) close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
