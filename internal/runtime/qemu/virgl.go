package qemu

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// virglSupportFn verifies virgl 3D prerequisites; injectable for tests.
var virglSupportFn = checkVirglSupport

func checkVirglSupport(renderNode string) error {
	if strings.TrimSpace(renderNode) == "" {
		renderNode = "/dev/dri/renderD128"
	}
	if _, err := os.Stat(renderNode); err != nil {
		return fmt.Errorf("display.gpu=virgl requires a DRM render node at %s: %v (check /dev/dri permissions and your GPU driver, or use display.gpu: virtio-gpu)", renderNode, err)
	}
	out, err := exec.Command(qemuSystemBinary, "-display", "help").CombinedOutput()
	if err != nil {
		return fmt.Errorf("probe qemu display backends: %w", err)
	}
	if !strings.Contains(string(out), "egl-headless") {
		return fmt.Errorf("display.gpu=virgl requires QEMU with egl-headless support; upgrade qemu-system-x86_64 or use display.gpu: virtio-gpu")
	}
	return nil
}
