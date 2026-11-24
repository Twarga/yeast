package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Twarga/yeast/internal/app"
)

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

func objSchema(props map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func intg(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}
func arr(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}
func boolean(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

var readOnly = map[string]any{"readOnlyHint": true}
var destructive = map[string]any{"destructiveHint": true}

// toolList returns every tool; policy still gates each call server-side.
func (s *Server) toolList() []toolDef {
	inst := str("instance name (defaults to the first running instance)")
	return []toolDef{
		{Name: "vm_status", Description: "Status of all instances (or one)", Annotations: readOnly,
			InputSchema: objSchema(map[string]any{"instance": inst})},
		{Name: "vm_logs", Description: "Serial/console log of an instance", Annotations: readOnly,
			InputSchema: objSchema(map[string]any{"instance": inst, "tail": intg("tail lines")})},
		{Name: "vm_ports", Description: "SSH/service/display port bindings", Annotations: readOnly,
			InputSchema: objSchema(map[string]any{"instance": inst})},
		{Name: "vm_snapshots", Description: "List snapshots for an instance (kind + real size)", Annotations: readOnly,
			InputSchema: objSchema(map[string]any{"instance": inst}, "instance")},
		{Name: "vm_read_file", Description: "Read a file from inside the guest", Annotations: readOnly,
			InputSchema: objSchema(map[string]any{"instance": inst, "path": str("absolute guest path")}, "path")},
		{Name: "vm_screenshot", Description: "PNG screendump of the live desktop (image content block)", Annotations: readOnly,
			InputSchema: objSchema(map[string]any{"instance": inst}, "instance")},

		{Name: "vm_up", Description: "Boot/reconcile all project instances",
			InputSchema: objSchema(map[string]any{})},
		{Name: "vm_down", Description: "Stop all instances",
			InputSchema: objSchema(map[string]any{})},
		{Name: "vm_exec", Description: "Run a command inside a guest (64KiB tail cap; guest-side timeout kills the whole tree)",
			InputSchema: objSchema(map[string]any{
				"instance":  inst,
				"command":   arr("command argv"),
				"timeout_s": intg("seconds; policy-capped"),
				"cwd":       str("guest working directory"),
				"env":       map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
				"user":      str("guest user (sudo)"),
				"stdin_b64": str("base64 stdin"),
			}, "command")},
		{Name: "vm_write_file", Description: "Write a file inside the guest (base64 content)",
			InputSchema: objSchema(map[string]any{
				"instance": inst, "path": str("guest path"), "content_b64": str("base64 content"),
			}, "path", "content_b64")},
		{Name: "vm_copy_in", Description: "Copy a host file into the guest",
			InputSchema: objSchema(map[string]any{
				"instance": inst, "host_path": str("host source"), "guest_path": str("guest destination"),
			}, "host_path", "guest_path")},
		{Name: "vm_copy_out", Description: "Copy a guest file to the host",
			InputSchema: objSchema(map[string]any{
				"instance": inst, "guest_path": str("guest source"), "host_path": str("host destination"),
			}, "guest_path", "host_path")},
		{Name: "vm_input", Description: "Inject keystrokes via QMP (keys chord or text)",
			InputSchema: objSchema(map[string]any{
				"instance": inst, "keys": arr("key chord e.g. [meta_l, ret]"), "text": str("text to type"),
			})},
		{Name: "vm_snapshot", Description: "Snapshot an instance (overlay if stopped, live if running)",
			InputSchema: objSchema(map[string]any{
				"instance": inst, "name": str("snapshot name"), "description": str("description"),
			}, "instance", "name")},
		{Name: "vm_restore", Description: "Restore an instance from a snapshot",
			InputSchema: objSchema(map[string]any{
				"instance": inst, "name": str("snapshot name"),
			}, "instance", "name")},
		{Name: "vm_fork", Description: "Create copy-on-write sibling VMs",
			InputSchema: objSchema(map[string]any{
				"instance": inst, "count": intg("fork count"), "from": str("snapshot name"), "boot": boolean("start immediately"),
			}, "instance")},
		{Name: "vm_export", Description: "Export a disk image (qcow2/rootfs/iso)",
			InputSchema: objSchema(map[string]any{
				"instance":  inst,
				"format":    str("qcow2|rootfs|iso"),
				"output":    str("output file path"),
				"sysprep":   boolean("generalize disk before exporting"),
				"skel_user": str("copy user dotfiles into /etc/skel (iso only)"),
				"builder":   str("iso builder name (default debian-live)"),
			}, "instance", "format", "output")},
		{Name: "vm_destroy", Description: "Destroy all instances (destructive)", Annotations: destructive,
			InputSchema: objSchema(map[string]any{"confirm": boolean("must be true")}, "confirm")},
	}
}

// invokeTool runs one tool call and returns the MCP result object.
func (s *Server) invokeTool(ctx context.Context, name string, raw json.RawMessage) map[string]any {
	var args map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return errorResult(fmt.Errorf("invalid arguments: %w", err))
		}
	}
	str := func(key string) string {
		var v string
		if a, ok := args[key]; ok {
			_ = json.Unmarshal(a, &v)
		}
		return v
	}
	strSlice := func(key string) []string {
		var v []string
		if a, ok := args[key]; ok {
			_ = json.Unmarshal(a, &v)
		}
		return v
	}
	num := func(key string) int {
		var v int
		if a, ok := args[key]; ok {
			_ = json.Unmarshal(a, &v)
		}
		return v
	}
	boolean := func(key string) bool {
		var v bool
		if a, ok := args[key]; ok {
			_ = json.Unmarshal(a, &v)
		}
		return v
	}
	target := app.GuestTargetOptions{ProjectRoot: s.root, Target: str("instance")}
	jsonText := func(v any) map[string]any {
		data, _ := json.MarshalIndent(v, "", "  ")
		return textResult(string(data))
	}

	switch name {
	case "vm_status":
		result, err := s.service.Status(ctx, app.StatusOptions{ProjectRoot: s.root})
		if err != nil {
			return errorResult(err)
		}
		if target.Target != "" {
			for _, inst := range result.Instances {
				if inst.Name == target.Target {
					return jsonText(inst)
				}
			}
			return errorResult(fmt.Errorf("instance %q not found", target.Target))
		}
		return jsonText(result)

	case "vm_logs":
		result, err := s.service.Logs(ctx, app.LogsOptions{
			GuestTargetOptions: target, TailLines: num("tail"),
		})
		if err != nil {
			return errorResult(err)
		}
		return textResult(result.Content)

	case "vm_ports":
		result, err := s.service.Status(ctx, app.StatusOptions{ProjectRoot: s.root})
		if err != nil {
			return errorResult(err)
		}
		type portRow struct {
			Instance string `json:"instance"`
			Kind     string `json:"kind"`
			Host     string `json:"host"`
			HostPort int    `json:"host_port"`
			Guest    int    `json:"guest_port,omitempty"`
		}
		var rows []portRow
		for _, inst := range result.Instances {
			if target.Target != "" && inst.Name != target.Target {
				continue
			}
			if inst.SSHPort > 0 {
				rows = append(rows, portRow{inst.Name, "ssh", inst.ManagementIP, inst.SSHPort, 22})
			}
			for _, p := range inst.Ports {
				rows = append(rows, portRow{inst.Name, "service", p.Host, p.HostPort, p.GuestPort})
			}
			if inst.Display != nil {
				rows = append(rows, portRow{inst.Name, "display/" + inst.Display.Type, inst.Display.Host, inst.Display.Port, 0})
			}
		}
		return jsonText(rows)

	case "vm_snapshots":
		result, err := s.service.Snapshots(ctx, app.SnapshotsOptions{
			ProjectRoot: s.root, Target: target.Target,
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result.Snapshots)

	case "vm_read_file":
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("yeast-mcp-read-%d", time.Now().UnixNano()))
		defer os.Remove(tmp)
		_, err := s.service.Copy(ctx, app.CopyOptions{
			GuestTargetOptions: target,
			Direction:          app.CopyFromGuest,
			Source:             str("path"), Destination: tmp,
			Timeout: 30 * time.Second,
		})
		if err != nil {
			return errorResult(err)
		}
		data, err := os.ReadFile(tmp)
		if err != nil {
			return errorResult(err)
		}
		if len(data) > 64*1024 {
			data = data[len(data)-64*1024:]
			return textResult("[truncated to last 64KiB]\n" + string(data))
		}
		return textResult(string(data))

	case "vm_screenshot":
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("yeast-mcp-shot-%d.png", time.Now().UnixNano()))
		defer os.Remove(tmp)
		result, err := s.service.Screenshot(ctx, app.ScreenshotOptions{
			ProjectRoot: s.root, Target: target.Target, OutputPath: tmp,
		})
		if err != nil {
			return errorResult(err)
		}
		data, err := os.ReadFile(result.Path)
		if err != nil {
			return errorResult(err)
		}
		return imageResult(base64.StdEncoding.EncodeToString(data), "image/png")

	case "vm_up":
		result, err := s.service.Up(ctx, app.UpOptions{ProjectRoot: s.root})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_down":
		result, err := s.service.Down(ctx, app.DownOptions{ProjectRoot: s.root})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_exec":
		if s.policy.SnapshotBeforeExec {
			snapTarget := target.Target
			if snapTarget == "" {
				if st, stErr := s.service.Status(ctx, app.StatusOptions{ProjectRoot: s.root}); stErr == nil {
					for _, inst := range st.Instances {
						if inst.Status == "running" {
							snapTarget = inst.Name
							break
						}
					}
				}
			}
			if snapTarget != "" {
				_, _ = s.service.Snapshot(ctx, app.SnapshotOptions{
					ProjectRoot: s.root, Target: snapTarget,
					Name: fmt.Sprintf("pre-exec-%d", time.Now().Unix()),
				})
			}
		}
		var env map[string]string
		if a, ok := args["env"]; ok {
			_ = json.Unmarshal(a, &env)
		}
		var stdin []byte
		if b := str("stdin_b64"); b != "" {
			stdin, _ = base64.StdEncoding.DecodeString(b)
		}
		timeout := s.policy.ClampExecTimeout(time.Duration(num("timeout_s")) * time.Second)
		result, err := s.service.Exec(ctx, app.ExecOptions{
			GuestTargetOptions: target,
			Command:            strSlice("command"),
			Timeout:            timeout,
			Cwd:                str("cwd"),
			Env:                env,
			User:               str("user"),
			Stdin:              stdin,
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result.Run)

	case "vm_write_file":
		content, err := base64.StdEncoding.DecodeString(str("content_b64"))
		if err != nil {
			return errorResult(fmt.Errorf("invalid content_b64: %w", err))
		}
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("yeast-mcp-write-%d", time.Now().UnixNano()))
		if err := os.WriteFile(tmp, content, 0600); err != nil {
			return errorResult(err)
		}
		defer os.Remove(tmp)
		result, err := s.service.Copy(ctx, app.CopyOptions{
			GuestTargetOptions: target,
			Direction:          app.CopyToGuest,
			Source:             tmp, Destination: str("path"),
			Timeout: 30 * time.Second,
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_copy_in":
		result, err := s.service.Copy(ctx, app.CopyOptions{
			GuestTargetOptions: target,
			Direction:          app.CopyToGuest,
			Source:             str("host_path"), Destination: str("guest_path"),
			Timeout: 5 * time.Minute,
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_copy_out":
		result, err := s.service.Copy(ctx, app.CopyOptions{
			GuestTargetOptions: target,
			Direction:          app.CopyFromGuest,
			Source:             str("guest_path"), Destination: str("host_path"),
			Timeout: 5 * time.Minute,
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_input":
		result, err := s.service.Input(ctx, app.InputOptions{
			ProjectRoot: s.root, Target: target.Target,
			Keys: strSlice("keys"), Text: str("text"),
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_snapshot":
		result, err := s.service.Snapshot(ctx, app.SnapshotOptions{
			ProjectRoot: s.root, Target: target.Target,
			Name: str("name"), Description: str("description"),
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_restore":
		result, err := s.service.Restore(ctx, app.RestoreOptions{
			ProjectRoot: s.root, Target: target.Target, Name: str("name"),
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_fork":
		result, err := s.service.Fork(ctx, app.ForkOptions{
			ProjectRoot: s.root, Target: target.Target,
			Count: num("count"), From: str("from"), Boot: boolean("boot"),
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_export":
		result, err := s.service.Export(ctx, app.ExportOptions{
			GuestTargetOptions: target,
			Format:             str("format"),
			Output:             str("output"),
			Sysprep:            boolean("sysprep"),
			SkelUser:           str("skel_user"),
			Builder:            str("builder"),
		})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	case "vm_destroy":
		if !boolean("confirm") {
			return errorResult(fmt.Errorf("vm_destroy requires confirm: true"))
		}
		result, err := s.service.Destroy(ctx, app.DestroyOptions{ProjectRoot: s.root, KeepFiles: true})
		if err != nil {
			return errorResult(err)
		}
		return jsonText(result)

	default:
		return errorResult(fmt.Errorf("unknown tool %q", name))
	}
}
