# Yeast

> Fast local VMs for Linux with KVM, QEMU, and cloud-init.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21-cyan.svg)
![Platform](https://img.shields.io/badge/platform-linux-lightgrey.svg)

Yeast is a project-based VM tool for Linux. You describe one or more machines in `yeast.yaml`, pull a trusted base image, run `yeast up`, and Yeast handles the local QEMU/KVM lifecycle for you.

It is built for people who want:
- real VMs, not containers
- a simpler local workflow than older VM tooling
- cloud-init based guest setup
- repeatable terminal-friendly automation

## At A Glance

| Area | What Yeast gives you |
|---|---|
| VM model | Project-local VMs defined in `yeast.yaml` |
| Base images | Shared cache in `~/.yeast/cache/` |
| Provisioning | Cloud-init |
| Runtime | QEMU + KVM |
| Automation | `--json` on major commands |
| Networking | `user`, `private`, `bridge` |
| SSH access | Automatic host port forwarding to guest port `22` |

## Quick Install

## One command

If the repository is reachable over HTTPS:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

If you already cloned the repo:

```bash
bash install.sh
```

What the installer does:
- detects the Linux package manager
- installs Yeast dependencies
- installs Go for the source-build path
- clones and builds Yeast
- installs `yeast` into `/usr/local/bin`
- creates `~/.yeast/cache/`
- generates an SSH key if needed
- adds the user to the `kvm` group when possible

Optional overrides:

```bash
YEAST_REPO_URL=https://github.com/Twarga/yeast.git YEAST_REF=main bash install.sh
```

Important:
- if the repo is private, direct install only works for users who already have access
- if the installer adds you to the `kvm` group, log out and back in before your first `yeast up`

## Build from source

```bash
git clone https://github.com/Twarga/yeast.git
cd yeast
go build -o yeast ./cmd/yeast
sudo mv yeast /usr/local/bin/
```

## Requirements

Yeast currently supports:
- Linux only

Yeast needs:
- KVM support on the host
- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage`
- `ssh`
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

## 5-Minute Quick Start

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
    disk_size: 20G
    user: yeast
    sudo: none
```

You can also generate the starter config with your own values:

```bash
yeast init \
  --name api \
  --image ubuntu-24.04 \
  --memory 2048 \
  --cpus 2 \
  --disk-size 25G \
  --user operator \
  --sudo password
```

### 3. Discover and pull an image

List supported images:

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

### 4. Start the environment

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

### 6. Connect

```bash
yeast ssh web
```

If your bootstrap user is different:

```bash
yeast ssh web --user operator
```

### 7. Stop everything

```bash
yeast down
```

## Example Config

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 2048
    cpus: 2
    disk_size: 25G
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
- `disk_size`
- `user`
- `sudo`
- `env`
- `user_data`

Important behavior:
- `user_data`, if provided, replaces Yeast's generated cloud-init
- custom `user_data` does not automatically merge your SSH key, user, sudo policy, or env values
- `disk_size` creates a larger overlay disk on first boot and grows an existing disk when the configured size increases
- Yeast does not provide reusable cloud-init templates or presets yet; custom provisioning is raw `user_data`

## Networking

Yeast supports 3 network modes on `up` and `restart`:

| Mode | What it does |
|---|---|
| `user` | Default QEMU user-mode NAT with SSH port forwarding |
| `private` | User-mode NAT with `restrict=on` |
| `bridge` | Host bridge attachment plus a restricted management NIC for SSH |

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
- network mode is selected at command time
- network mode is not stored in `yeast.yaml`

## Command Cheat Sheet

| Command | What it does |
|---|---|
| `yeast doctor` | Check whether the host is ready |
| `yeast init` | Create a starter `yeast.yaml` |
| `yeast pull` | List and pull trusted images |
| `yeast up` | Start all VMs in the current project |
| `yeast status` | Show tracked VM state |
| `yeast ssh <name>` | Connect to a running VM |
| `yeast halt [name...]` | Stop selected tracked VMs |
| `yeast down` | Stop all tracked VMs |
| `yeast restart [name...]` | Restart configured VMs |
| `yeast destroy [name...]` | Stop and remove local instance data |

Most major commands support:

```bash
yeast <command> --json
```

`yeast ssh` is interactive and does not support JSON output.

## How Yeast Stores Data

Project directory:
- `yeast.yaml`: desired VM definitions
- `yeast.state`: runtime state for the current project

Home directory:
- `~/.yeast/cache/`: shared base images
- `~/.yeast/instances/<name>/`: per-instance runtime files

Each instance directory typically contains:
- `disk.qcow2`
- `seed.iso`
- `user-data`
- `meta-data`
- `vm.log`

## Current Limits

Yeast is intentionally small right now.

Current non-goals or missing features:
- Linux only
- no GUI
- no global VM inventory command
- no arbitrary port-forward configuration
- no snapshots or suspend/resume
- no per-instance `yeast up <name>`
- no guest IP discovery in bridge mode

## Read More

For the full user guide, see [docs.md](docs.md).

Also useful:
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CHANGELOG.md](CHANGELOG.md)
- [LICENSE](LICENSE)
