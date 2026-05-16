package images

import (
	"fmt"
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
