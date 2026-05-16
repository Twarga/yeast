# Yeast Installation

Yeast v0.1 currently targets Linux hosts with KVM.

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

The installer attempts to install host packages, build Yeast, install the binary, create cache directories, and generate an SSH key if needed.

If the installer adds your user to a group, log out and back in before running `yeast up`.
