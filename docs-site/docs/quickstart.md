---
title: Quickstart
description: Get your first VM running in 5 minutes
---

# Quickstart

Get your first Yeast VM running in 5 minutes.

## Create a Project

First, create a directory for your project:

```bash
mkdir my-lab && cd my-lab
```

Initialize a new Yeast project:

```bash
yeast init
```

This creates:
- `.yeast/project.json` - Project identity file
- `yeast.yaml` - Default configuration

## Configure Your VM

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
- Ubuntu 24.04 base image
- 512 MB RAM
- 1 CPU
- SSH access on port 2222

## Pull the Base Image

Download the Ubuntu cloud image:

```bash
yeast pull ubuntu-24.04
```

This downloads the image to `~/.yeast/cache/images/`. It only runs once - subsequent projects reuse the cached image.

## Start the VM

Boot the virtual machine:

```bash
yeast up
```

First boot takes 30-60 seconds. Yeast will:
1. Create a disk image
2. Generate cloud-init configuration
3. Start QEMU with KVM acceleration
4. Wait for SSH to become available

## SSH into the VM

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

# Install a package
sudo apt update && sudo apt install -y nginx

# Check the service
systemctl status nginx

# Exit the VM
exit
```

## Check Status

From your host, check the VM status:

```bash
yeast status
```

## Stop the VM

Stop the VM (preserves the disk):

```bash
yeast down
```

## Destroy Everything

Remove all VMs and project files:

```bash
yeast destroy
```

## What You Learned

| Concept | Command |
|---------|---------|
| Initialize project | `yeast init` |
| Pull base image | `yeast pull ubuntu-24.04` |
| Start VMs | `yeast up` |
| SSH into VM | `yeast ssh web` |
| Check status | `yeast status` |
| Stop VMs | `yeast down` |
| Destroy project | `yeast destroy` |

## Next Steps

- [Configuration](./configuration) - Learn about all yeast.yaml options
- [Provisioning](./provisioning) - Install packages and configure VMs
- [Networking](./networking) - Create private networks
- [Snapshots](./snapshots) - Take and restore snapshots
- [Tutorials](/tutorials/) - Step-by-step guided labs
