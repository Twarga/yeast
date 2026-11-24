package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Twarga/yeast/internal/config"
	"github.com/Twarga/yeast/internal/guest"
	"github.com/Twarga/yeast/internal/images"
	"github.com/Twarga/yeast/internal/provision/cloudinit"
	provssh "github.com/Twarga/yeast/internal/provision/ssh"
	rtm "github.com/Twarga/yeast/internal/runtime"
)

// RunOptions drives `yeast run --rm <image> -- <cmd>`: boot a cached image
// into an ephemeral, offline-by-default VM, run a command, destroy.
type RunOptions struct {
	Image     string
	Command   []string
	Env       []string // KEY=VAL
	MemoryMiB int
	CPUs      int
	Timeout   time.Duration
	Stdin     []byte
	// Online opts the ephemeral VM into unrestricted NAT (default is
	// egress.mode=offline — guests cannot reach the network).
	Online bool
}

type RunResult struct {
	Image      string `json:"image"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Ephemeral  bool   `json:"ephemeral"`
	Offline    bool   `json:"offline"`
}

// Run boots an ephemeral VM from a cached image, executes Command over SSH,
// and tears everything down — even on error or signal-cancelled ctx.
func (s *Service) Run(ctx context.Context, options RunOptions) (RunResult, error) {
	started := time.Now()
	if strings.TrimSpace(options.Image) == "" {
		return RunResult{}, WrapError(ErrorCodeInvalidArgument, "image name is required", nil)
	}
	if len(options.Command) == 0 {
		return RunResult{}, WrapError(ErrorCodeInvalidArgument, "run command is required (use -- <cmd>)", nil)
	}

	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	cachePaths, err := images.ResolveCachePaths(filepath.Join(yeastHome, "images"), options.Image)
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("resolve image %s: %v", options.Image, err), err)
	}
	if _, err := os.Stat(cachePaths.ImageFile); err != nil {
		return RunResult{}, WrapError(ErrorCodeNotFound,
			fmt.Sprintf("image %q is not cached — run `yeast pull %s` first", options.Image, options.Image), err)
	}

	userKey, err := s.discoverSSHKey()
	if err != nil {
		return RunResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
	}

	runDir := filepath.Join(yeastHome, "run", fmt.Sprintf("run-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	// --rm semantics: always tear down, whatever happens next.
	defer func() {
		_ = s.runtime.Destroy(context.Background(), rtm.RuntimeInstance{
			Name: "yeast-run", RuntimeDir: runDir})
		_ = os.RemoveAll(runDir)
	}()

	name := "yeast-run"
	memory := options.MemoryMiB
	if memory <= 0 {
		memory = 1024
	}
	cpus := options.CPUs
	if cpus <= 0 {
		cpus = 2
	}
	mac := deriveMACAddress("run", name, "management")

	userData, err := s.renderUserData(cloudinit.UserDataInput{
		Hostname:      name,
		Instance:      config.Instance{Name: name, User: config.DefaultUser, Sudo: config.DefaultSudo},
		AuthorizedKey: userKey,
	})
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	metaData, err := s.renderMetaData(cloudinit.MetaDataInput{Hostname: name})
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	networkConfig, err := s.renderNetworkConfig(cloudinit.NetworkConfigInput{
		ManagementInterfaceName: defaultManagementNIC,
		ManagementMACAddress:    mac,
	})
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	seed, err := s.createSeedISO(ctx, cloudinit.SeedInput{
		InstanceName: name, RuntimeDir: runDir,
		UserData: userData, MetaData: metaData, NetworkConfig: networkConfig,
	})
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	sshPort, err := freeLoopbackPort()
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	egress := &rtm.EgressPlan{Mode: "offline"}
	if options.Online {
		egress = nil
	}
	plan := rtm.MachinePlan{
		Name:          name,
		RuntimeDir:    runDir,
		LogPath:       filepath.Join(runDir, "vm.log"),
		MemoryMiB:     memory,
		CPUs:          cpus,
		SeedImagePath: seed.ISOPath,
		Egress:        egress,
		Disk: rtm.DiskPlan{
			BaseImagePath: cachePaths.ImageFile,
			DiskPath:      filepath.Join(runDir, "disk.qcow2"),
		},
		Networks: rtm.NetworkPlan{
			Management: rtm.ManagementNetworkPlan{
				SSHHost:       "127.0.0.1",
				SSHPort:       sshPort,
				InterfaceName: defaultManagementNIC,
				MACAddress:    mac,
			},
		},
	}
	if _, err := s.runtime.PrepareDisk(ctx, plan); err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	instance, err := s.runtime.Start(ctx, plan)
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeRuntime, fmt.Sprintf("start ephemeral VM: %v", err), err)
	}
	defer func() {
		_ = s.runtime.Stop(context.Background(), instance, 5*time.Second)
	}()

	address, err := s.sshAddress("127.0.0.1", sshPort)
	if err != nil {
		return RunResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := s.waitForTCP(ctx, guest.ReadinessOptions{Address: address, Timeout: defaultReadinessTimeout}); err != nil {
		return RunResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("wait for ssh readiness: %v", err), err)
	}

	transport := s.provisionTransport
	if transport == nil {
		transport = provssh.NewLocalTransport()
	}
	remoteCmd := buildRunRemoteCommand(options)
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	runRes, runErr := transport.Run(ctx, provssh.RunRequest{
		User:    "yeast",
		Host:    "127.0.0.1",
		Port:    sshPort,
		Command: remoteCmd,
		Timeout: timeout + 30*time.Second,
		Stdin:   options.Stdin,
	})
	if runErr != nil && runRes.ExitCode == 0 && !errors.Is(runErr, context.DeadlineExceeded) {
		return RunResult{}, WrapError(ErrorCodeRuntime, fmt.Sprintf("run command: %v", runErr), runErr)
	}
	return RunResult{
		Image:      options.Image,
		Command:    commandString(options.Command),
		ExitCode:   runRes.ExitCode,
		Stdout:     runRes.Stdout,
		Stderr:     runRes.Stderr,
		DurationMs: time.Since(started).Milliseconds(),
		Ephemeral:  true,
		Offline:    !options.Online,
	}, nil
}

func buildRunRemoteCommand(options RunOptions) string {
	cmd := commandString(options.Command)
	if len(options.Env) > 0 {
		exports := make([]string, 0, len(options.Env))
		for _, kv := range options.Env {
			exports = append(exports, kv)
		}
		cmd = "env " + strings.Join(exports, " ") + " " + cmd
	}
	return cmd
}

func freeLoopbackPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}
