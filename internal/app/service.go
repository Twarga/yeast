package app

import (
	"context"
	"net/http"
	"time"
	"yeast/internal/guest"
	"yeast/internal/images"
	"yeast/internal/project"
	"yeast/internal/provision/cloudinit"
	provssh "yeast/internal/provision/ssh"
	rtm "yeast/internal/runtime"
	"yeast/internal/runtime/qemu"
)

var Version = "0.0.0-dev"

type Service struct {
	version                 string
	resolveYeastHome        func() (string, error)
	downloadImage           func(image images.TrustedImage, destination string, options images.DownloadOptions) error
	discoverSSHKey          func() (string, error)
	renderUserData          func(input cloudinit.UserDataInput) (string, error)
	renderMetaData          func(input cloudinit.MetaDataInput) (string, error)
	renderNetworkConfig     func(input cloudinit.NetworkConfigInput) (string, error)
	createSeedISO           func(ctx context.Context, input cloudinit.SeedInput) (cloudinit.SeedResult, error)
	waitForTCP              func(ctx context.Context, options guest.ReadinessOptions) error
	sshAddress              func(host string, port int) (string, error)
	runSSH                  func(ctx context.Context, args []string) error
	managementPortAvailable func(port int) bool
	sleep                   func(time.Duration)
	provisionTransport      provssh.Transport
	runtime                 rtm.Runtime
	httpClient              *http.Client
}

func NewService() *Service {
	return &Service{
		version:             Version,
		resolveYeastHome:    project.DefaultYeastHome,
		downloadImage:       images.Download,
		discoverSSHKey:      cloudinit.DiscoverAuthorizedKey,
		renderUserData:      cloudinit.RenderUserData,
		renderMetaData:      cloudinit.RenderMetaData,
		renderNetworkConfig: cloudinit.RenderNetworkConfig,
		createSeedISO:       cloudinit.CreateSeedISO,
		waitForTCP:          guest.WaitForTCP,
		sshAddress:          guest.SSHAddress,
		runSSH:              guest.RunSSH,
		sleep:               time.Sleep,
		runtime:             qemu.NewRuntime(),
		httpClient:          http.DefaultClient,
	}
}

func (s *Service) Version() string {
	if s == nil || s.version == "" {
		return Version
	}
	return s.version
}

func (s *Service) downloadOptions() images.DownloadOptions {
	return images.DownloadOptions{
		Timeout: 30 * time.Minute,
		Client:  s.httpClient,
	}
}

func (s *Service) waitForManagementPortRelease(ctx context.Context, host string, port int, timeout time.Duration) error {
	if port <= 0 {
		return nil
	}
	portAvailable := s.managementPortAvailable
	if portAvailable == nil {
		portAvailable = func(port int) bool {
			return managementPortAvailable(host, port)
		}
	}
	sleep := s.sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		if portAvailable(port) {
			return nil
		}
		select {
		case <-waitCtx.Done():
			return waitCtx.Err()
		default:
			sleep(100 * time.Millisecond)
		}
	}
}
