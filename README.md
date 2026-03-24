# Yeast

> Fast local VMs for Linux with KVM, QEMU, and cloud-init.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21-cyan.svg)
![Platform](https://img.shields.io/badge/platform-linux-lightgrey.svg)

Yeast is a project-based local VM orchestrator for Linux. It is built for developers who want real virtual machines locally without the weight and sprawl of older VM workflows.

Yeast gives you:
- project-local VM definitions in `yeast.yaml`
- trusted base image pulls with pinned SHA256 verification
- cloud-init bootstrapping
- fast copy-on-write overlays instead of full image copies
- simple lifecycle commands: `up`, `status`, `ssh`, `halt`, `down`, `restart`, `destroy`
- JSON output for automation

## Why Yeast

Yeast is opinionated and intentionally small.

It is designed for:
- Linux development environments
- reproducible local VM workflows
- simple KVM/QEMU usage without a large plugin ecosystem
- command-line automation

It is not trying to be:
- a GUI hypervisor manager
- a whole-machine VM inventory system
- a multi-host orchestrator
- a full cloud platform

## Installation

## One-command installer

If the repo is reachable over HTTPS, the installer can be run directly:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/dev/install.sh | bash
```

If you already cloned the repo locally:

```bash
bash install.sh
```

What `install.sh` does:
- detects the Linux package manager
- installs Yeast host dependencies
- installs Go for the source-build path
- clones and builds Yeast
- installs `yeast` into `/usr/local/bin`
- creates `~/.yeast/cache/`
- generates an SSH key if needed
- adds the user to the `kvm` group when possible

Optional installer overrides:

```bash
YEAST_REPO_URL=https://github.com/Twarga/yeast.git YEAST_REF=dev bash install.sh
```

Important:
- if the repo is private, the curl installer only works for users who already have access
- if the installer adds you to the `kvm` group, log out and back in before your first `yeast up`

## Build from source

```bash
git clone https://github.com/Twarga/yeast.git
cd yeast
go build -o yeast ./cmd/yeast
sudo mv yeast /usr/local/bin/
```

## Prerequisites

Yeast currently supports:
- Linux only

Yeast needs:
- KVM support on the host
- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage`
- an SSH public key in `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`

Typical package installs:

```bash
# Ubuntu / Debian
sudo apt install qemu-system-x86 qemu-utils genisoimage

# Fedora / RHEL
sudo dnf install qemu-system-x86 qemu-img genisoimage

# Arch Linux
sudo pacman -S qemu-base cdrtools
```

If needed, add your user to the `kvm` group:

```bash
sudo usermod -aG kvm $USER
```

## Quick Start

### 1. Check the host

```bash
yeast doctor
```

### 2. Create a project

```bash
mkdir my-project
cd my-project
yeast init
```

Default starter config:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 1
    user: yeast
    sudo: none
```

You can also shape the starter config immediately:

```bash
yeast init \
  --name api \
  --image ubuntu-24.04 \
  --memory 2048 \
  --cpus 2 \
  --user operator \
  --sudo password
```

### 3. Discover and pull an image

List supported trusted images:

```bash
yeast pull --list
```

Pull one:

```bash
yeast pull ubuntu-22.04
```

Current built-in trusted images:
- `ubuntu-22.04`
- `ubuntu-24.04`

### 4. Start the VMs

```bash
yeast up
```

### 5. Check status

```bash
yeast status
```

Example output:

```text
NAME    STATUS    PID     IP           SSH PORT
web     running   12345   127.0.0.1    45678
```

For scripts:

```bash
yeast status --json
```

### 6. Connect over SSH

```bash
yeast ssh web
```

If you initialized the instance with a different user:

```bash
yeast ssh web --user operator
```

### 7. Stop the environment

```bash
yeast down
```

## Example `yeast.yaml`

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 2048
    cpus: 2
    user: yeast
    sudo: password
    env:
      APP_ENV: development
      LOG_LEVEL: debug
```

Supported fields today:
- `name`
- `image`
- `memory`
- `cpus`
- `user`
- `sudo`
- `env`
- `user_data`

Important config behavior:
- `user_data`, if provided, replaces Yeast's generated bootstrap cloud-init
- custom `user_data` does not automatically merge in your SSH key, user, sudo policy, or env values
- configurable disk size is not supported yet

## Networking

Yeast supports 3 network modes on `up` and `restart`:

- `user`: default QEMU user-mode NAT with SSH port forwarding
- `private`: user-mode NAT with `restrict=on`
- `bridge`: host bridge attachment plus a restricted management NIC for SSH

Examples:

```bash
yeast up --network-mode user
yeast up --network-mode private
yeast up --network-mode bridge --bridge br0
yeast restart web --network-mode bridge --bridge br0
```

Current networking limits:
- only SSH port forwarding is built in
- no custom `8080:80` style forwards yet
- network mode is chosen at command time
- network mode is not stored in `yeast.yaml`

## Project Model

Yeast is project-local.

Project directory files:
- `yeast.yaml`: desired VM definitions
- `yeast.state`: runtime state for the current project

Home directory paths:
- `~/.yeast/cache/`: shared base images
- `~/.yeast/instances/<name>/`: per-instance runtime files

Each instance directory typically contains:
- `disk.qcow2`
- `seed.iso`
- `user-data`
- `meta-data`
- `vm.log`

## Command Surface

Main commands:
- `yeast doctor`
- `yeast init`
- `yeast pull`
- `yeast up`
- `yeast status`
- `yeast ssh`
- `yeast halt`
- `yeast down`
- `yeast restart`
- `yeast destroy`

Most major commands support:

```bash
yeast <command> --json
```

`yeast ssh` is interactive and does not support JSON output.

## Current Limits

Current non-goals or missing features:
- Linux only
- no GUI
- no global VM inventory command
- no arbitrary port-forward configuration
- no configurable disk size
- no snapshots or suspend/resume
- no per-instance `yeast up <name>`
- no guest IP discovery in bridge mode

## Docs

For the full user guide, see [docs.md](docs.md).

Also useful:
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CHANGELOG.md](CHANGELOG.md)
- [LICENSE](LICENSE)
