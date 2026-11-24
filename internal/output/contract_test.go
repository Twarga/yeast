package output

// Contract tests for the frozen yeast.v1 JSON contract.
//
// These tests pin two guarantees:
//  1. Every --json / --events payload validates against the checked-in
//     JSON Schemas under schemas/yeast.v1/.
//  2. Machine-readable output never contains ANSI escape bytes (0x1b),
//     regardless of color profile or terminal state.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/images"
	"github.com/Twarga/yeast/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var schemaDir = filepath.Join("..", "..", "schemas", "yeast.v1")

func loadSchemaFile(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(schemaDir, name))
	if err != nil {
		t.Fatalf("read schema %s: %v", name, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("schema %s is not valid JSON: %v", name, err)
	}
	return doc
}

func TestSchemaFilesAreWellFormed(t *testing.T) {
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		t.Fatalf("read schema dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one schema file")
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".schema.json") {
			continue
		}
		doc := loadSchemaFile(t, entry.Name())
		if doc["$schema"] == "" {
			t.Errorf("%s: missing $schema", entry.Name())
		}
		if doc["title"] == "" {
			t.Errorf("%s: missing title", entry.Name())
		}
		if doc["$id"] == "" {
			t.Errorf("%s: missing $id", entry.Name())
		}
	}
}

// commandFixtures returns one representative `data` payload per --json
// command, built from the real result types so drift in struct tags fails here.
func commandFixtures() map[string]any {
	ts := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	snap := state.SnapshotState{
		Name:        "clean",
		CreatedAt:   ts,
		Description: "baseline",
		DiskPath:    "/snapshots/clean.qcow2",
	}
	statusInstance := app.StatusInstanceResult{
		Name:         "web",
		Status:       "running",
		PID:          4242,
		ManagementIP: "127.0.0.1",
		SSHPort:      2205,
		Ports: []app.PortForwardResult{
			{Host: "127.0.0.1", HostPort: 8080, GuestPort: 80, Protocol: "tcp", URL: "http://127.0.0.1:8080"},
		},
		User:               "yeast",
		ProvisioningStatus: state.ProvisioningStatusReady,
		Display: &app.DisplayResult{
			Type: "vnc", Host: "127.0.0.1", Port: 5901,
			URL: "vnc://127.0.0.1:5901", GPU: "virtio-vga", Firmware: "bios",
		},
	}
	return map[string]any{
		"clean": app.CleanResult{
			ProjectID: "p1",
			Instances: []app.CleanInstanceResult{{Name: "web", Status: "cleaned", CleanedPIDs: []int{4242}}},
		},
		"copy": app.CopyResult{
			ProjectID:   "p1",
			Instance:    "web",
			Direction:   app.CopyToGuest,
			Source:      "./a.txt",
			Destination: "/home/yeast/a.txt",
			StartedAt:   ts,
			FinishedAt:  ts.Add(time.Second),
			Duration:    time.Second,
		},
		"delete-snapshot": app.DeleteSnapshotResult{ProjectID: "p1", Instance: "web", Snapshot: "clean"},
		"explain": app.ExplainResult{
			Code:     "failed_precondition",
			Title:    "Host precondition failed",
			Cause:    "The host lacks something Yeast requires.",
			Diagnose: []string{"run: yeast doctor"},
			Fix:      []string{"apply guided fixes: yeast doctor --fix --yes"},
		},
		"destroy": app.DestroyResult{
			ProjectID:    "p1",
			Instances:    []app.DestroyInstanceResult{{Name: "web", Status: "destroyed"}},
			FilesDeleted: true,
		},
		"doctor": app.DoctorResult{
			Checks: []app.DoctorCheck{
				{
					Name:    "/dev/kvm",
					Status:  app.CheckStatusBlocker,
					Details: "not found",
					Fix: &app.DoctorFix{
						Action:             app.DoctorFixActionCommand,
						Description:        "enable KVM",
						Command:            []string{"modprobe", "kvm_intel"},
						NeedsSudo:          true,
						ManualInstructions: "enable virtualization in BIOS",
					},
				},
			},
			Blockers:    1,
			Warnings:    0,
			Environment: "native-linux",
			SupportTier: "A",
		},
		"down": app.DownResult{
			ProjectID: "p1",
			Instances: []app.DownInstanceResult{{Name: "web", Status: "stopped"}},
		},
		"exec": app.ExecResult{
			ProjectID: "p1",
			Instance:  "web",
			Run: app.GuestCommandResult{
				Command:    "whoami",
				ExitCode:   0,
				Stdout:     "yeast\n",
				Stderr:     "",
				StartedAt:  ts,
				FinishedAt: ts.Add(50 * time.Millisecond),
				Duration:   50 * time.Millisecond,
				TimedOut:   false,
			},
		},
		"images-clean": app.ImageCleanResult{
			DryRun:    true,
			Removed:   []app.CleanedImageItem{{Name: "ubuntu-22.04", Size: 1024, SizeH: "1K"}},
			TotalSize: 1024,
		},
		"init": app.InitResult{
			ProjectRoot:  "/tmp/proj",
			ConfigPath:   "/tmp/proj/yeast.yaml",
			MetadataPath: "/tmp/proj/.yeast/project.json",
			ProjectID:    "p1",
			Template:     "ubuntu-basic",
			Created:      true,
		},
		"inspect": app.InspectResult{
			ProjectID:     "p1",
			Instance:      statusInstance,
			SnapshotNames: []string{"clean"},
			SnapshotCount: 1,
		},
		"logs": app.LogsResult{ProjectID: "p1", Instance: "web", LogPath: "/vm.log", Content: "boot ok\n"},
		"provision": app.ProvisionResult{
			ProjectID: "p1",
			Instance: app.ProvisionInstanceResult{
				Name:               "web",
				ProvisioningStatus: state.ProvisioningStatusReady,
				SSHAddress:         "127.0.0.1:2205",
				SSHPort:            2205,
			},
		},
		"pull": app.PullResult{
			List:   true,
			Images: []string{"ubuntu-24.04"},
			ImageGroups: []app.ImageGroup{
				{Category: "linux", Images: []images.TrustedImage{{Name: "ubuntu-24.04", URL: "https://example.com/i.qcow2", Checksum: "abc", Category: "linux", Description: "ubuntu", CloudInit: true, Size: "1G"}}},
			},
			CachedImages: []app.CachedImageInfo{{Name: "ubuntu-24.04", Path: "/cache/ubuntu-24.04/image.qcow2"}},
		},
		"restore":  app.RestoreResult{ProjectID: "p1", Instance: "web", Snapshot: snap},
		"snapshot": app.SnapshotResult{ProjectID: "p1", Instance: "web", Snapshot: snap},
		"screenshot": app.ScreenshotResult{
			Instance: "web", Path: "/shots/web-1.png",
			Width: 1280, Height: 800, Bytes: 20480, SHA256: "deadbeef",
		},
		"input": app.InputResult{Instance: "web", Keys: []string{"meta_l", "ret"}},
		"export": app.ExportResult{
			ProjectID: "p1", Instance: "web", Format: "qcow2",
			Path: "/out/web.qcow2", Bytes: 1048576, SHA256: "cafe",
		},
		"boot-iso": app.BootISOResult{
			ISOPath: "/isos/live.iso", Name: "bootiso-1", PID: 4321,
			RuntimeDir: "/home/u/.yeast/boot-iso/bootiso-1", VNCUrl: "vnc://127.0.0.1:5900",
			Screenshot: "/shots/iso.png", Removed: true,
		},
		"fork": app.ForkResult{
			ProjectID: "p1",
			Source:    "web",
			Forks: []app.ForkEntry{
				{Name: "web-fork-1", RuntimeDir: "/run/yeast/p1/web-fork-1", SSHPort: 2223, Status: "stopped"},
				{Name: "web-fork-2", RuntimeDir: "/run/yeast/p1/web-fork-2", SSHPort: 2224, PID: 4242, Status: "running"},
			},
		},
		"migrate": app.MigrateResult{
			ProjectID: "p1", DryRun: false, Instances: 2,
			Changes: []app.MigrateChange{
				{Action: "reconcile", Detail: "instance statuses refreshed"},
				{Instance: "web", Action: "snapshot-kind", Detail: "clean → copy"},
				{Instance: "web", Action: "create-snapshots-dir", Detail: "/p/.yeast/instances/web/snapshots"},
			},
		},
		"snapshots": app.SnapshotsResult{
			ProjectID: "p1",
			Instance:  "web",
			Snapshots: []state.SnapshotState{snap},
		},
		"status": app.StatusResult{ProjectID: "p1", Instances: []app.StatusInstanceResult{statusInstance}},
		"up": app.UpResult{
			ProjectID: "p1",
			Instances: []app.UpInstanceResult{
				{
					Name: "web", Status: "running",
					SSHAddress: "127.0.0.1:2205", SSHPort: 2205,
					Ports: []app.PortForwardResult{{Host: "127.0.0.1", HostPort: 8080, GuestPort: 80, Protocol: "tcp"}},
				},
			},
			Profile: &app.ProfileData{
				Phases: []app.ProfilePhase{{Name: "vm_boot", Duration: time.Second}},
				Total:  2 * time.Second,
			},
		},
		"update": app.UpdateResult{
			CurrentVersion:   "v1.1.4",
			TargetVersion:    "v1.1.5",
			UpdateAvailable:  true,
			CheckOnly:        true,
			ChecksumVerified: true,
			Success:          true,
		},
		"version": "v1.1.5",
	}
}

// TestJSONFixturesConformToSchemas renders every command payload through the
// real RenderJSONSuccess path and validates it against envelope + per-command
// schemas from schemas/yeast.v1/.
func TestJSONFixturesConformToSchemas(t *testing.T) {
	envelopeSchema := loadSchemaFile(t, "envelope.schema.json")
	commandsSchema := loadSchemaFile(t, "commands.schema.json")
	commandDefs, _ := commandsSchema["$defs"].(map[string]any)

	// Shared nested types are $defs but not command names.
	nonCommandDefs := map[string]bool{
		"doctor.result": true, "doctor.fix": true, "doctor.fix-apply": true,
		"doctor.fix-step": true, "init.result": true, "templates": true,
		"trusted-image": true, "status-instance": true, "port-forward": true,
		"snapshot-state": true, "display": true,
		"snapshot-all": true, "snapshot-all-entry": true,
		"restore-all": true, "fork-entry": true,
		"migrate-change": true,
	}
	fixtures := commandFixtures()
	for command := range commandDefs {
		if nonCommandDefs[command] {
			continue
		}
		if _, ok := fixtures[command]; !ok {
			t.Errorf("schema defines command payload %q but no fixture covers it", command)
		}
	}
	for command, fixture := range fixtures {
		if _, ok := commandDefs[command]; !ok {
			t.Errorf("fixture for command %q has no schema definition", command)
			continue
		}
		var buf bytes.Buffer
		if err := RenderJSONSuccess(&buf, command, fixture); err != nil {
			t.Fatalf("%s: render failed: %v", command, err)
		}
		if bytes.ContainsRune(buf.Bytes(), '\x1b') {
			t.Fatalf("%s: JSON output contains ANSI escape byte", command)
		}

		var doc any
		if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
			t.Fatalf("%s: output is not valid JSON: %v", command, err)
		}
		if errs := validateAgainst(envelopeSchema, doc, "$", schemaDefs(envelopeSchema)); len(errs) > 0 {
			t.Fatalf("%s: envelope schema violations:\n%s", command, strings.Join(errs, "\n"))
		}
		obj, _ := doc.(map[string]any)
		if errs := validateAgainst(commandDefs[command], obj["data"], "$.data", commandDefs); len(errs) > 0 {
			t.Fatalf("%s: payload schema violations:\n%s", command, strings.Join(errs, "\n"))
		}
	}
}

// TestErrorAndEventFixturesConform covers the error envelope and the --events
// JSON Lines record.
func TestErrorAndEventFixturesConform(t *testing.T) {
	envelopeSchema := loadSchemaFile(t, "envelope.schema.json")
	eventSchema := loadSchemaFile(t, "event.schema.json")

	var errBuf bytes.Buffer
	appErr := app.WithDetails(
		app.WrapError(app.ErrorCodePrecondition, "kvm required", errors.New("no /dev/kvm")),
		map[string]any{"recovery": []string{"yeast doctor"}},
	)
	if err := RenderJSONError(&errBuf, appErr); err != nil {
		t.Fatalf("render error: %v", err)
	}
	if bytes.ContainsRune(errBuf.Bytes(), '\x1b') {
		t.Fatal("error JSON contains ANSI escape byte")
	}
	var errDoc any
	if err := json.Unmarshal(errBuf.Bytes(), &errDoc); err != nil {
		t.Fatalf("error output is not JSON: %v", err)
	}
	if errs := validateAgainst(envelopeSchema, errDoc, "$", schemaDefs(envelopeSchema)); len(errs) > 0 {
		t.Fatalf("error envelope violations:\n%s", strings.Join(errs, "\n"))
	}

	var evBuf bytes.Buffer
	event := app.NewEvent("up", app.EventVMStarting, app.EventOptions{
		ProjectID: "p1",
		Instance:  "web",
		Message:   "Starting VM",
		Now:       time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
		Data:      map[string]any{"port": 2205},
	})
	if err := RenderJSONEvent(&evBuf, event); err != nil {
		t.Fatalf("render event: %v", err)
	}
	if bytes.ContainsRune(evBuf.Bytes(), '\x1b') {
		t.Fatal("event JSON contains ANSI escape byte")
	}
	var evDoc any
	if err := json.Unmarshal(evBuf.Bytes(), &evDoc); err != nil {
		t.Fatalf("event output is not JSON: %v", err)
	}
	if errs := validateAgainst(eventSchema, evDoc, "$", schemaDefs(eventSchema)); len(errs) > 0 {
		t.Fatalf("event schema violations:\n%s", strings.Join(errs, "\n"))
	}
}

// TestHumanFixturesNeverLeakANSIIntoJSON double-checks the frozen rule that
// machine output stays byte-clean even when data itself contains strings that
// look like terminal styling. Renderers must never inject escapes.
func TestHumanFixturesNeverLeakANSIIntoJSON(t *testing.T) {
	payload := map[string]any{
		"name":    "\x1b[31malready-styled\x1b[0m",
		"details": "plain",
	}
	var buf bytes.Buffer
	if err := RenderJSONSuccess(&buf, "status", payload); err != nil {
		t.Fatalf("render: %v", err)
	}
	// Escapes inside user data are preserved as JSON \u001b escapes, never raw bytes.
	if bytes.ContainsRune(buf.Bytes(), '\x1b') {
		t.Fatal("raw ANSI byte leaked into JSON output")
	}
}

// ─── Minimal JSON Schema validator ──────────────────────────────────────────
// Supports the keyword subset used by schemas/yeast.v1: $ref (local $defs),
// oneOf, anyOf, type, required, properties, additionalProperties, items, enum,
// const, minLength. `format` is annotation-only and ignored.

func validateAgainst(schema any, value any, path string, defs map[string]any) []string {
	s, ok := schema.(map[string]any)
	if !ok {
		return nil
	}

	var errs []string

	if ref, ok := s["$ref"].(string); ok {
		target, err := resolveLocalRef(ref, defs)
		if err != nil {
			return []string{fmt.Sprintf("%s: bad $ref %q: %v", path, ref, err)}
		}
		return validateAgainst(target, value, path, defs)
	}

	if subs, ok := s["oneOf"].([]any); ok {
		matches := 0
		var collected []string
		for _, sub := range subs {
			subErrs := validateAgainst(sub, value, path, defs)
			if len(subErrs) == 0 {
				matches++
			} else {
				collected = append(collected, subErrs...)
			}
		}
		if matches != 1 {
			errs = append(errs, fmt.Sprintf("%s: expected exactly one oneOf match, got %d", path, matches))
			errs = append(errs, collected...)
		}
		return errs
	}
	if subs, ok := s["anyOf"].([]any); ok {
		for _, sub := range subs {
			if len(validateAgainst(sub, value, path, defs)) == 0 {
				return nil
			}
		}
		return []string{fmt.Sprintf("%s: no anyOf branch matched", path)}
	}

	if c, ok := s["const"]; ok {
		if !jsonEqual(c, value) {
			errs = append(errs, fmt.Sprintf("%s: const %v != %v", path, c, value))
		}
	}
	if e, ok := s["enum"].([]any); ok {
		found := false
		for _, candidate := range e {
			if jsonEqual(candidate, value) {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("%s: value %v not in enum", path, value))
		}
	}

	if typ, ok := s["type"].(string); ok && !typeMatches(typ, value) {
		return append(errs, fmt.Sprintf("%s: expected type %s, got %T (%v)", path, typ, value, value))
	}

	switch v := value.(type) {
	case map[string]any:
		if req, ok := s["required"].([]any); ok {
			for _, r := range req {
				if _, present := v[r.(string)]; !present {
					errs = append(errs, fmt.Sprintf("%s: missing required property %q", path, r))
				}
			}
		}
		var props map[string]any
		if p, ok := s["properties"].(map[string]any); ok {
			props = p
			for key, propSchema := range props {
				if val, present := v[key]; present {
					errs = append(errs, validateAgainst(propSchema, val, path+"."+key, defs)...)
				}
			}
		}
		if add, ok := s["additionalProperties"].(bool); ok && !add {
			for key := range v {
				if _, known := props[key]; !known {
					errs = append(errs, fmt.Sprintf("%s: unexpected property %q", path, key))
				}
			}
		}
	case []any:
		if items, ok := s["items"]; ok {
			for i, item := range v {
				errs = append(errs, validateAgainst(items, item, fmt.Sprintf("%s[%d]", path, i), defs)...)
			}
		}
	case string:
		if ml, ok := s["minLength"].(float64); ok && len(v) < int(ml) {
			errs = append(errs, fmt.Sprintf("%s: string shorter than minLength %d", path, int(ml)))
		}
	}
	return errs
}

// schemaDefs extracts the $defs table of a schema document for $ref resolution.
func schemaDefs(schema map[string]any) map[string]any {
	defs, _ := schema["$defs"].(map[string]any)
	return defs
}

// resolveLocalRef resolves "#/$defs/name" against a document's own $defs table.
func resolveLocalRef(ref string, defs map[string]any) (any, error) {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil, fmt.Errorf("unsupported non-local ref %q", ref)
	}
	name := strings.TrimPrefix(ref, prefix)
	if defs == nil {
		return nil, fmt.Errorf("schema has no $defs table")
	}
	target, ok := defs[name]
	if !ok {
		return nil, fmt.Errorf("missing $defs entry %q", name)
	}
	return target, nil
}

func typeMatches(typ string, value any) bool {
	switch typ {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		f, ok := value.(float64)
		return ok && f == float64(int64(f))
	case "number":
		_, ok := value.(float64)
		return ok
	default:
		return false
	}
}

func jsonEqual(a, b any) bool {
	rawA, _ := json.Marshal(a)
	rawB, _ := json.Marshal(b)
	return bytes.Equal(rawA, rawB)
}
