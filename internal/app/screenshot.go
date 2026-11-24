package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
)

type displayContext struct {
	instanceName string
	state        state.InstanceState
	projectDir   string
	runtime      rtm.RuntimeInstance
}

// resolveDisplayInstance loads project state, reconciles liveness, selects
// the target running instance, and requires display to be enabled.
func (s *Service) resolveDisplayInstance(ctx context.Context, projectRoot, target string) (displayContext, error) {
	var dc displayContext
	root := projectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return dc, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return dc, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return dc, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return dc, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return dc, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return dc, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return dc, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	changed := state.Reconcile(&currentState, state.ReconcileOptions{
		IsProcessAlive: func(pid int) bool {
			info, err := s.runtime.Inspect(ctx, rtm.RuntimeInstance{PID: pid})
			return err == nil && info.State == rtm.ProcessStateRunning
		},
	})
	if changed {
		if err := state.Save(paths.StateFile, currentState); err != nil {
			return dc, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}

	name, instState, err := selectSSHInstance(currentState, target)
	if err != nil {
		return dc, err
	}
	if instState.Display == nil {
		return dc, WrapError(ErrorCodePrecondition, fmt.Sprintf(
			"instance %q has no display enabled; add display.enabled: true to yeast.yaml and re-run yeast up", name), nil)
	}
	dc = displayContext{
		instanceName: name,
		state:        instState,
		projectDir:   paths.ProjectDir,
		runtime: rtm.RuntimeInstance{
			Name:       name,
			RuntimeDir: instState.RuntimeDir,
			PID:        instState.PID,
		},
	}
	return dc, nil
}

type ScreenshotOptions struct {
	ProjectRoot string
	Target      string
	OutputPath  string
}

type ScreenshotResult struct {
	Instance string `json:"instance"`
	Path     string `json:"path"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Bytes    int64  `json:"bytes"`
	SHA256   string `json:"sha256"`
}

func (s *Service) Screenshot(ctx context.Context, options ScreenshotOptions) (result ScreenshotResult, err error) {
	started := time.Now()
	dc, err := s.resolveDisplayInstance(ctx, options.ProjectRoot, options.Target)
	if err != nil {
		return ScreenshotResult{}, err
	}
	defer func() {
		root := options.ProjectRoot
		if root == "" {
			root = "."
		}
		abs, absErr := filepath.Abs(root)
		if absErr == nil {
			s.auditCall(ctx, abs, "screenshot", map[string]any{"instance": options.Target}, started, err)
		}
	}()
	controller, ok := s.runtime.(rtm.DisplayController)
	if !ok {
		return ScreenshotResult{}, WrapError(ErrorCodePrecondition, "runtime does not support display control", nil)
	}

	out := options.OutputPath
	if out == "" {
		dir := filepath.Join(dc.projectDir, "screenshots")
		out = filepath.Join(dir, fmt.Sprintf("%s-%s.png", dc.instanceName, time.Now().UTC().Format("20060102-150405")))
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return ScreenshotResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return ScreenshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	// Atomic write: dump to a part file, then rename into place.
	part := abs + ".part"
	defer func() { _ = os.Remove(part) }()
	if err := controller.Screenshot(ctx, dc.runtime, part); err != nil {
		return ScreenshotResult{}, WrapError(ErrorCodeRuntime, fmt.Sprintf("screenshot %s: %v", dc.instanceName, err), err)
	}
	if err := os.Rename(part, abs); err != nil {
		return ScreenshotResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("move screenshot into place: %v", err), err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return ScreenshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	width, height := pngDimensions(abs)
	sum, err := fileSHA256(abs)
	if err != nil {
		return ScreenshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	return ScreenshotResult{
		Instance: dc.instanceName,
		Path:     abs,
		Width:    width,
		Height:   height,
		Bytes:    info.Size(),
		SHA256:   sum,
	}, nil
}

func pngDimensions(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
