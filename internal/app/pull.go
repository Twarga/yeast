package app

import (
	"errors"
	"fmt"
	"yeast/internal/images"
)

var ErrUnsupportedImage = errors.New("unsupported image")

type PullOptions struct {
	ImageName string
	List      bool
}

type PullResult struct {
	List        bool
	Images      []string
	ImageName   string
	ImagePath   string
	ManifestURL string
	SHA256      string
}

func (s *Service) Pull(options PullOptions) (PullResult, error) {
	if options.List {
		return PullResult{
			List:   true,
			Images: images.SupportedImages(),
		}, nil
	}

	image, ok := images.Lookup(options.ImageName)
	if !ok {
		return PullResult{}, fmt.Errorf("%w: %s", ErrUnsupportedImage, options.ImageName)
	}

	cacheRoot, err := s.resolveYeastHome()
	if err != nil {
		return PullResult{}, err
	}
	cachePaths, err := images.ResolveCachePaths(cacheRoot+"/cache/images", image.Name)
	if err != nil {
		return PullResult{}, err
	}

	if err := s.downloadImage(image, cachePaths.ImageFile, s.downloadOptions()); err != nil {
		return PullResult{}, err
	}

	return PullResult{
		ImageName:   image.Name,
		ImagePath:   cachePaths.ImageFile,
		ManifestURL: image.URL,
		SHA256:      image.SHA256,
	}, nil
}
