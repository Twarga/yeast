package images

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

func FileChecksum(path string, expected string) (string, error) {
	hasher, err := checksumHasher(expected)
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum %s: %w", path, err)
	}
	defer file.Close()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("read file for checksum %s: %w", path, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func VerifyChecksum(path string, expected string) error {
	sum, err := FileChecksum(path, expected)
	if err != nil {
		return err
	}
	if !strings.EqualFold(sum, expected) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", path, strings.ToLower(expected), strings.ToLower(sum))
	}
	return nil
}

func VerifySHA256(path string, expected string) error {
	return VerifyChecksum(path, expected)
}

func checksumHasher(expected string) (hash.Hash, error) {
	trimmed := strings.TrimSpace(expected)
	switch len(trimmed) {
	case sha256.Size * 2:
		return sha256.New(), nil
	case sha512.Size * 2:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported checksum length %d", len(trimmed))
	}
}
