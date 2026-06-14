---
title: Quickstart
description: Get your first VM running in 5 minutes
---

# Quickstart

Get your first Yeast VM running in 5 minutes.

## Prerequisites

Before starting, make sure you have:

- [Yeast installed](./installation)
- KVM support enabled (`yeast doctor` should pass)
- An SSH key (`~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`)

## Step 1: Create a Project

First, create a directory for your project:

```bash
mkdir my-lab && cd my-lab
```

Initialize a new Yeast project:

```bash
yeast init
```

This creates:
- `.yeast/project.json` — Project identity file with a unique ID
- `yeast.yaml` — Default configuration

## Step 2: Configure Your VM

Edit `yeast.yaml` with your preferred editor:

```yaml
version: 1
instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    memory: 512
    cpus: 1
    ssh_port: 2222
    user: yeast
    sudo: nopasswd
```

This defines a single VM named `web` with:
- **Ubuntu 24.04** base image
- **512 MB** RAM
- **1 CPU**
- **SSH access** on port 2222
- **User** `yeast` with passwordless sudo

### What's in this config?

| Field | Description |
|---|---|
| `name` | The instance name. Must be unique within the project. |
| `hostname` | The guest OS hostname. |
| `image` | The base image to use. Must be pulled first. |
| `memory` | RAM in megabytes. |
| `cpus` | Number of virtual CPUs. |
| `ssh_port` | Host port for SSH access. Must be unique across all instances. |
| `user` | Username created by cloud-init. |
| `sudo` | Sudo policy. `nopasswd` allows sudo without password. |

## Step 3: Review or Pre-Pull the Base Image

List available images:

```bash
yeast pull --list
```

Optionally pre-download the Ubuntu cloud image:

```bash
yeast pull ubuntu-24.04
```

This downloads the image to `~/.yeast/cache/images/`. It only runs once — subsequent projects reuse the cached image. If you skip this step, `yeast up` can auto-pull supported cloud images. Images marked manual in `yeast pull --list` print setup instructions instead of downloading automatically.

```
Pulling ubuntu-24.04...
Image downloaded successfully.
```

## Step 4: Start the VM

Boot the virtual machine:

```bash
yeast up
```

First boot can take 30-90 seconds depending on the host, image cache, and cloud-init. Yeast will:
1. Create a disk image from the base image
2. Generate cloud-init configuration (user, SSH key, hostname)
3. Build a seed ISO with cloud-init files
4. Start QEMU with KVM acceleration
5. Wait for SSH to become available
6. Run any configured provisioning steps

Expected output:

```
All instances ready
  NAME  STATUS   SSH             LAB IP
  web   running  127.0.0.1:2222
```

## Step 5: SSH into the VM

Connect to your VM:

```bash
yeast ssh web
```

You're now inside a real Ubuntu VM! Try some commands:

```bash
# Check the hostname
hostname

# Check the IP address
ip addr show

# Check if you're the yeast user
whoami

# Install a package
sudo apt update && sudo apt install -y curl

# Check the service
systemctl status ssh

# Exit the VM
exit
```

## Step 6: Check Status

From your host, check the VM status:

```bash
yeast status
```

Expected output:

```
Project status
  NAME  STATUS   SSH             LAB IP
  web   running  127.0.0.1:2222
```

## Step 7: Stop the VM

Stop the VM (preserves the disk for later):

```bash
yeast down
```

The VM stops but the project files remain. You can start it again with `yeast up`.

## Step 8: Destroy Everything

Remove all VMs and project files:

```bash
yeast destroy
```

**Warning:** This permanently deletes VMs, disks, and snapshots. It cannot be undone.

## What You Learned

| Concept | Command |
|---|---|
| Initialize project | `yeast init` |
| List images | `yeast pull --list` |
| Pre-pull base image | `yeast pull ubuntu-24.04` |
| Start VMs | `yeast up` |
| SSH into VM | `yeast ssh web` |
| Check status | `yeast status` |
| Stop VMs | `yeast down` |
| Destroy project | `yeast destroy` |

## Next Steps

- [Configuration](./configuration) — Learn about all `yeast.yaml` options
- [Provisioning](./provisioning) — Install packages and configure VMs automatically
- [Networking](./networking) — Create private networks for multi-VM labs
- [Snapshots](./snapshots) — Take and restore VM snapshots
- [Tutorials](/tutorials/) — Step-by-step guided labs

## Common Issues

### "SSH timeout" during `yeast up`

Wait longer or check VM logs:

```bash
yeast logs web
```

### "port 2222 is already in use"

Change the `ssh_port` in `yeast.yaml` to another port (e.g., 2223).

### "KVM not available"

Check [Troubleshooting](./troubleshooting) for KVM setup.
