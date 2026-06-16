# Install on Linux

This is the normal Yeast install path.

Yeast runs on Linux hosts with QEMU/KVM.

## Requirements

You need:

- Linux
- AMD64/x86_64
- `/dev/kvm` for fast hardware virtualization
- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage` or `mkisofs`
- `ssh`
- an SSH public key at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`

## Install With The Script

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

For an explicit release:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | YEAST_REF=v1.1.0 bash
```

The installer is meant for fresh Linux hosts. It may install host packages, prepare the image cache directory, check SSH key availability, and run `yeast doctor`.

## Manual Release Install

```bash
VERSION="v1.1.0"
curl -LO "https://github.com/Twarga/yeast/releases/download/${VERSION}/yeast_linux_amd64.tar.gz"
curl -LO "https://github.com/Twarga/yeast/releases/download/${VERSION}/SHA256SUMS.txt"
grep "yeast_linux_amd64.tar.gz" SHA256SUMS.txt | sha256sum -c -
tar -xzf yeast_linux_amd64.tar.gz
sudo install -m 0755 yeast /usr/local/bin/yeast
```

The release archive must extract a binary named `yeast`.

## Verify The Install

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

If `/dev/kvm` exists but your user cannot access it, add yourself to the `kvm` group:

```bash
sudo usermod -aG kvm "$USER"
```

Then log out and back in.

Check access again:

```bash
yeast doctor
```

## SSH Key

If you do not have an SSH key yet:

```bash
ssh-keygen -t ed25519
```

Yeast uses your public key for guest access.

## Update Later

Check for a newer release:

```bash
yeast update --check
```

Update to a specific tag:

```bash
yeast update --version v1.1.0
```

## Next Step

Continue with the [Quickstart](quickstart.md).
