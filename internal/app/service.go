package app

import (
	"context"
	"net/http"
	"time"
	"yeast/internal/guest"
	"yeast/internal/images"
	"yeast/internal/project"
	"yeast/internal/provision/cloudinit"
	rtm "yeast/internal/runtime"
	"yeast/internal/runtime/qemu"
)

var Version = "0.0.0-dev"

type Service struct {
	version          string
	resolveYeastHome func() (string, error)
	downloadImage    func(image images.TrustedImage, destination string, options images.DownloadOptions) error
	discoverSSHKey   func() (string, error)
	renderUserData   func(input cloudinit.UserDataInput) (string, error)
	renderMetaData   func(input cloudinit.MetaDataInput) (string, error)
	createSeedISO    func(ctx context.Context, input cloudinit.SeedInput) (cloudinit.SeedResult, error)
	waitForTCP       func(ctx context.Context, options guest.ReadinessOptions) error
	sshAddress       func(host string, port int) (string, error)
	runSSH           func(ctx context.Context, args []string) error
	runtime          rtm.Runtime
	httpClient       *http.Client
}

func NewService() *Service {
	return &Service{
		version:          Version,
		resolveYeastHome: project.DefaultYeastHome,
		downloadImage:    images.Download,
		discoverSSHKey:   cloudinit.DiscoverAuthorizedKey,
		renderUserData:   cloudinit.RenderUserData,
		renderMetaData:   cloudinit.RenderMetaData,
		createSeedISO:    cloudinit.CreateSeedISO,
		waitForTCP:       guest.WaitForTCP,
		sshAddress:       guest.SSHAddress,
		runSSH:           guest.RunSSH,
		runtime:          qemu.NewRuntime(),
		httpClient:       http.DefaultClient,
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
