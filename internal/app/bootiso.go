package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	rtm "github.com/Twarga/yeast/internal/runtime"
)

// BootISOOptions drives `yeast boot-iso <path.iso>` — boot any local ISO
// in a clean Yeast VM (CD-ROM boot, UEFI + VNC + QMP screenshot optional)
// for automated ISO boot-testing.
type BootISOOptions struct {
	ISOPath    string
	UEFI       bool
	Headless   bool   // no display
	DiskSize   string // optional empty scratch disk ("20G")
	Wait       time.Duration
	Screenshot string // PNG path — taken after Wait via QMP
	Remove     bool   // --rm: tear the VM down when the command returns
}

type BootISOResult struct {
	ISOPath    string `json:"iso_path"`
	Name       string `json:"name"`
	PID        int    `json:"pid"`
	RuntimeDir string `json:"runtime_dir"`
	VNCUrl     string `json:"vnc_url,omitempty"`
	Screenshot string `json:"screenshot,omitempty"`
	Removed    bool   `json:"removed"`
}

// BootISO starts QEMU straight from a CD-ROM ISO. With --rm it waits,
// optionally screenshots via QMP, then destroys the VM — the boot-test
// loop OS builders need.
func (s *Service) BootISO(ctx context.Context, options BootISOOptions) (BootISOResult, error) {
	isoAbs, err := filepath.Abs(options.ISOPath)
	if err != nil {
		return BootISOResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if _, err := os.Stat(isoAbs); err != nil {
		return BootISOResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("iso not found: %s", options.ISOPath), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return BootISOResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	name := fmt.Sprintf("bootiso-%d", time.Now().Unix())
	runDir := filepath.Join(yeastHome, "boot-iso", name)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return BootISOResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	started := false
	cleanup := func() {
		_ = s.runtime.Destroy(context.Background(), rtm.RuntimeInstance{Name: name, RuntimeDir: runDir})
		_ = os.RemoveAll(runDir)
	}
	// Always clean up on failure; on success clean only when --rm asked.
	defer func() {
		if !started || options.Remove {
			cleanup()
		}
	}()

	sshPort, err := freeLoopbackPort()
	if err != nil {
		return BootISOResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	plan := rtm.MachinePlan{
		Name:        name,
		RuntimeDir:  runDir,
		LogPath:     filepath.Join(runDir, "vm.log"),
		MemoryMiB:   2048,
		CPUs:        2,
		BootISOPath: isoAbs,
		Networks: rtm.NetworkPlan{Management: rtm.ManagementNetworkPlan{
			SSHHost: "127.0.0.1", SSHPort: sshPort,
			InterfaceName: defaultManagementNIC, MACAddress: deriveMACAddress("bootiso", name, "management"),
		}},
	}
	if options.UEFI {
		plan.Firmware = "uefi"
	}
	if options.DiskSize != "" {
		exporter, ok := s.runtime.(rtm.DiskExporter)
		if !ok {
			return BootISOResult{}, WrapError(ErrorCodePrecondition, "runtime cannot create scratch disks", nil)
		}
		diskPath := filepath.Join(runDir, "disk.qcow2")
		if err := exporter.CreateBlankDisk(ctx, diskPath, options.DiskSize); err != nil {
			return BootISOResult{}, WrapError(ErrorCodeRuntime, fmt.Sprintf("create scratch disk: %v", err), err)
		}
		plan.Disk = rtm.DiskPlan{DiskPath: diskPath}
	} else {
		// QEMU requires some drive for the disk arg; a 1M scratch is fine.
		plan.Disk = rtm.DiskPlan{DiskPath: filepath.Join(runDir, "disk.qcow2")}
		if exporter, ok := s.runtime.(rtm.DiskExporter); ok {
			_ = exporter.CreateBlankDisk(ctx, plan.Disk.DiskPath, "64M")
		}
	}
	if !options.Headless {
		vncPort, err := freeLoopbackPort()
		if err != nil {
			return BootISOResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		plan.Display = &rtm.DisplayPlan{
			Type: "vnc", Host: "127.0.0.1", Port: vncPort,
		}
	}

	instance, err := s.runtime.Start(ctx, plan)
	if err != nil {
		return BootISOResult{}, WrapError(ErrorCodeRuntime, fmt.Sprintf("boot iso: %v", err), err)
	}
	started = true

	if options.Wait > 0 {
		select {
		case <-ctx.Done():
			return BootISOResult{}, WrapError(ErrorCodeRuntime, ctx.Err().Error(), ctx.Err())
		case <-time.After(options.Wait):
		}
	}
	var shotPath string
	if options.Screenshot != "" {
		controller, ok := s.runtime.(rtm.DisplayController)
		if !ok {
			return BootISOResult{}, WrapError(ErrorCodePrecondition, "runtime does not support screenshots", nil)
		}
		shotPath, _ = filepath.Abs(options.Screenshot)
		if err := controller.Screenshot(ctx, instance, shotPath); err != nil {
			return BootISOResult{}, WrapError(ErrorCodeRuntime, fmt.Sprintf("screenshot: %v", err), err)
		}
	}

	res := BootISOResult{
		ISOPath:    isoAbs,
		Name:       name,
		PID:        instance.PID,
		RuntimeDir: runDir,
		Screenshot: shotPath,
		Removed:    options.Remove,
	}
	if instance.Display != nil {
		res.VNCUrl = instance.Display.URL
	}
	return res, nil
}
