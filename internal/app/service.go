package app

const Version = "0.0.0-dev"

type Service struct {
	version string
}

func NewService() *Service {
	return &Service{version: Version}
}

func (s *Service) Version() string {
	if s == nil || s.version == "" {
		return Version
	}
	return s.version
}
