---
title: Introduction
description: Linux-first local VM orchestration for QEMU/KVM
---

# Yeast

**Turn a folder into real VMs.**

Yeast is a Linux-first local VM orchestration tool for QEMU/KVM. It lets you define virtual machines in a single YAML file, provision them with cloud-init, and manage their entire lifecycle with simple commands.

## What is Yeast?

Yeast is a command-line tool that creates and manages virtual machines on your Linux machine. Unlike heavy virtualization platforms, Yeast focuses on simplicity and developer experience.

With Yeast, you can:

- Define VMs in a single `yeast.yaml` file
- Boot real Linux machines in seconds
- Provision them with packages, files, and shell commands
- Create private networks for VM-to-VM communication
- Take snapshots and restore to any state
- Automate everything with JSON output and event streams

## Who is Yeast for?

**Developers** who need local VMs for testing, development, or learning Linux.

**System administrators** who want to test configurations before deploying to production.

**Students** who want to learn Linux, networking, or infrastructure without cloud costs.

**Security researchers** who need isolated lab environments for testing.

## Why Yeast?

### Simple YAML Configuration

One `yeast.yaml` file defines your entire lab. No GUI, no complex setup.

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    cpus: 1
    ssh_port: 2222
```

### Real VMs, Not Containers

Yeast runs real virtual machines with QEMU/KVM. You get full hardware virtualization, not shared kernels.

### Cloud-Init Provisioning

VMs are provisioned with cloud-init on first boot. No manual configuration required.

### Snapshots and Restore

Stop VMs, take snapshots, and restore to any state. Perfect for testing and lab environments.

### Private Networking

Create private networks for multi-VM environments. VMs can communicate with each other.

### JSON Automation

Stable JSON output and event streams for scripting and integration with other tools.

## Quick Example

```bash
# Create a new project
mkdir my-lab && cd my-lab
yeast init

# Edit yeast.yaml to define your VMs
vi yeast.yaml

# Pull the base image
yeast pull ubuntu-24.04

# Start the VMs
yeast up

# SSH into a VM
yeast ssh web

# Stop the VMs
yeast down

# Destroy everything
yeast destroy
```

## Next Steps

- [Installation](./installation) - Install Yeast on your system
- [Quickstart](./quickstart) - Get your first VM running in 5 minutes
- [Configuration](./configuration) - Learn about yeast.yaml
- [Tutorials](/tutorials/) - Step-by-step guided labs
