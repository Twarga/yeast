package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"yeast/pkg/cloudinit"
	"yeast/pkg/util"
)

const (
	activeVMLogName       = "vm.log"
	vmLogArchivePrefix    = "vm."
	vmLogArchiveSuffix    = ".log"
	defaultVMLogRetention = 5

	NetworkModeUser    = "user"
	NetworkModePrivate = "private"
	NetworkModeBridge  = "bridge"
)

type NetworkOptions struct {
	Mode   string
	Bridge string
}

type Machine struct {
	Name     string
	Image    string
	Memory   int
	CPUs     int
	DiskSize string
	Dir      string
	SSHPort  int
	User     string
	Sudo     string
	UserData string
	Env      map[string]string
	Network  NetworkOptions
}

func New(name, image string, memory, cpus int, diskSize, userData string, env map[string]string, user, sudo string, network NetworkOptions) *Machine {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".yeast", "instances", name)
	return &Machine{
		Name:     name,
		Image:    image,
		Memory:   memory,
		CPUs:     cpus,
		DiskSize: strings.TrimSpace(diskSize),
		Dir:      dir,
		User:     user,
		Sudo:     sudo,
		UserData: userData,
		Env:      env,
		Network:  network,
	}
}

func (m *Machine) CreateDisk() error {
	overlayPath := filepath.Join(m.Dir, "disk.qcow2")
	if _, err := os.Stat(overlayPath); err == nil {
		return m.ensureDiskSize(overlayPath)
	}

	if err := os.MkdirAll(m.Dir, 0755); err != nil {
		return err
	}

	// Resolve base image path
	// For MVP, assume images are in ~/.yeast/cache/
	home, _ := os.UserHomeDir()
	baseImage := filepath.Join(home, ".yeast", "cache", m.Image+".img")

	// Check if base image exists
	if _, err := os.Stat(baseImage); os.IsNotExist(err) {
		return fmt.Errorf("base image not found at %s. Please download it manually.", baseImage)
	}

	// #nosec G204 -- command and argument structure are fixed; variable paths are local instance/image files.
	args := []string{"create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", baseImage,
		overlayPath,
	}
	if strings.TrimSpace(m.DiskSize) != "" {
		normalized, err := util.NormalizeByteSize(m.DiskSize)
		if err != nil {
			return fmt.Errorf("invalid disk size %q: %w", m.DiskSize, err)
		}
		args = append(args, normalized)
	}
	cmd := exec.Command("qemu-img", args...)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create disk: %s: %s", err, string(output))
	}

	return nil
}

func (m *Machine) ensureDiskSize(diskPath string) error {
	if strings.TrimSpace(m.DiskSize) == "" {
		return nil
	}

	desiredSize, err := util.ParseByteSize(m.DiskSize)
	if err != nil {
		return fmt.Errorf("invalid disk size %q: %w", m.DiskSize, err)
	}
	currentSize, err := diskVirtualSizeBytes(diskPath)
	if err != nil {
		return fmt.Errorf("failed to inspect disk size for %s: %w", diskPath, err)
	}
	if desiredSize <= currentSize {
		return nil
	}

	normalized, err := util.NormalizeByteSize(m.DiskSize)
	if err != nil {
		return fmt.Errorf("invalid disk size %q: %w", m.DiskSize, err)
	}

	// #nosec G204 -- qemu-img is invoked with explicit subcommand and validated local disk path/size.
	cmd := exec.Command("qemu-img", "resize", diskPath, normalized)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to resize disk: %s: %s", err, string(output))
	}
	return nil
}

func diskVirtualSizeBytes(path string) (int64, error) {
	// #nosec G204 -- qemu-img info is invoked with an explicit local disk path.
	cmd := exec.Command("qemu-img", "info", "--output", "json", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("qemu-img info failed: %s: %s", err, string(output))
	}

	var info struct {
		VirtualSize int64 `json:"virtual-size"`
	}
	if err := json.Unmarshal(output, &info); err != nil {
		return 0, fmt.Errorf("failed to parse qemu-img info output: %w", err)
	}
	if info.VirtualSize <= 0 {
		return 0, fmt.Errorf("qemu-img info returned invalid virtual size for %s", path)
	}
	return info.VirtualSize, nil
}

func (m *Machine) PrepareConfig() error {
	var ud string
	if strings.TrimSpace(m.UserData) != "" {
		ud = ensureCloudConfigHeader(m.UserData)
	} else {
		key, err := cloudinit.LoadSSHKey()
		if err != nil {
			return err
		}
		ud = cloudinit.GenerateUserData(cloudinit.UserData{
			Hostname:   m.Name,
			SSHKey:     key,
			Username:   m.User,
			SudoPolicy: cloudinit.SudoPolicy(m.Sudo),
			Env:        m.Env,
		})
	}

	md := cloudinit.GenerateMetaData(m.Name, m.Name)

	return cloudinit.CreateISO(m.Dir, ud, md)
}

func ensureCloudConfigHeader(userData string) string {
	trimmed := strings.TrimSpace(userData)
	if strings.HasPrefix(trimmed, "#cloud-config") {
		return userData
	}
	return "#cloud-config\n" + userData
}

func (m *Machine) Start() (int, error) {
	diskPath := filepath.Join(m.Dir, "disk.qcow2")
	seedPath := filepath.Join(m.Dir, "seed.iso")

	args := []string{
		"-enable-kvm",
		"-m", fmt.Sprintf("%d", m.Memory),
		"-smp", fmt.Sprintf("%d", m.CPUs),
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", diskPath),
		"-drive", fmt.Sprintf("file=%s,format=raw,media=cdrom", seedPath),
		"-nographic",
	}
	networkArgs, err := m.qemuNetworkArgs()
	if err != nil {
		return 0, err
	}
	args = append(args, networkArgs...)

	// #nosec G204 -- qemu invocation is intentional; args are built from validated config and fixed option templates.
	cmd := exec.Command("qemu-system-x86_64", args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	cmd.Dir = m.Dir

	logFile, err := prepareVMLogFile(m.Dir, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return 0, err
	}
	_ = logFile.Close()

	pid := cmd.Process.Pid

	if err := cmd.Process.Release(); err != nil {
		return 0, err
	}

	return pid, nil
}

func (m *Machine) qemuNetworkArgs() ([]string, error) {
	mode := strings.ToLower(strings.TrimSpace(m.Network.Mode))
	if mode == "" {
		mode = NetworkModeUser
	}

	switch mode {
	case NetworkModeUser:
		return []string{
			"-netdev", fmt.Sprintf("user,id=mgmt0,hostfwd=tcp::%d-:22", m.SSHPort),
			"-device", "virtio-net-pci,netdev=mgmt0",
		}, nil
	case NetworkModePrivate:
		return []string{
			"-netdev", fmt.Sprintf("user,id=mgmt0,restrict=on,hostfwd=tcp::%d-:22", m.SSHPort),
			"-device", "virtio-net-pci,netdev=mgmt0",
		}, nil
	case NetworkModeBridge:
		bridge := strings.TrimSpace(m.Network.Bridge)
		if bridge == "" {
			return nil, errors.New("network mode bridge requires a bridge name")
		}
		return []string{
			"-netdev", fmt.Sprintf("bridge,id=lan0,br=%s", bridge),
			"-device", "virtio-net-pci,netdev=lan0",
			"-netdev", fmt.Sprintf("user,id=mgmt0,restrict=on,hostfwd=tcp::%d-:22", m.SSHPort),
			"-device", "virtio-net-pci,netdev=mgmt0",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported network mode %q (supported: %s, %s, %s)", mode, NetworkModeUser, NetworkModePrivate, NetworkModeBridge)
	}
}

func prepareVMLogFile(instanceDir string, now time.Time) (*os.File, error) {
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create instance directory for logs: %w", err)
	}

	if err := rotateActiveVMLog(instanceDir, now); err != nil {
		return nil, err
	}
	if err := pruneArchivedVMLogs(instanceDir, vmLogRetention()); err != nil {
		return nil, err
	}

	activePath := filepath.Join(instanceDir, activeVMLogName)
	f, err := os.OpenFile(activePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open active VM log file %s: %w", activePath, err)
	}
	return f, nil
}

func rotateActiveVMLog(instanceDir string, now time.Time) error {
	activePath := filepath.Join(instanceDir, activeVMLogName)
	if _, err := os.Stat(activePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat active VM log %s: %w", activePath, err)
	}

	archivePath := filepath.Join(instanceDir, archiveVMLogName(now))
	if err := os.Rename(activePath, archivePath); err != nil {
		return fmt.Errorf("failed to rotate active VM log %s -> %s: %w", activePath, archivePath, err)
	}
	return nil
}

func archiveVMLogName(now time.Time) string {
	utc := now.UTC()
	return fmt.Sprintf("%s%s.%d%s", vmLogArchivePrefix, utc.Format("20060102T150405Z"), utc.UnixNano(), vmLogArchiveSuffix)
}

func pruneArchivedVMLogs(instanceDir string, retention int) error {
	entries, err := os.ReadDir(instanceDir)
	if err != nil {
		return fmt.Errorf("failed to read instance directory for log pruning: %w", err)
	}

	type archived struct {
		name    string
		modTime time.Time
	}
	archives := make([]archived, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == activeVMLogName {
			continue
		}
		if !strings.HasPrefix(name, vmLogArchivePrefix) || !strings.HasSuffix(name, vmLogArchiveSuffix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to inspect archived log %s: %w", filepath.Join(instanceDir, name), err)
		}
		archives = append(archives, archived{name: name, modTime: info.ModTime()})
	}

	sort.Slice(archives, func(i, j int) bool {
		if archives[i].modTime.Equal(archives[j].modTime) {
			return archives[i].name > archives[j].name
		}
		return archives[i].modTime.After(archives[j].modTime)
	})

	for i := retention; i < len(archives); i++ {
		if err := os.Remove(filepath.Join(instanceDir, archives[i].name)); err != nil {
			return fmt.Errorf("failed to remove archived VM log %s: %w", archives[i].name, err)
		}
	}
	return nil
}

func vmLogRetention() int {
	raw := strings.TrimSpace(os.Getenv("YEAST_VM_LOG_RETENTION"))
	if raw == "" {
		return defaultVMLogRetention
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultVMLogRetention
	}
	return n
}
