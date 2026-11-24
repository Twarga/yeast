package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Twarga/yeast/internal/config"
	"github.com/Twarga/yeast/internal/provision/cloudinit"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
)

const (
	displayPortAutoFirst = 5901
	displayPortAutoLast  = 5999
	defaultRenderNode    = "/dev/dri/renderD128"
)

// DisplayResult is the user-facing view of a running instance's display.
type DisplayResult struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	URL      string `json:"url"`
	GPU      string `json:"gpu,omitempty"`
	Firmware string `json:"firmware,omitempty"`
}

// findVirtiofsd locates the virtiofs daemon; injectable for tests.
var findVirtiofsd = func() (string, error) {
	for _, candidate := range []string{
		"/usr/libexec/virtiofsd",
		"/usr/lib/qemu/virtiofsd",
		"/usr/libexec/virtiofsd-rs",
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return exec.LookPath("virtiofsd")
}

// buildDisplayPlan converts display config into a runtime plan and reserves
// the VNC port (explicit or auto-allocated from 5901..5999) in allocated binds.
func buildDisplayPlan(instance config.Instance, portAvailable func(int) bool, allocated map[string]string) (*rtm.DisplayPlan, error) {
	if instance.Display == nil || !instance.Display.Enabled {
		return nil, nil
	}
	display := instance.Display

	width, height := 0, 0
	if display.Resolution != "" {
		if _, err := fmt.Sscanf(display.Resolution, "%dx%d", &width, &height); err != nil {
			return nil, fmt.Errorf("instance %s display.resolution %q: %v", instance.Name, display.Resolution, err)
		}
	}

	host := display.Host
	if host == "" {
		host = config.DefaultDisplayHost
	}
	port := display.Port
	if port == 0 {
		port = autoDisplayPort(host, portAvailable, allocated)
		if port == 0 {
			return nil, fmt.Errorf("instance %s could not auto-allocate a display port in %d..%d; set display.port explicitly", instance.Name, displayPortAutoFirst, displayPortAutoLast)
		}
	} else {
		key := hostPortKey(host, port)
		if previous, exists := allocated[key]; exists {
			return nil, fmt.Errorf("instance %s display port %s:%d conflicts with %s", instance.Name, host, port, previous)
		}
		if !portAvailable(port) {
			return nil, fmt.Errorf("instance %s display port %s:%d is already in use on the host", instance.Name, host, port)
		}
	}
	allocated[hostPortKey(host, port)] = instance.Name + " display"

	renderNode := ""
	if display.GPU == "virgl" {
		renderNode = defaultRenderNode
	}
	return &rtm.DisplayPlan{
		Type:       display.Type,
		Host:       host,
		Port:       port,
		GPU:        display.GPU,
		Width:      width,
		Height:     height,
		RenderNode: renderNode,
	}, nil
}

func autoDisplayPort(host string, portAvailable func(int) bool, allocated map[string]string) int {
	for port := displayPortAutoFirst; port <= displayPortAutoLast; port++ {
		if _, used := allocated[hostPortKey(host, port)]; used {
			continue
		}
		if portAvailable == nil || portAvailable(port) {
			return port
		}
	}
	return 0
}

// resolveMountPlans turns config mounts into runtime plans: sources are
// resolved against the project root, must exist, and the driver is resolved
// (auto prefers virtiofsd when installed, else 9p).
func resolveMountPlans(projectRoot string, instance config.Instance) ([]rtm.MountPlan, error) {
	if len(instance.Mounts) == 0 {
		return nil, nil
	}
	plans := make([]rtm.MountPlan, 0, len(instance.Mounts))
	for i, mount := range instance.Mounts {
		source := strings.TrimSpace(mount.Source)
		if !filepath.IsAbs(source) {
			source = filepath.Join(projectRoot, source)
		}
		source = filepath.Clean(source)
		info, err := os.Stat(source)
		if err != nil {
			return nil, fmt.Errorf("instance %s mounts[%d] source %s: %v", instance.Name, i, mount.Source, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("instance %s mounts[%d] source %s is not a directory", instance.Name, i, mount.Source)
		}
		driver, daemonPath, err := resolveMountDriver(mount.Driver)
		if err != nil {
			return nil, fmt.Errorf("instance %s mounts[%d]: %v", instance.Name, i, err)
		}
		plans = append(plans, rtm.MountPlan{
			Source:     source,
			Target:     mount.Target,
			ReadOnly:   mount.ReadOnly,
			Driver:     driver,
			Tag:        fmt.Sprintf("yeast_mnt_%d", i),
			DaemonPath: daemonPath,
		})
	}
	return plans, nil
}

func resolveMountDriver(requested string) (driver, daemonPath string, err error) {
	switch requested {
	case "9p":
		return "9p", "", nil
	case "virtiofs", "", "auto":
		if path, lookErr := findVirtiofsd(); lookErr == nil {
			return "virtiofs", path, nil
		}
		if requested == "virtiofs" {
			return "", "", fmt.Errorf("driver virtiofs requested but virtiofsd was not found (install qemu virtiofsd or use driver: 9p)")
		}
		return "9p", "", nil
	default:
		return "", "", fmt.Errorf("invalid driver %q", requested)
	}
}

func buildDisplayState(plan *rtm.DisplayPlan, firmware string) *state.DisplayState {
	if plan == nil {
		return nil
	}
	return &state.DisplayState{
		Type:     plan.Type,
		Host:     plan.Host,
		Port:     plan.Port,
		URL:      fmt.Sprintf("vnc://%s:%d", plan.Host, plan.Port),
		GPU:      plan.GPU,
		Firmware: firmware,
	}
}

func buildMountStates(plans []rtm.MountPlan) []state.MountState {
	if len(plans) == 0 {
		return nil
	}
	result := make([]state.MountState, 0, len(plans))
	for _, plan := range plans {
		result = append(result, state.MountState{
			Source:   plan.Source,
			Target:   plan.Target,
			Driver:   plan.Driver,
			ReadOnly: plan.ReadOnly,
		})
	}
	return result
}

func displayResultFromState(display *state.DisplayState) *DisplayResult {
	if display == nil {
		return nil
	}
	return &DisplayResult{
		Type:     display.Type,
		Host:     display.Host,
		Port:     display.Port,
		URL:      display.URL,
		GPU:      display.GPU,
		Firmware: display.Firmware,
	}
}

func mountSpecs(plans []rtm.MountPlan) []cloudinit.MountSpec {
	if len(plans) == 0 {
		return nil
	}
	specs := make([]cloudinit.MountSpec, 0, len(plans))
	for _, plan := range plans {
		specs = append(specs, cloudinit.MountSpec{
			Target:   plan.Target,
			Driver:   plan.Driver,
			Tag:      plan.Tag,
			ReadOnly: plan.ReadOnly,
		})
	}
	return specs
}

// egressPlan maps config egress to the runtime plan. nil config or empty
// mode means open NAT (no runtime plan needed).
func egressPlan(eg *config.EgressConfig) *rtm.EgressPlan {
	if eg == nil {
		return nil
	}
	switch strings.TrimSpace(eg.Mode) {
	case "offline", "allowlist":
		return &rtm.EgressPlan{Mode: strings.TrimSpace(eg.Mode)}
	default:
		return nil
	}
}

// writeEgressAllowlist persists the allowlist domains to the instance
// runtime dir; the egress-proxy sidecar reads it at spawn.
func writeEgressAllowlist(runtimeDir string, eg *config.EgressConfig) error {
	if eg == nil || strings.TrimSpace(eg.Mode) != "allowlist" {
		return nil
	}
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return fmt.Errorf("create runtime dir for egress allowlist: %w", err)
	}
	return os.WriteFile(filepath.Join(runtimeDir, "egress.allowlist"),
		[]byte(strings.Join(eg.Allowlist, "\n")+"\n"), 0o600)
}
