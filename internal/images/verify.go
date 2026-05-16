package images

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func FileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum %s: %w", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("read file for checksum %s: %w", path, err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func VerifySHA256(path string, expected string) error {
	sum, err := FileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(sum, expected) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", path, strings.ToLower(expected), strings.ToLower(sum))
	}
	return nil
}
