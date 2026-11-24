package export

import (
	"strings"
	"testing"
)

func TestDebianLiveBuilderSteps(t *testing.T) {
	b, err := BuilderByName("debian-live")
	if err != nil {
		t.Fatalf("BuilderByName: %v", err)
	}
	if b.Name() != "debian-live" {
		t.Fatalf("name: %s", b.Name())
	}
	steps := b.Steps(BuildSpec{WorkDir: "/tmp/yeast-iso", Output: "/tmp/yeast-live.iso"})
	if len(steps) < 6 {
		t.Fatalf("expected full pipeline, got %d steps", len(steps))
	}
	var joined string
	for _, s := range steps {
		joined += s.Command + "\n"
		if s.Timeout <= 0 {
			t.Fatalf("step %s missing timeout", s.Name)
		}
	}
	for _, want := range []string{"mksquashfs", "filesystem.squashfs", "vmlinuz", "initrd", "grub.cfg", "grub-mkrescue", "/tmp/yeast-live.iso", "xorriso", "apt-get install"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("pipeline missing %q", want)
		}
	}
}

func TestDebianLiveSkelAndBranding(t *testing.T) {
	b, _ := BuilderByName("")
	steps := b.Steps(BuildSpec{WorkDir: "/w", Output: "/o.iso", SkelUser: "yeast", OSRelease: "YeastOS 2.0"})
	var names []string
	for _, s := range steps {
		names = append(names, s.Name)
	}
	if !contains(names, "skel") || !contains(names, "branding") {
		t.Fatalf("skel/branding steps missing: %v", names)
	}
	var skelCmd string
	for _, s := range steps {
		if s.Name == "skel" {
			skelCmd = s.Command
		}
	}
	if !strings.Contains(skelCmd, "/home/yeast") || !strings.Contains(skelCmd, "/etc/skel") {
		t.Fatalf("skel step wrong: %s", skelCmd)
	}
}

func TestBuilderByNameUnknown(t *testing.T) {
	if _, err := BuilderByName("archiso"); err == nil {
		t.Fatal("expected error for unregistered builder")
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
