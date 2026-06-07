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
	Progress  images.DownloadProgressSink
}

type PullResult struct {
	List        bool     `json:"list,omitempty"`
	Images      []string `json:"images,omitempty"`
	ImageName   string   `json:"image_name,omitempty"`
	ImagePath   string   `json:"image_path,omitempty"`
	ManifestURL string   `json:"manifest_url,omitempty"`
	SHA256      string   `json:"sha256,omitempty"`
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
		cause := fmt.Errorf("%w: %s", ErrUnsupportedImage, options.ImageName)
		return PullResult{}, WrapError(ErrorCodeInvalidArgument, cause.Error(), cause)
	}

	cacheRoot, err := s.resolveYeastHome()
	if err != nil {
		return PullResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	cachePaths, err := images.ResolveCachePaths(cacheRoot+"/cache/images", image.Name)
	if err != nil {
		return PullResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	downloadOptions := s.downloadOptions()
	downloadOptions.Progress = options.Progress
	if err := s.downloadImage(image, cachePaths.ImageFile, downloadOptions); err != nil {
		return PullResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	return PullResult{
		ImageName:   image.Name,
		ImagePath:   cachePaths.ImageFile,
		ManifestURL: image.URL,
		SHA256:      image.SHA256,
	}, nil
}
