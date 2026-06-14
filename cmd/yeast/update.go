package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		force   bool
		check   bool
		version string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update yeast to the latest release",
		Long:  `Checks GitHub for the latest release, downloads the pre-built binary, verifies its checksum, and replaces the current binary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, force, check, version)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force update even if already on latest version")
	cmd.Flags().BoolVar(&check, "check", false, "Only check for updates, don't install")
	cmd.Flags().StringVar(&version, "version", "", "Update to specific version (e.g., v1.1.0)")

	return cmd
}

type ReleaseInfo struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Body       string    `json:"body"`
	Published  time.Time `json:"published_at"`
	Assets     []Asset   `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func runUpdate(cmd *cobra.Command, force, check bool, targetVersion string) error {
	currentVersion := app.Version
	if currentVersion == "" {
		currentVersion = "dev"
	}

	fmt.Printf("Current version: %s\n", currentVersion)

	if targetVersion != "" {
		if !strings.HasPrefix(targetVersion, "v") {
			targetVersion = "v" + targetVersion
		}
	} else {
		info, err := fetchLatestRelease()
		if err != nil {
			return fmt.Errorf("fetch latest release: %w", err)
		}
		targetVersion = info.TagName
	}

	if !force && targetVersion == currentVersion {
		fmt.Println("Already on latest version.")
		return nil
	}

	fmt.Printf("Target version: %s\n", targetVersion)

	if check {
		if targetVersion != currentVersion {
			fmt.Println("Update available!")
		} else {
			fmt.Println("No update available.")
		}
		return nil
	}

	binaryName := binaryNameForPlatform()
	if binaryName == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	assetURL := fmt.Sprintf("https://github.com/Twarga/yeast/releases/download/%s/%s", targetVersion, binaryName)
	checksumsURL := fmt.Sprintf("https://github.com/Twarga/yeast/releases/download/%s/SHA256SUMS.txt", targetVersion)

	fmt.Printf("Downloading %s...\n", binaryName)
	binaryData, err := downloadFile(assetURL)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}

	fmt.Println("Downloading checksums...")
	checksumsData, err := downloadFile(checksumsURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	expectedHash, err := parseChecksums(string(checksumsData), binaryName)
	if err != nil {
		return fmt.Errorf("parse checksums: %w", err)
	}

	fmt.Println("Verifying checksum...")
	actualHash := sha256Sum(binaryData)
	if !bytes.Equal(expectedHash, actualHash) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", hex.EncodeToString(expectedHash), hex.EncodeToString(actualHash))
	}
	fmt.Println("Checksum verified!")

	var binaryBytes []byte
	if strings.HasSuffix(binaryName, ".tar.gz") {
		fmt.Println("Extracting archive...")
		binaryBytes, err = extractBinaryFromTarGz(binaryData, "yeast")
		if err != nil {
			return fmt.Errorf("extract binary: %w", err)
		}
	} else {
		binaryBytes = binaryData
	}

	binaryPath, err := findBinaryPath()
	if err != nil {
		return fmt.Errorf("find binary path: %w", err)
	}

	fmt.Printf("Installing to %s...\n", binaryPath)
	if err := atomicReplaceBinary(binaryPath, binaryBytes); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}

	fmt.Printf("Successfully updated to %s!\n", targetVersion)
	return nil
}

func fetchLatestRelease() (*ReleaseInfo, error) {
	url := "https://api.github.com/repos/Twarga/yeast/releases/latest"
	req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "yeast-update")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func downloadFile(url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	req.Header.Set("User-Agent", "yeast-update")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func parseChecksums(content, binaryName string) ([]byte, error) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == binaryName {
			return hex.DecodeString(parts[0])
		}
	}
	return nil, fmt.Errorf("checksum not found for %s", binaryName)
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func binaryNameForPlatform() string {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "yeast_linux_amd64.tar.gz"
		case "arm64":
			return "yeast_linux_arm64.tar.gz"
		}
	}
	return ""
}

func extractBinaryFromTarGz(data []byte, binaryName string) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(header.Name) == binaryName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %s not found in archive", binaryName)
}

func findBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	realExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return realExe, nil
}

func atomicReplaceBinary(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)
	tmpFile, err := os.CreateTemp(dir, "yeast-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Chmod(0755); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, targetPath)
}