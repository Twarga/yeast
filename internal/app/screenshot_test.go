package app

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
)

type fakeDisplayRuntime struct {
	screenshotCalls []string
	sentKeys        [][]string
	sentText        []string
	screenshotErr   error
	sendErr         error
	inspectRunning  bool
}

func (f *fakeDisplayRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}
func (f *fakeDisplayRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}
func (f *fakeDisplayRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	return nil
}
func (f *fakeDisplayRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	st := rtm.ProcessStateStopped
	if f.inspectRunning {
		st = rtm.ProcessStateRunning
	}
	return rtm.ProcessInfo{PID: instance.PID, State: st}, nil
}
func (f *fakeDisplayRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}
func (f *fakeDisplayRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}
func (f *fakeDisplayRuntime) DeleteSnapshot(ctx context.Context, plan rtm.DeleteSnapshotPlan) error {
	return nil
}
func (f *fakeDisplayRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}
func (f *fakeDisplayRuntime) Screenshot(ctx context.Context, instance rtm.RuntimeInstance, outputPath string) error {
	f.screenshotCalls = append(f.screenshotCalls, outputPath)
	if f.screenshotErr != nil {
		return f.screenshotErr
	}
	f2, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f2.Close()
	return png.Encode(f2, image.NewRGBA(image.Rect(0, 0, 16, 8)))
}
func (f *fakeDisplayRuntime) SendKeys(ctx context.Context, instance rtm.RuntimeInstance, keys []string, holdTimeMs int) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sentKeys = append(f.sentKeys, append([]string{}, keys...))
	return nil
}
func (f *fakeDisplayRuntime) SendText(ctx context.Context, instance rtm.RuntimeInstance, text string) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sentText = append(f.sentText, text)
	return nil
}

// setupDisplayProject inits a project with one running, display-enabled
// instance and returns root + paths for state assertions.
func setupDisplayProject(t *testing.T, service *Service, display *state.DisplayState) (root string) {
	t.Helper()
	root = t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		t.Fatal(err)
	}
	current := state.New(metadata.ID)
	current.Instances["desk"] = state.InstanceState{
		Status:     "running",
		PID:        4242,
		SSHPort:    2222,
		RuntimeDir: filepath.Join(root, "rt", "desk"),
		Display:    display,
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatal(err)
	}
	return root
}

func displayState() *state.DisplayState {
	return &state.DisplayState{Type: "vnc", Host: "127.0.0.1", Port: 5901, URL: "vnc://127.0.0.1:5901", GPU: "virtio-vga"}
}

func TestScreenshotWritesResult(t *testing.T) {
	service := NewService()
	fake := &fakeDisplayRuntime{inspectRunning: true}
	service.runtime = fake
	root := setupDisplayProject(t, service, displayState())

	out := filepath.Join(t.TempDir(), "shot.png")
	result, err := service.Screenshot(context.Background(), ScreenshotOptions{ProjectRoot: root, Target: "desk", OutputPath: out})
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.screenshotCalls) != 1 {
		t.Fatalf("screenshot calls = %v", fake.screenshotCalls)
	}
	if result.Path != out || result.Width != 16 || result.Height != 8 {
		t.Fatalf("result = %+v", result)
	}
	if result.Bytes == 0 || result.SHA256 == "" {
		t.Fatalf("result missing bytes/sha: %+v", result)
	}
	if _, err := os.Stat(out + ".part"); !os.IsNotExist(err) {
		t.Fatal("part file left behind")
	}
}

func TestScreenshotRequiresDisplay(t *testing.T) {
	service := NewService()
	service.runtime = &fakeDisplayRuntime{inspectRunning: true}
	root := setupDisplayProject(t, service, nil) // no display in state

	_, err := service.Screenshot(context.Background(), ScreenshotOptions{ProjectRoot: root, Target: "desk"})
	if err == nil {
		t.Fatal("expected display-required error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrorCodePrecondition {
		t.Fatalf("expected failed_precondition, got %v", err)
	}
}

func TestScreenshotRequiresRunning(t *testing.T) {
	service := NewService()
	fake := &fakeDisplayRuntime{inspectRunning: false} // reconcile will mark stopped
	service.runtime = fake
	root := setupDisplayProject(t, service, displayState())

	_, err := service.Screenshot(context.Background(), ScreenshotOptions{ProjectRoot: root, Target: "desk"})
	if err == nil {
		t.Fatal("expected not-running error")
	}
}

func TestInputKeysAndText(t *testing.T) {
	service := NewService()
	fake := &fakeDisplayRuntime{inspectRunning: true}
	service.runtime = fake
	root := setupDisplayProject(t, service, displayState())

	res, err := service.Input(context.Background(), InputOptions{ProjectRoot: root, Target: "desk", Keys: []string{"meta_l", "ret"}, HoldTimeMs: 80})
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.sentKeys) != 1 || fake.sentKeys[0][0] != "meta_l" {
		t.Fatalf("sent keys = %v", fake.sentKeys)
	}
	if len(res.Keys) != 2 {
		t.Fatalf("result = %+v", res)
	}

	res, err = service.Input(context.Background(), InputOptions{ProjectRoot: root, Target: "desk", Text: "ls\n"})
	if err != nil {
		t.Fatal(err)
	}
	if fake.sentText[0] != "ls\n" || res.Chars != 3 {
		t.Fatalf("text = %v chars=%d", fake.sentText, res.Chars)
	}
}

func TestInputRequiresExactlyOneSource(t *testing.T) {
	service := NewService()
	service.runtime = &fakeDisplayRuntime{inspectRunning: true}
	root := setupDisplayProject(t, service, displayState())

	if _, err := service.Input(context.Background(), InputOptions{ProjectRoot: root, Target: "desk"}); err == nil {
		t.Fatal("expected missing-input error")
	}
	if _, err := service.Input(context.Background(), InputOptions{ProjectRoot: root, Target: "desk", Keys: []string{"ret"}, Text: "x"}); err == nil {
		t.Fatal("expected both-inputs error")
	}
}
