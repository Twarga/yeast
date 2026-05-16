package cloudinit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrNoSSHPublicKey = errors.New("no supported ssh public key found")

func DiscoverAuthorizedKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	for _, candidate := range []string{"id_ed25519.pub", "id_rsa.pub"} {
		path := filepath.Join(home, ".ssh", candidate)
		key, err := readAuthorizedKey(path)
		if err == nil {
			return key, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return "", err
	}

	return "", fmt.Errorf("%w: checked ~/.ssh/id_ed25519.pub and ~/.ssh/id_rsa.pub", ErrNoSSHPublicKey)
}

func readAuthorizedKey(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read ssh public key %s: %w", path, err)
	}

	key := strings.TrimSpace(string(content))
	if key == "" {
		return "", fmt.Errorf("ssh public key %s is empty", path)
	}

	return key, nil
}
