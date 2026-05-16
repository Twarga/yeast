package app

import (
	"net/http"
	"time"
	"yeast/internal/images"
	"yeast/internal/project"
	"yeast/internal/provision/cloudinit"
)

const Version = "0.0.0-dev"

type Service struct {
	version          string
	resolveYeastHome func() (string, error)
	downloadImage    func(image images.TrustedImage, destination string, options images.DownloadOptions) error
	discoverSSHKey   func() (string, error)
	httpClient       *http.Client
}

func NewService() *Service {
	return &Service{
		version:          Version,
		resolveYeastHome: project.DefaultYeastHome,
		downloadImage:    images.Download,
		discoverSSHKey:   cloudinit.DiscoverAuthorizedKey,
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
