---
title: Installation
description: Install Yeast on your Linux or WSL system
---

# Installation

This guide walks you through installing Yeast on your system.

## System Requirements

- **Operating System**: Linux (Ubuntu 20.04+, Debian 11+, Fedora 38+, or similar)
- **Architecture**: x86_64 (AMD64)
- **RAM**: 4 GB minimum (8 GB recommended)
- **Disk**: 10 GB free space
- **Virtualization**: KVM support enabled (strongly recommended)

### WSL Support

Yeast can run on **WSL2** with the following caveats:

| Feature | Native Linux | WSL2 |
|---|---|---|
| KVM acceleration | Full | Limited (see below) |
| VM performance | Native speed | Slower without KVM |
| Private networking | Works | Works |
| QEMU/KVM | Works | Works with `qemu-system-x86_64` |

**Important:** WSL1 is **not supported** — it lacks a Linux kernel and cannot run QEMU/KVM VMs.

### Checking WSL2 KVM Support

```bash
# Check if KVM is available in WSL2
ls -la /dev/kvm

# If /dev/kvm exists, check nested virtualization
cat /proc/cpuinfo | grep vmx
cat /proc/cpuinfo | grep svm
```

**If `/dev/kvm` exists:** You have KVM support! Performance will be near-native.

**If `/dev/kvm` does not exist:** VMs will still work but run in TCG (software emulation) mode, which is significantly slower. To enable KVM in WSL2:

1. **Enable nested virtualization in Windows:**
   - Open PowerShell as Administrator
   - Run: `wsl --shutdown`
   - Create/edit `%USERPROFILE%\.wslconfig`:
   ```ini
   [wsl2]
   nestedVirtualization=true
   ```
   - Restart WSL: `wsl --shutdown` then reopen WSL

2. **Alternatively, enable Hyper-V:**
   - Open "Turn Windows features on or off"
   - Check "Virtual Machine Platform" and "Windows Subsystem for Linux"
   - Restart your computer

3. **Verify after restart:**
   ```bash
   ls -la /dev/kvm
   # Should show: crw-rw---- 1 root kvm 10, 232 ...
   ```

## Prerequisites

Yeast requires the following packages to be installed:

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install -y qemu-kvm qemu-utils genisoimage ssh-client
```

### Fedora/RHEL

```bash
sudo dnf install -y qemu-kvm qemu-img genisoimage openssh-clients
```

### Arch Linux

```bash
sudo pacman -S qemu-full cdrtools openssh
```

### WSL2 (Ubuntu)

```bash
sudo apt update
sudo apt install -y qemu-kvm qemu-utils genisoimage ssh-client

# If KVM is not available, install QEMU without KVM
sudo apt install -y qemu-system-x86 qemu-utils genisoimage ssh-client
```

## Check KVM Support

Verify that KVM is available on your system:

```bash
# Check if KVM module is loaded
lsmod | grep kvm

# Check if /dev/kvm exists
ls -la /dev/kvm

# Check if your user can access KVM
groups | grep kvm
```

If `/dev/kvm` doesn't exist or your user doesn't have access:

```bash
# Add your user to the kvm group
sudo usermod -aG kvm $USER

# Log out and log back in for changes to take effect
```

## Install Yeast

### Using the Install Script

The easiest way to install Yeast is using the official install script:

```bash
# Works in bash, zsh, fish, and any POSIX shell
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

::: tip Shell Compatibility
The `curl ... | bash` syntax works in **all shells** (bash, zsh, fish, etc.). The script itself runs inside bash regardless of your current shell.

If your shell doesn't support piping (rare), download then run:
```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh -o /tmp/yeast-install.sh
bash /tmp/yeast-install.sh
```
:::

This script will:
1. Detect your Linux distro (Ubuntu, Debian, Fedora, Arch, openSUSE, Alpine, etc.)
2. Check what prerequisites are already installed
3. Install only what's missing
4. Build Yeast from source
5. Install it to `/usr/local/bin/`
6. Verify the installation

### Manual Installation

If you prefer manual installation:

```bash
# Download the current release tarball and checksum file
VERSION="v1.1.0"
curl -LO "https://github.com/Twarga/yeast/releases/download/${VERSION}/yeast_linux_amd64.tar.gz"
curl -LO "https://github.com/Twarga/yeast/releases/download/${VERSION}/SHA256SUMS.txt"

# Verify the tarball
grep "yeast_linux_amd64.tar.gz" SHA256SUMS.txt | sha256sum -c -

# Extract the binary and install it
tar -xzf yeast_linux_amd64.tar.gz
sudo install -m 0755 yeast /usr/local/bin/yeast
```

### Build from Source

To build Yeast from source:

```bash
# Clone the repository
git clone https://github.com/Twarga/yeast.git
cd yeast

# Build the binary
go build -o yeast ./cmd/yeast

# Install it
sudo mv yeast /usr/local/bin/
```

## Verify Installation

After installation, verify that Yeast is working:

```bash
# Check the version
yeast --version

# Check system requirements
yeast doctor
```

The `yeast doctor` command checks:
- KVM support
- Required packages
- SSH key availability
- System resources

### Expected Output

A successful `yeast doctor` output looks like:

```
System Check
  OS:        linux
  KVM:       available
  QEMU:      found at /usr/bin/qemu-system-x86_64
  SSH key:   found at ~/.ssh/id_ed25519.pub
  Resources: sufficient
```

## SSH Key Setup

Yeast uses SSH keys for VM access. If you don't have an SSH key:

```bash
# Generate a new SSH key
ssh-keygen -t ed25519 -C "your_email@example.com"

# Start the SSH agent
eval "$(ssh-agent -s)"

# Add your key to the agent
ssh-add ~/.ssh/id_ed25519
```

## Troubleshooting Installation

### "KVM not available" on WSL2

If you see this warning, VMs will work but run slower:

```bash
# Check Windows version (must be Windows 11 22H2+ for nested virtualization)
winver

# In PowerShell (as Admin), check Hyper-V status
Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
Get-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform
```

**Fix:** Enable nested virtualization (see [WSL Support](#wsl-support) section above) or accept slower TCG mode.

### "qemu-system-x86_64 not found"

```bash
# Find where QEMU is installed
which qemu-system-x86_64 || find /usr -name qemu-system-x86_64 2>/dev/null

# If not found, install QEMU
sudo apt install qemu-system-x86  # Ubuntu/Debian
sudo dnf install qemu-system-x86    # Fedora
```

### "Permission denied on /dev/kvm"

```bash
# Fix permissions
sudo chmod 660 /dev/kvm
sudo chown root:kvm /dev/kvm

# Add user to kvm group and re-login
sudo usermod -aG kvm $USER
```

## Next Steps

- [Quickstart](./quickstart) — Get your first VM running in 5 minutes
- [Configuration](./configuration) — Learn about yeast.yaml
- [Troubleshooting](./troubleshooting) — Common issues and fixes
