# 🍞 Yeast

> Fast local VMs for Linux — powered by KVM, QEMU, and cloud-init.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21-cyan.svg)
![Platform](https://img.shields.io/badge/platform-linux-lightgrey.svg)

Yeast is a project-local VM tool for developers who want **real virtual machines** — not containers — with a workflow that doesn't get in the way. Describe your machines in `yeast.yaml`, pull a trusted image, run `yeast up`, and Yeast handles QEMU/KVM, cloud-init, networking, and SSH for you.

---

## Table of Contents

- [Overview](#overview)
- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Networking](#networking)
- [Command Reference](#command-reference)
- [How Yeast Stores Data](#how-yeast-stores-data)
- [Current Limits](#current-limits)

---

## Overview

| Area | What Yeast gives you |
|---|---|
| VM model | Project-local VMs defined in `yeast.yaml` |
| Base images | Shared cache in `~/.yeast/cache/` |
| Provisioning | cloud-init |
| Runtime | QEMU + KVM |
| Networking | `user`, `private`, `bridge` |
| SSH access | Automatic host port forwarding → guest port 22 |
| Automation | `--json` on all major commands |

---

## Requirements

Yeast runs on **Linux only**.

**System dependencies:**

```bash
# Ubuntu / Debian
sudo apt install qemu-system-x86 qemu-utils genisoimage

# Fedora / RHEL
sudo dnf install qemu-system-x86 qemu-img genisoimage

# Arch Linux
sudo pacman -S qemu-base cdrtools
```

**KVM access:**

```bash
sudo usermod -aG kvm $USER
# Log out and back in before running yeast up
```

**SSH key:** Yeast expects a key at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`.

---

## Installation

### One-command install

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/dev/install.sh | bash
```

What the installer does:
- Detects your Linux package manager and installs dependencies
- Installs Go (for the source-build path)
- Clones and builds Yeast
- Places `yeast` in `/usr/local/bin`
- Creates `~/.yeast/cache/`
- Generates an SSH key if none exists
- Adds you to the `kvm` group when possible

> **Note:** If the repo is private, this only works for users who already have access.

### Already cloned the repo?

```bash
bash install.sh
```

### Custom install (override defaults)

```bash
YEAST_REPO_URL=https://github.com/Twarga/yeast.git YEAST_REF=dev bash install.sh
```

### Build from source

```bash
git clone https://github.com/Twarga/yeast.git
cd yeast
go build -o yeast ./cmd/yeast
sudo mv yeast /usr/local/bin/
```

---

## Quick Start

### 1. Check the host

```bash
yeast doctor
```

### 2. Create a project

```bash
mkdir my-project && cd my-project
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

Or generate with your own values:

```bash
yeast init \
  --name api \
  --image ubuntu-24.04 \
  --memory 2048 \
  --cpus 2 \
  --user operator \
  --sudo password
```

### 3. Pull an image

```bash
yeast pull --list        # see available images
yeast pull ubuntu-22.04  # pull one
```

Currently available images: `ubuntu-22.04`, `ubuntu-24.04`

### 4. Start the environment

```bash
yeast up
```

### 5. Check status

```bash
yeast status
```

Example output:

```
NAME    STATUS    PID     IP           SSH PORT
web     running   12345   127.0.0.1    45678
```

For scripting:

```bash
yeast status --json
```

### 6. Connect via SSH

```bash
yeast ssh web

# If using a custom bootstrap user:
yeast ssh web --user operator
```

### 7. Tear down

```bash
yeast down
```

---

## Configuration

### yeast.yaml example

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

### Supported fields

| Field | Type | Description |
|---|---|---|
| `name` | string | Instance name, used in all commands |
| `image` | string | Base image slug (e.g. `ubuntu-22.04`) |
| `memory` | int (MB) | RAM in megabytes |
| `cpus` | int | Number of virtual CPUs |
| `user` | string | Bootstrap user created by cloud-init |
| `sudo` | string | `none` \| `nopassword` \| `password` |
| `env` | map | Environment variables injected via cloud-init |
| `user_data` | string | Raw cloud-init user-data — **replaces** Yeast's generated config entirely |

> **Warning:** If `user_data` is set, Yeast's generated cloud-init is not used. Your SSH key, user, sudo policy, and env values are **not** merged automatically — include them yourself.

---

## Networking

Yeast supports three network modes. Pass `--network-mode` to `yeast up` or `yeast restart`. The mode is **not** stored in `yeast.yaml` — it is selected at command time.

| Mode | Description |
|---|---|
| `user` | Default QEMU user-mode NAT with SSH port forwarding |
| `private` | User-mode NAT with `restrict=on` — VM cannot reach the host or internet |
| `bridge` | Attaches to a host bridge + restricted management NIC for SSH |

**Examples:**

```bash
yeast up --network-mode user
yeast up --network-mode private
yeast up --network-mode bridge --bridge br0
yeast restart web --network-mode bridge --bridge br0
```

**Current networking limits:**
- Only SSH port forwarding is built in — no custom `8080:80`-style forwarding yet
- No guest IP discovery in bridge mode
- Network mode is not persisted to `yeast.yaml`

---

## Command Reference

| Command | Description |
|---|---|
| `yeast doctor` | Check host readiness (KVM, QEMU, dependencies) |
| `yeast init` | Create a starter `yeast.yaml` in the current directory |
| `yeast pull` | List available images (`--list`) or pull one by slug |
| `yeast up` | Start all VMs defined in the current project |
| `yeast status` | Show tracked VM state |
| `yeast ssh <name>` | Open an interactive SSH session to a running VM |
| `yeast halt [name...]` | Stop selected tracked VMs |
| `yeast down` | Stop all tracked VMs in the current project |
| `yeast restart [name...]` | Restart configured VMs (accepts `--network-mode`) |
| `yeast destroy [name...]` | Stop VMs and remove all local instance data |

Most commands support `--json` for machine-readable output. `yeast ssh` is interactive and does not.

---

## How Yeast Stores Data

**Project directory:**
- `yeast.yaml` — desired VM definitions
- `yeast.state` — runtime state for the current project

**Home directory:**
- `~/.yeast/cache/` — shared base images (reused across projects)
- `~/.yeast/instances/<name>/` — per-instance runtime files

Each instance directory contains:

```
disk.qcow2
seed.iso
user-data
meta-data
vm.log
```

---

## Current Limits

Yeast is intentionally small. The following are out of scope for the current MVP:

- Linux only — no macOS or Windows
- No GUI
- No global VM inventory command
- No arbitrary port-forward configuration (SSH only)
- No configurable disk size
- No snapshots or suspend/resume
- No per-instance `yeast up <name>`
- No guest IP discovery in bridge mode

---

## Read More

- [docs.md](docs.md) — full user guide
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CHANGELOG.md](CHANGELOG.md)
- [LICENSE](LICENSE)
