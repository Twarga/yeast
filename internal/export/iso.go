// Package export contains pluggable bootable-ISO builders. A builder
// emits the remote command sequence that turns a running (cloned) guest
// into a live .iso — executed over SSH by the app layer inside an
// ephemeral VM so the host needs no root or toolchain.
package export

import (
	"fmt"
	"strings"
	"time"
)

// BuildSpec carries everything a builder needs to emit its commands.
type BuildSpec struct {
	// WorkDir is the guest-side staging directory (deleted first).
	WorkDir string
	// Output is the guest-side .iso path the last step produces.
	Output string
	// SkelUser copies /home/<user> dotfiles into /etc/skel when set.
	SkelUser string
	// OSRelease overrides /etc/os-release PRETTY_NAME branding when set.
	OSRelease string
}

// Step is one remote command in the build pipeline.
type Step struct {
	Name    string
	Command string
	Timeout time.Duration
}

// Builder names a concrete ISO pipeline (debian-live today; archiso/mkosi
// are natural future implementations of this interface).
type Builder interface {
	Name() string
	// Steps emits the ordered remote commands; all run under a shell with
	// sudo available. The final step must produce BuildSpec.Output.
	Steps(spec BuildSpec) []Step
}

// BuilderByName resolves a builder name to an implementation.
func BuilderByName(name string) (Builder, error) {
	switch name {
	case "", "debian-live":
		return debianLiveBuilder{}, nil
	default:
		return nil, fmt.Errorf("unknown iso builder %q (available: debian-live)", name)
	}
}

// debianLiveBuilder produces a hybrid BIOS/UEFI Debian-family live ISO:
// squashfs rootfs + the guest's own kernel/initrd + grub-mkrescue.
type debianLiveBuilder struct{}

func (debianLiveBuilder) Name() string { return "debian-live" }

func (b debianLiveBuilder) Steps(spec BuildSpec) []Step {
	w := spec.WorkDir
	live := w + "/iso/live"
	grub := w + "/iso/boot/grub"
	steps := []Step{
		{"tools", "sudo apt-get update -qq && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq " +
			"live-boot squashfs-tools xorriso grub-pc-bin grub-efi-amd64-bin mtools >/dev/null && " +
			"sudo update-initramfs -u >/dev/null", 10 * time.Minute},
		{"stage", fmt.Sprintf("sudo rm -rf %[1]s && mkdir -p %[2]s %[3]s %[4]s/live",
			w, live, grub, w+"/iso"), time.Minute},
	}
	if spec.SkelUser != "" {
		steps = append(steps, Step{"skel",
			fmt.Sprintf("sudo mkdir -p /etc/skel && sudo cp -a /home/%s/. /etc/skel/ 2>/dev/null || true",
				spec.SkelUser), time.Minute})
	}
	if spec.OSRelease != "" {
		steps = append(steps, Step{"branding",
			fmt.Sprintf("sudo sed -i 's/^PRETTY_NAME=.*/PRETTY_NAME=\"%s\"/' /etc/os-release",
				strings.ReplaceAll(spec.OSRelease, `"`, `\"`)), time.Minute})
	}
	relWork := strings.TrimPrefix(w, "/")
	steps = append(steps,
		Step{"squashfs", fmt.Sprintf(
			"sudo mksquashfs / %s/filesystem.squashfs -comp xz -wildcards "+
				"-e proc sys dev tmp run mnt media '%s' 'var/tmp/*' 'tmp/*' 'var/lib/cloud/*' 2>/dev/null || "+
				"sudo mksquashfs / %s/filesystem.squashfs -comp xz -e proc sys dev tmp run mnt media '%s' var/tmp",
			live, relWork, live, relWork), 30 * time.Minute},
		Step{"kernel", fmt.Sprintf(
			"VMLINUZ=$(ls -t /boot/vmlinuz-* 2>/dev/null | head -n1); "+
				"INITRD=$(ls -t /boot/initrd.img-* 2>/dev/null | head -n1); "+
				"if [ -n \"$VMLINUZ\" ] && [ -n \"$INITRD\" ]; then "+
				"sudo cp \"$VMLINUZ\" %[1]s/vmlinuz && sudo cp \"$INITRD\" %[1]s/initrd; "+
				"else sudo cp /boot/vmlinuz %[1]s/vmlinuz && sudo cp /boot/initrd.img %[1]s/initrd; fi",
			live), 2 * time.Minute},
		Step{"grub-config", fmt.Sprintf(
			"sudo tee %s/grub.cfg >/dev/null <<'EOF'\n"+
				"set timeout=5\nset default=0\n\n"+
				"menuentry \"Yeast Live\" {\n"+
				"  linux /live/vmlinuz boot=live quiet splash ---\n"+
				"  initrd /live/initrd\n}\nEOF", grub), time.Minute},
		Step{"mkrescue", fmt.Sprintf(
			"sudo grub-mkrescue -o %s %s/iso 2>&1 | tail -5",
			spec.Output, w), 15 * time.Minute},
		Step{"verify", fmt.Sprintf("test -s %s && echo ISO-OK", spec.Output), time.Minute},
	)
	return steps
}
