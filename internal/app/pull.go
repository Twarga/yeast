package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"yeast/internal/images"
)

var ErrUnsupportedImage = errors.New("unsupported image")

type PullOptions struct {
	ImageName string
	List      bool
	Cached    bool
}

type PullResult struct {
	List          bool              `json:"list,omitempty"`
	Images        []string          `json:"images,omitempty"`
	ImageGroups   []ImageGroup      `json:"image_groups,omitempty"`
	ImageName     string            `json:"image_name,omitempty"`
	ImagePath     string            `json:"image_path,omitempty"`
	ManifestURL   string            `json:"manifest_url,omitempty"`
	SHA256        string            `json:"sha256,omitempty"`
	SearchResults []string          `json:"search_results,omitempty"`
	ManualHint    string            `json:"manual_hint,omitempty"`
	CachedImages  []CachedImageInfo `json:"cached_images,omitempty"`
}

type ImageGroup struct {
	Category string                `json:"category"`
	Images   []images.TrustedImage `json:"images"`
}

type CachedImageInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (s *Service) Pull(options PullOptions) (PullResult, error) {
	if options.Cached {
		return s.pullCached()
	}

	if options.List {
		return s.pullList(), nil
	}

	image, ok := images.Lookup(options.ImageName)
	if !ok {
		matches := images.Search(options.ImageName)
		if len(matches) == 1 {
			image, _ = images.Lookup(matches[0])
		} else if len(matches) > 1 {
			return PullResult{SearchResults: matches}, nil
		} else {
			msg := fmt.Sprintf("image %q not found", options.ImageName)
			suggestions := images.SuggestSimilar(options.ImageName, 3)
			if len(suggestions) > 0 {
				msg += "\n\nDid you mean?\n"
				for _, s := range suggestions {
					msg += fmt.Sprintf("  - %s\n", s)
				}
			}
			msg += "\nRun \"yeast pull --list\" for all available images."
			cause := fmt.Errorf("%w: %s", ErrUnsupportedImage, options.ImageName)
			return PullResult{}, WrapError(ErrorCodeInvalidArgument, msg, cause)
		}
	}

	// Manual images — show instructions instead of downloading.
	if image.URL == "" && image.ManualInstructions != "" {
		return PullResult{
			ImageName:  image.Name,
			ManualHint: image.ManualInstructions,
		}, nil
	}

	cacheDir, err := s.resolveImageCacheDir()
	if err != nil {
		return PullResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	cachePaths, err := images.ResolveCachePaths(cacheDir, image.Name)
	if err != nil {
		return PullResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	if err := s.downloadImage(image, cachePaths.ImageFile, s.downloadOptions()); err != nil {
		return PullResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	return PullResult{
		ImageName:   image.Name,
		ImagePath:   cachePaths.ImageFile,
		ManifestURL: image.URL,
		SHA256:      image.Checksum,
	}, nil
}

func (s *Service) pullList() PullResult {
	categoryOrder := []images.ImageCategory{
		images.CategoryGeneral,
		images.CategoryDevOps,
		images.CategoryEnterprise,
		images.CategorySecurity,
		images.CategoryMinimal,
		images.CategoryNiche,
	}

	all := images.ListByCategory()
	var groups []ImageGroup
	for _, cat := range categoryOrder {
		imgs, ok := all[cat]
		if !ok || len(imgs) == 0 {
			continue
		}
		groups = append(groups, ImageGroup{
			Category: string(cat),
			Images:   imgs,
		})
	}

	return PullResult{
		List:        true,
		Images:      images.SupportedImages(),
		ImageGroups: groups,
	}
}

func (s *Service) pullCached() (PullResult, error) {
	cacheDir, err := s.resolveImageCacheDir()
	if err != nil {
		return PullResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	cached := images.GetCachedImages(cacheDir)
	var infos []CachedImageInfo
	for _, name := range cached {
		paths, _ := images.ResolveCachePaths(cacheDir, name)
		infos = append(infos, CachedImageInfo{
			Name: name,
			Path: paths.ImageDir,
		})
	}
	return PullResult{CachedImages: infos}, nil
}

func (s *Service) resolveImageCacheDir() (string, error) {
	cacheRoot, err := s.resolveYeastHome()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cacheRoot) == "" {
		return "", fmt.Errorf("resolve yeast home: empty path")
	}
	return filepath.Join(cacheRoot, "cache", "images"), nil
}
