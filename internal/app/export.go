package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Twarga/yeast/internal/config"
	"github.com/Twarga/yeast/internal/export"
	provssh "github.com/Twarga/yeast/internal/provision/ssh"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
)

// ExportOptions drives `yeast export <instance> --format <fmt> -o <path>`.
type ExportOptions struct {
	GuestTargetOptions
	Format   string // qcow2 | rootfs | iso
	Output   string
	Sysprep  bool
	SkelUser string // iso: copy this user's dotfiles into /etc/skel
	Builder  string // iso: builder name (default "debian-live")
}

type ExportResult struct {
	ProjectID string `json:"project_id"`
	Instance  string `json:"instance"`
	Format    string `json:"format"`
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
	Builder   string `json:"builder,omitempty"`
}

// sysprepScript generalizes a cloned disk for sharing: clears machine-id,
// host SSH keys, cloud-init state, package caches, then trims free space.
const sysprepScript = `set -e
sudo truncate -s0 /etc/machine-id
sudo rm -f /etc/ssh/ssh_host_*
sudo rm -rf /var/lib/cloud
(sudo apt-get clean -y 2>/dev/null || sudo pacman -Scc --noconfirm 2>/dev/null) || true
sudo fstrim -av 2>/dev/null || true
sudo sync
`

// rootfsExcludes are mounted/virtual filesystems that must not enter the
// exported root filesystem tarball.
var rootfsExcludes = []string{"proc", "sys", "dev", "tmp", "run", "mnt", "media"}

func (s *Service) Export(ctx context.Context, options ExportOptions) (result ExportResult, err error) {
	started := time.Now()
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return ExportResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	defer func() {
		s.auditCall(ctx, absoluteRoot, "export", map[string]any{
			"instance": options.Target, "format": options.Format, "output": options.Output,
		}, started, err)
	}()

	switch options.Format {
	case "qcow2", "rootfs", "iso":
	default:
		return ExportResult{}, WrapError(ErrorCodeInvalidArgument,
			fmt.Sprintf("unsupported export format %q (expected qcow2, rootfs, or iso)", options.Format), nil)
	}
	if strings.TrimSpace(options.Target) == "" {
		return ExportResult{}, WrapError(ErrorCodeInvalidArgument, "export target instance is required", nil)
	}
	if strings.TrimSpace(options.Output) == "" {
		return ExportResult{}, WrapError(ErrorCodeInvalidArgument, "export output path (-o) is required", nil)
	}

	metadata, currentState, err := s.loadProjectStateForGuestControl(ctx, absoluteRoot)
	if err != nil {
		return ExportResult{}, err
	}
	inst, ok := currentState.Instances[options.Target]
	if !ok {
		return ExportResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}

	diskPath := inst.ActiveDisk
	if diskPath == "" {
		diskPath = filepath.Join(inst.RuntimeDir, "disk.qcow2")
	}
	running := inst.PID > 0
	sshUser := inst.User
	if sshUser == "" {
		sshUser = config.DefaultUser
	}

	result.ProjectID = metadata.ID
	result.Instance = options.Target
	result.Format = options.Format
	result.Path = options.Output

	switch options.Format {
	case "qcow2":
		err = s.exportQCOW2(ctx, options.Target, inst, diskPath, running, options)
	case "rootfs":
		err = s.exportRootfs(ctx, options.Target, inst, diskPath, running, sshUser, options)
	case "iso":
		var builder string
		builder, err = s.exportISO(ctx, options.Target, inst, diskPath, sshUser, options)
		result.Builder = builder
	}
	if err != nil {
		return ExportResult{}, err
	}

	fi, err := os.Stat(options.Output)
	if err != nil {
		return ExportResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("export output missing: %v", err), err)
	}
	result.Bytes = fi.Size()
	sum, err := fileSHA256(options.Output)
	if err != nil {
		return ExportResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("checksum export: %v", err), err)
	}
	result.SHA256 = sum
	return result, nil
}

// exportQCOW2 flattens the instance disk. --sysprep clones → boots an
// ephemeral VM → generalizes → exports the clone; never touches the live
// disk. Requires the instance stopped (convert needs exclusive access).
func (s *Service) exportQCOW2(ctx context.Context, name string, inst state.InstanceState, diskPath string, running bool, options ExportOptions) error {
	exporter, ok := s.runtime.(rtm.DiskExporter)
	if !ok {
		return WrapError(ErrorCodePrecondition, "runtime does not support disk export", nil)
	}
	if running {
		return WrapError(ErrorCodePrecondition,
			fmt.Sprintf("instance %q is running — stop it first (qcow2 export needs exclusive disk access)", name), nil)
	}

	src := diskPath
	if options.Sysprep {
		clone, cleanup, err := s.sysprepClone(ctx, diskPath)
		if err != nil {
			return err
		}
		defer cleanup()
		src = clone
	}
	outAbs, err := filepath.Abs(options.Output)
	if err != nil {
		return WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := exporter.ExportDisk(ctx, src, outAbs); err != nil {
		return WrapError(ErrorCodeRuntime, fmt.Sprintf("export qcow2: %v", err), err)
	}
	return nil
}

// exportRootfs streams a guest-side tar (perms+xattrs preserved) through
// the host's zstd into out.tar.zst. Running instance → direct SSH; stopped
// → an ephemeral clone boot so the export is still non-mutating.
func (s *Service) exportRootfs(ctx context.Context, name string, inst state.InstanceState, diskPath string, running bool, sshUser string, options ExportOptions) error {
	port := inst.SSHPort
	_ = name // instance name available for messages
	if !running || options.Sysprep {
		// Stopped instance → clone+ephemeral boot (non-mutating); running +
		// --sysprep is rejected because sysprep must see a consistent disk.
		if running {
			return WrapError(ErrorCodePrecondition,
				"--sysprep requires the instance stopped (it clones and boots a throwaway copy)", nil)
		}
		evm, cleanup, err := s.bootClone(ctx, name, diskPath, false)
		if err != nil {
			return err
		}
		defer cleanup()
		port = evm.SSHPort
		sshUser = config.DefaultUser
	}

	transport := s.provisionTransport
	if transport == nil {
		transport = provssh.NewLocalTransport()
	}
	streamer, ok := transport.(provssh.Streamer)
	if !ok {
		return WrapError(ErrorCodePrecondition, "transport does not support streaming export", nil)
	}

	excludes := make([]string, 0, len(rootfsExcludes))
	for _, e := range rootfsExcludes {
		excludes = append(excludes, "--exclude=./"+e)
	}
	remoteCmd := fmt.Sprintf("sudo tar --xattrs --acls --numeric-owner -C / %s -cf - .",
		strings.Join(excludes, " "))

	outAbs, err := filepath.Abs(options.Output)
	if err != nil {
		return WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := streamZstd(ctx, streamer, provssh.StreamRequest{
		User: sshUser, Host: "127.0.0.1", Port: port, Command: remoteCmd,
	}, outAbs); err != nil {
		return WrapError(ErrorCodeRuntime, fmt.Sprintf("export rootfs: %v", err), err)
	}
	return nil
}

// exportISO builds a bootable live ISO inside an ephemeral clone VM using
// a pluggable builder (debian-live), then downloads the artifact.
func (s *Service) exportISO(ctx context.Context, name string, inst state.InstanceState, diskPath, sshUser string, options ExportOptions) (string, error) {
	if inst.PID > 0 {
		return "", WrapError(ErrorCodePrecondition,
			fmt.Sprintf("instance %q is running — ISO builds run on a stopped instance's cloned disk", name), nil)
	}
	builderName := options.Builder
	if builderName == "" {
		builderName = "debian-live"
	}
	builder, err := export.BuilderByName(builderName)
	if err != nil {
		return "", WrapError(ErrorCodeInvalidArgument, err.Error(), err)
	}

	evm, cleanup, err := s.bootClone(ctx, name, diskPath, true)
	if err != nil {
		return "", err
	}
	defer cleanup()

	transport := s.provisionTransport
	if transport == nil {
		transport = provssh.NewLocalTransport()
	}
	if options.Sysprep {
		res, runErr := transport.Run(ctx, provssh.RunRequest{
			User: sshUser, Host: "127.0.0.1", Port: evm.SSHPort,
			Command: sysprepScript, Timeout: 3 * time.Minute,
		})
		if runErr != nil || res.ExitCode != 0 {
			return "", WrapError(ErrorCodeRuntime,
				fmt.Sprintf("iso sysprep failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr)), runErr)
		}
	}
	spec := export.BuildSpec{
		WorkDir:  "/var/tmp/yeast-iso-work",
		Output:   "/var/tmp/yeast-live.iso",
		SkelUser: options.SkelUser,
	}
	for i, step := range builder.Steps(spec) {
		res, runErr := transport.Run(ctx, provssh.RunRequest{
			User: sshUser, Host: "127.0.0.1", Port: evm.SSHPort,
			Command: step.Command, Timeout: step.Timeout,
		})
		if runErr != nil || res.ExitCode != 0 {
			return "", WrapError(ErrorCodeRuntime,
				fmt.Sprintf("iso build step %d (%s) failed (exit %d): %s",
					i, step.Name, res.ExitCode, strings.TrimSpace(res.Stderr)), runErr)
		}
	}
	outAbs, err := filepath.Abs(options.Output)
	if err != nil {
		return "", WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := os.MkdirAll(filepath.Dir(outAbs), 0o755); err != nil {
		return "", WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := transport.Download(ctx, provssh.DownloadRequest{
		User: config.DefaultUser, Host: "127.0.0.1", Port: evm.SSHPort,
		Source: spec.Output, Destination: outAbs,
		Timeout: 10 * time.Minute,
	}); err != nil {
		return "", WrapError(ErrorCodeRuntime, fmt.Sprintf("download iso: %v", err), err)
	}
	return builder.Name(), nil
}

// sysprepClone clones the disk, boots the clone ephemerally, runs the
// sysprep script, and returns the clone path plus a cleanup func.
func (s *Service) sysprepClone(ctx context.Context, diskPath string) (string, func(), error) {
	exporter, ok := s.runtime.(rtm.DiskExporter)
	if !ok {
		return "", nil, WrapError(ErrorCodePrecondition, "runtime does not support disk clone", nil)
	}
	dir, err := os.MkdirTemp("", "yeast-sysprep-*")
	if err != nil {
		return "", nil, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	clone, err := exporter.CloneDisk(ctx, diskPath, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, WrapError(ErrorCodeRuntime, fmt.Sprintf("clone disk: %v", err), err)
	}
	vmDir := filepath.Join(dir, "vm")
	evm, vmCleanup, err := s.ephemeralBoot(ctx, "yeast-sysprep", vmDir, clone, 0, 0, false)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, WrapError(ErrorCodeRuntime, fmt.Sprintf("boot sysprep clone: %v", err), err)
	}
	defer vmCleanup()
	_ = evm

	transport := s.provisionTransport
	if transport == nil {
		transport = provssh.NewLocalTransport()
	}
	res, runErr := transport.Run(ctx, provssh.RunRequest{
		User: config.DefaultUser, Host: "127.0.0.1", Port: evm.SSHPort,
		Command: sysprepScript, Timeout: 3 * time.Minute,
	})
	if runErr != nil || res.ExitCode != 0 {
		return "", nil, WrapError(ErrorCodeRuntime,
			fmt.Sprintf("sysprep failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr)), runErr)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	return clone, cleanup, nil
}

// bootClone clones the disk and boots it ephemerally — ISO builds and
// stopped-VM rootfs exports share this path.
func (s *Service) bootClone(ctx context.Context, name, diskPath string, online bool) (*ephemeralVM, func(), error) {
	exporter, ok := s.runtime.(rtm.DiskExporter)
	if !ok {
		return nil, nil, WrapError(ErrorCodePrecondition, "runtime does not support disk clone", nil)
	}
	dir, err := os.MkdirTemp("", "yeast-export-*")
	if err != nil {
		return nil, nil, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	clone, err := exporter.CloneDisk(ctx, diskPath, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, nil, WrapError(ErrorCodeRuntime, fmt.Sprintf("clone disk: %v", err), err)
	}
	evm, vmCleanup, err := s.ephemeralBoot(ctx, "yeast-"+name, filepath.Join(dir, "vm"), clone, 0, 0, online)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, nil, WrapError(ErrorCodeRuntime, fmt.Sprintf("boot export clone: %v", err), err)
	}
	cleanup := func() {
		vmCleanup()
		_ = os.RemoveAll(dir)
	}
	return evm, cleanup, nil
}

// streamZstd pipes a remote command's stdout through host `zstd` into dst.
func streamZstd(ctx context.Context, streamer provssh.Streamer, req provssh.StreamRequest, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	fail := func(err error) error {
		out.Close()
		os.Remove(tmp)
		return err
	}

	zstdPath, err := exec.LookPath("zstd")
	if err != nil {
		return fail(fmt.Errorf("zstd not found on host (install zstd for .tar.zst export)"))
	}
	zstdCmd := exec.CommandContext(ctx, zstdPath, "-19", "-T0")
	zstdCmd.Stdout = out
	pr, pw := io.Pipe()
	zstdCmd.Stdin = pr
	if err := zstdCmd.Start(); err != nil {
		return fail(err)
	}
	req.Stdout = pw
	streamErr := streamer.Stream(ctx, req)
	_ = pw.Close()
	waitErr := zstdCmd.Wait()
	out.Close()
	if streamErr != nil {
		return fail(streamErr)
	}
	if waitErr != nil {
		return fail(fmt.Errorf("zstd: %w", waitErr))
	}
	return os.Rename(tmp, dst)
}
