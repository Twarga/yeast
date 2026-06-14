package app

import (
	"fmt"
	"os"
	"yeast/internal/images"
)

type ImageCleanOptions struct {
	ImageName string
	All       bool
	DryRun    bool
}

type ImageCleanResult struct {
	DryRun     bool               `json:"dry_run,omitempty"`
	Removed    []CleanedImageItem `json:"removed,omitempty"`
	TotalSize  int64              `json:"total_size_bytes,omitempty"`
	TotalSizeH string             `json:"total_size,omitempty"`
}

type CleanedImageItem struct {
	Name  string `json:"name"`
	Size  int64  `json:"size_bytes"`
	SizeH string `json:"size,omitempty"`
}

func (s *Service) CleanImages(options ImageCleanOptions) (ImageCleanResult, error) {
	cacheDir, err := s.resolveImageCacheDir()
	if err != nil {
		return ImageCleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	if options.All {
		if options.DryRun {
			cached := images.GetCachedImages(cacheDir)
			var items []CleanedImageItem
			var total int64
			for _, name := range cached {
				paths, _ := images.ResolveCachePaths(cacheDir, name)
				info := dirSize(paths.ImageDir)
				items = append(items, CleanedImageItem{
					Name:  name,
					Size:  info,
					SizeH: humanSize(info),
				})
				total += info
			}
			return ImageCleanResult{
				DryRun:     true,
				Removed:    items,
				TotalSize:  total,
				TotalSizeH: humanSize(total),
			}, nil
		}
		total, err := images.RemoveAllCached(cacheDir)
		if err != nil {
			return ImageCleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		return ImageCleanResult{
			TotalSize:  total,
			TotalSizeH: humanSize(total),
		}, nil
	}

	if options.ImageName == "" {
		return ImageCleanResult{}, WrapError(ErrorCodeInvalidArgument, "specify an image name or use --all", nil)
	}

	if options.DryRun {
		paths, err := images.ResolveCachePaths(cacheDir, options.ImageName)
		if err != nil {
			return ImageCleanResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
		}
		if !images.IsCached(cacheDir, options.ImageName) {
			return ImageCleanResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("image %q not cached", options.ImageName), nil)
		}
		info := dirSize(paths.ImageDir)
		return ImageCleanResult{
			DryRun: true,
			Removed: []CleanedImageItem{{
				Name:  options.ImageName,
				Size:  info,
				SizeH: humanSize(info),
			}},
			TotalSize:  info,
			TotalSizeH: humanSize(info),
		}, nil
	}

	if !images.IsCached(cacheDir, options.ImageName) {
		return ImageCleanResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("image %q not cached", options.ImageName), nil)
	}

	size, err := images.RemoveCachedImage(cacheDir, options.ImageName)
	if err != nil {
		return ImageCleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	return ImageCleanResult{
		Removed: []CleanedImageItem{{
			Name:  options.ImageName,
			Size:  size,
			SizeH: humanSize(size),
		}},
		TotalSize:  size,
		TotalSizeH: humanSize(size),
	}, nil
}

func dirSize(path string) int64 {
	var size int64
	entries, _ := os.ReadDir(path)
	for _, entry := range entries {
		info, err := entry.Info()
		if err == nil {
			size += info.Size()
		}
	}
	return size
}

func humanSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
