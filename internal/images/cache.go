package images

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	ImageFileName    = "image.qcow2"
	ManifestFileName = "manifest.json"
)

var imageNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)

type CachePaths struct {
	Root         string
	ImageName    string
	ImageDir     string
	ImageFile    string
	ManifestFile string
}

func ResolveCachePaths(cacheRoot string, imageName string) (CachePaths, error) {
	root := filepath.Clean(strings.TrimSpace(cacheRoot))
	if root == "" || root == "." {
		return CachePaths{}, fmt.Errorf("image cache root is required")
	}
	if !IsSafeImageName(imageName) {
		return CachePaths{}, fmt.Errorf("invalid image name %q", imageName)
	}

	imageDir := filepath.Join(root, imageName)
	return CachePaths{
		Root:         root,
		ImageName:    imageName,
		ImageDir:     imageDir,
		ImageFile:    filepath.Join(imageDir, ImageFileName),
		ManifestFile: filepath.Join(imageDir, ManifestFileName),
	}, nil
}

func IsSafeImageName(name string) bool {
	if !imageNamePattern.MatchString(name) {
		return false
	}
	return !strings.Contains(name, "..")
}

// RemoveCachedImage deletes a cached image directory and returns the freed size in bytes.
func RemoveCachedImage(cacheRoot, imageName string) (int64, error) {
	paths, err := ResolveCachePaths(cacheRoot, imageName)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(paths.ImageDir)
	if err != nil {
		return 0, fmt.Errorf("image %q not cached", imageName)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("image %q not cached", imageName)
	}
	// Calculate size before removal.
	var size int64
	entries, _ := os.ReadDir(paths.ImageDir)
	for _, entry := range entries {
		fi, err := entry.Info()
		if err == nil {
			size += fi.Size()
		}
	}
	if err := os.RemoveAll(paths.ImageDir); err != nil {
		return 0, fmt.Errorf("remove cached image %s: %w", imageName, err)
	}
	return size, nil
}

// RemoveAllCached removes all cached images and returns total freed size in bytes.
func RemoveAllCached(cacheRoot string) (int64, error) {
	cached := GetCachedImages(cacheRoot)
	var totalSize int64
	for _, name := range cached {
		size, err := RemoveCachedImage(cacheRoot, name)
		if err != nil {
			return totalSize, err
		}
		totalSize += size
	}
	return totalSize, nil
}
