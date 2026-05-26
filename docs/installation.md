# Yeast Installation

Yeast currently targets Linux hosts with KVM.

## Requirements

Required host tools:

- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage` or `mkisofs`
- `ssh`
- `ssh-keygen`
- Go 1.25+ if building from source

Required host access:

- `/dev/kvm` must exist and be accessible
- the current user should be allowed to use KVM
- an SSH public key must exist at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`

## Install Host Packages

Ubuntu / Debian:

```bash
sudo apt install qemu-system-x86 qemu-utils genisoimage openssh-client
```

Fedora / RHEL:

```bash
sudo dnf install qemu-system-x86 qemu-img genisoimage openssh-clients
```

Arch Linux:

```bash
sudo pacman -S qemu-base cdrtools openssh
```

openSUSE:

```bash
sudo zypper install qemu-x86 qemu-tools genisoimage openssh
```

Alpine:

```bash
sudo apk add qemu-system-x86_64 qemu-img cdrkit openssh-client
```

## KVM Permissions

If `/dev/kvm` exists but your user cannot access it, add the user to the host KVM group.

Common Linux command:

```bash
sudo usermod -aG kvm $USER
```

Then log out and back in.

On some distributions the group name or permission model may differ. Use `ls -l /dev/kvm` to inspect the required group.

## SSH Key

If you do not have a supported SSH key:

```bash
ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
```

Yeast uses the public key during cloud-init bootstrap so `yeast ssh` can connect after the VM starts.

## Build From Source

```bash
git clone https://github.com/Twarga/yeast.git
cd yeast
go build -o yeast ./cmd/yeast
sudo install -m 0755 yeast /usr/local/bin/yeast
```

Check the install:

```bash
yeast version
yeast doctor
```

## Installer Script

From a cloned repo:

```bash
bash install.sh
```

From GitHub:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

The installer installs the latest stable release by default. It attempts to install host packages, build Yeast, install the binary, verify the installed version, create cache directories, and generate an SSH key if needed.

To install an explicit release, branch, or commit:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | YEAST_REF=main bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | YEAST_REF=v0.9.0 bash
```

When `YEAST_REF` is a semantic version tag such as `v0.9.0`, the installer injects that version into the built binary and verifies `yeast version` after installation.

When installing a branch or commit that is not a semantic version tag, the installer builds with `0.0.0-dev` unless `YEAST_EXPECTED_VERSION` is set.

If the installer adds your user to a group, log out and back in before running `yeast up`.

## Install Script Behavior

The installer is a release-critical path.

The script should support:

- `apt`
- `dnf`
- `yum`
- `pacman`
- `zypper`
- `apk`

The script should install or verify:

- QEMU system runtime
- `qemu-img`
- `genisoimage` or compatible `mkisofs`
- `ssh`
- `ssh-keygen`
- `git`
- Go 1.25+ for source builds

The script should also:

- create `~/.yeast`
- create `~/.yeast/cache`
- create `~/.yeast/cache/images`
- generate an SSH key if none exists
- detect KVM permissions
- add the target user to the `kvm` group when that group exists
- explain when logout/login is required
- keep logs when install steps fail
- run `yeast doctor` after installing
- verify the installed binary exists and prints the expected version for release tags

Supported overrides:

```bash
YEAST_REPO_URL=https://github.com/Twarga/yeast.git
YEAST_REF=v0.9.0
YEAST_INSTALL_DIR=/usr/local/bin
YEAST_EXPECTED_VERSION=
YEAST_INSTALL_VERBOSE=1
YEAST_KEEP_LOGS=1
YEAST_MIN_GO_VERSION=1.25.0
YEAST_GO_VERSION=1.26.3
YEAST_GO_INSTALL_ROOT=/usr/local/lib/yeast/go
YEAST_GO_TARBALL_SHA256=
```

If the distribution Go package is older than `YEAST_MIN_GO_VERSION`, the installer downloads an official Go toolchain tarball from `go.dev` for the current Linux architecture.

Set `YEAST_GO_TARBALL_SHA256` if you want the installer to verify the downloaded Go toolchain with a pinned checksum.

The installer supports these CPU architectures:

- `amd64`
- `arm64`

## Upgrade

Run the installer again with the desired `YEAST_REF`.

Examples:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | YEAST_REF=v0.9.0 bash
```

The installer overwrites the binary at `YEAST_INSTALL_DIR/yeast`. It does not delete project directories, cached images, disks, snapshots, or state under `~/.yeast`.
