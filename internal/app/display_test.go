package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Twarga/yeast/internal/config"
	"github.com/Twarga/yeast/internal/state"
)

func TestBuildDisplayPlanDisabled(t *testing.T) {
	instance := config.Instance{Name: "web"}
	plan, err := buildDisplayPlan(instance, nil, map[string]string{})
	if err != nil || plan != nil {
		t.Fatalf("disabled display should produce nil plan, got %+v err=%v", plan, err)
	}

	instance.Display = &config.DisplayConfig{Enabled: false, Type: "vnc", Port: 5901}
	plan, err = buildDisplayPlan(instance, nil, map[string]string{})
	if err != nil || plan != nil {
		t.Fatalf("enabled=false should produce nil plan, got %+v err=%v", plan, err)
	}
}

func TestBuildDisplayPlanAutoPort(t *testing.T) {
	instance := config.Instance{
		Name:    "desktop",
		Display: &config.DisplayConfig{Enabled: true, Type: "vnc", Host: "127.0.0.1", GPU: "virtio-vga", Resolution: "1280x800"},
	}
	allocated := map[string]string{}
	plan, err := buildDisplayPlan(instance, func(int) bool { return true }, allocated)
	if err != nil {
		t.Fatal(err)
	}
	if plan == nil || plan.Port < displayPortAutoFirst || plan.Port > displayPortAutoLast {
		t.Fatalf("auto port out of range: %+v", plan)
	}
	if plan.Width != 1280 || plan.Height != 800 {
		t.Fatalf("resolution = %dx%d", plan.Width, plan.Height)
	}
	if allocated["127.0.0.1:5901"] == "" {
		t.Fatal("auto port was not reserved in allocated binds")
	}

	// second instance must not reuse the reserved port
	plan2, err := buildDisplayPlan(instance, func(int) bool { return true }, allocated)
	if err != nil {
		t.Fatal(err)
	}
	if plan2.Port == plan.Port {
		t.Fatalf("second auto allocation reused port %d", plan2.Port)
	}
}

func TestBuildDisplayPlanCollisionAndBusy(t *testing.T) {
	instance := config.Instance{
		Name:    "desktop",
		Display: &config.DisplayConfig{Enabled: true, Type: "vnc", Host: "127.0.0.1", Port: 5901},
	}
	allocated := map[string]string{"127.0.0.1:5901": "web ssh"}
	if _, err := buildDisplayPlan(instance, func(int) bool { return true }, allocated); err == nil {
		t.Fatal("expected collision with reserved bind")
	}
	if _, err := buildDisplayPlan(instance, func(int) bool { return false }, map[string]string{}); err == nil {
		t.Fatal("expected busy host port error")
	}
}

func TestResolveMountPlans(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dotfiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "afile"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	defer func(saved func() (string, error)) { findVirtiofsd = saved }(findVirtiofsd)
	findVirtiofsd = func() (string, error) { return "", errors.New("not found") }

	instance := config.Instance{
		Name: "web",
		Mounts: []config.MountConfig{
			{Source: "./dotfiles", Target: "/mnt/dotfiles", Driver: "auto"},
		},
	}
	plans, err := resolveMountPlans(root, instance)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans = %+v", plans)
	}
	abs := filepath.Join(root, "dotfiles")
	if plans[0].Source != abs {
		t.Fatalf("source = %q want %q", plans[0].Source, abs)
	}
	if plans[0].Driver != "9p" {
		t.Fatalf("auto without virtiofsd = %q, want 9p", plans[0].Driver)
	}
	if plans[0].Tag != "yeast_mnt_0" {
		t.Fatalf("tag = %q", plans[0].Tag)
	}

	// auto prefers virtiofs when the daemon exists
	findVirtiofsd = func() (string, error) { return "/usr/libexec/virtiofsd", nil }
	plans, err = resolveMountPlans(root, instance)
	if err != nil {
		t.Fatal(err)
	}
	if plans[0].Driver != "virtiofs" {
		t.Fatalf("auto with virtiofsd = %q", plans[0].Driver)
	}

	// explicit virtiofs without the daemon errors
	findVirtiofsd = func() (string, error) { return "", errors.New("not found") }
	instance.Mounts[0].Driver = "virtiofs"
	if _, err := resolveMountPlans(root, instance); err == nil {
		t.Fatal("expected virtiofs-missing error")
	}

	// missing source errors
	instance.Mounts[0] = config.MountConfig{Source: "./nope", Target: "/mnt/x", Driver: "9p"}
	if _, err := resolveMountPlans(root, instance); err == nil {
		t.Fatal("expected missing source error")
	}

	// non-directory source errors
	instance.Mounts[0] = config.MountConfig{Source: "./afile", Target: "/mnt/x", Driver: "9p"}
	if _, err := resolveMountPlans(root, instance); err == nil {
		t.Fatal("expected non-directory source error")
	}
}

func TestBuildDisplayStateAndResult(t *testing.T) {
	if got := buildDisplayState(nil, "bios"); got != nil {
		t.Fatalf("nil plan = %+v", got)
	}
	plan, err := buildDisplayPlan(config.Instance{
		Name:    "desk",
		Display: &config.DisplayConfig{Enabled: true, Type: "vnc", Port: 5902, GPU: "virgl"},
	}, func(int) bool { return true }, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	st := buildDisplayState(plan, "uefi")
	if st == nil || st.URL != "vnc://127.0.0.1:5902" || st.GPU != "virgl" || st.Firmware != "uefi" {
		t.Fatalf("state = %+v", st)
	}
	res := displayResultFromState(st)
	if res == nil || res.URL != st.URL {
		t.Fatalf("result = %+v", res)
	}
	if displayResultFromState(nil) != nil {
		t.Fatal("nil state should give nil result")
	}

	// round-trip through persisted state
	mounts := buildMountStates(nil)
	if mounts != nil {
		t.Fatalf("empty mounts = %+v", mounts)
	}
	st2 := state.InstanceState{Display: st}
	if displayResultFromState(st2.Display).Port != 5902 {
		t.Fatal("display lost through state round-trip")
	}
}
