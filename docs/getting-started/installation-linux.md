# Install on Linux

This is the standard Yeast install path.

Yeast runs on Linux hosts with QEMU/KVM. First-class support is Ubuntu and Debian. Fedora and Arch are supported. Other distros are best-effort.

## Requirements

- Linux (x86_64 recommended; see note on arm64 below)
- `/dev/kvm` for hardware-accelerated VMs
- `qemu-system-x86_64` and `qemu-img`
- `genisoimage`, `mkisofs`, or `xorriso`
- `ssh`
- an SSH public key at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`

## Option 1: Install Script (recommended)

One command installs the release binary, prepares `~/.yeast`, bootstraps an SSH key if needed, and runs `yeast doctor --fix --yes`:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

The script downloads the pre-built binary from GitHub releases, verifies its checksum, and then asks Yeast itself to install any missing supported host packages it can fix automatically. It does not install Go and does not build from source in the normal path.

To pin a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | YEAST_VERSION=v1.1.1 bash
```

**Environment variables:**

| Variable | Default | Purpose |
|---|---|---|
| `YEAST_VERSION` | `v1.1.1` | Release tag to install |
| `YEAST_INSTALL_DIR` | `/usr/local/bin` | Where to place the binary |
| `YEAST_INSTALL_MODE` | `binary` | Set to `source` for source build (advanced) |
| `YEAST_SKIP_DOCTOR` | `0` | Set to `1` to skip the post-install doctor check |
| `YEAST_INSTALL_VERBOSE` | `0` | Set to `1` for verbose step output |

## Option 2: Manual Release Install

Download, verify, and install the binary directly:

```bash
VERSION="v1.1.1"
curl -LO "https://github.com/Twarga/yeast/releases/download/${VERSION}/yeast_linux_amd64.tar.gz"
curl -LO "https://github.com/Twarga/yeast/releases/download/${VERSION}/SHA256SUMS.txt"
grep "yeast_linux_amd64.tar.gz" SHA256SUMS.txt | sha256sum -c -
tar -xzf yeast_linux_amd64.tar.gz
sudo install -m 0755 yeast /usr/local/bin/yeast
```

## Verify the Install

```bash
yeast version
yeast doctor
```

`yeast doctor` checks the host and reports blockers or warnings.

If the version command fails, check that `/usr/local/bin` is in your `PATH`:

```bash
command -v yeast
```

## Install Host Packages

The install script now attempts this automatically on supported distros by running `yeast doctor --fix --yes`. If you prefer to install the packages yourself, use the commands below:

On Ubuntu or Debian:

```bash
sudo apt update
sudo apt install -y qemu-kvm qemu-utils genisoimage openssh-client
```

On Fedora:

```bash
sudo dnf install -y qemu-kvm qemu-img genisoimage openssh-clients
```

On Arch Linux:

```bash
sudo pacman -S qemu-full cdrtools openssh
```

## KVM Access

If `/dev/kvm` exists but your user cannot access it:

```bash
sudo usermod -aG kvm "$USER"
```

Log out and back in, then check:

```bash
yeast doctor
```

## SSH Key

If you do not have an SSH key yet:

```bash
ssh-keygen -t ed25519
```

The install script and `yeast doctor --fix --yes` both try to create this automatically when it is missing.

## Architecture Notes

x86_64 (amd64) is the primary supported host target. A release binary exists for arm64, but full runtime validation on arm64 hosts has not been completed. If you are on arm64, run `yeast doctor` after installing and verify your results before relying on it.

## Update Later

Check for a newer release:

```bash
yeast update --check
```

Update to a specific version:

```bash
yeast update --version v1.1.1
```

## Source Build (advanced)

If you need to build from source — for example to test unreleased changes or contribute — use:

```bash
YEAST_INSTALL_MODE=source bash install.sh
```

This installs Go, clones the repository, and compiles the binary. Normal users do not need this.

## Next Step

Continue with the [Quickstart](quickstart.md).
