---
title: Introduction
description: What is Yeast and why it exists
---

# Introduction to Yeast

**Turn a folder into real VMs.**

Yeast is a Linux-first local VM orchestration tool for QEMU/KVM. It lets you define virtual machines in a single YAML file, provision them with cloud-init, and manage their entire lifecycle with simple commands.

## What Problem Does Yeast Solve?

Running local VMs on Linux is still more painful than it should be. The typical workflow means stitching together too many manual steps:

1. Downloading cloud images manually
2. Creating qcow2 disk images with the right size
3. Generating cloud-init configuration files
4. Building a seed ISO from those files
5. Composing complex QEMU command-line arguments
6. Tracking which SSH ports map to which VMs
7. Remembering which runtime files belong to which project

Yeast reduces all of this to a **project workflow** instead of a pile of ad hoc commands.

## What is Yeast?

Yeast is a command-line tool that creates and manages virtual machines on your Linux machine. Unlike heavy virtualization platforms, Yeast focuses on simplicity and developer experience.

### Core Workflow

```bash
# 1. Create a project
mkdir my-lab && cd my-lab
yeast init

# 2. Edit yeast.yaml to define your VMs
vi yeast.yaml

# 3. Pull a base image
yeast pull ubuntu-24.04

# 4. Start the VMs
yeast up

# 5. SSH into a VM
yeast ssh web

# 6. Stop the VMs
yeast down

# 7. Destroy everything
yeast destroy
```

## Key Features

| Feature | What It Means |
|---|---|
| **YAML-Driven** | One `yeast.yaml` file defines your entire lab. No GUI, no clicks. |
| **Real VMs** | Full hardware virtualization with QEMU/KVM. Not containers. Real Linux machines. |
| **Cloud-Init Provisioning** | VMs are provisioned automatically on first boot. No manual configuration. |
| **Snapshots** | Stop VMs, take snapshots, restore to any state. Safe lab reset in seconds. |
| **Private Networking** | Create private lab networks for multi-VM environments. VMs communicate natively. |
| **Guest Control** | Execute commands, copy files, and inspect VMs without leaving your terminal. |
| **JSON Automation** | Stable JSON output and event streams for scripting and CI/CD integration. |
| **Templates** | Start from built-in templates or create your own reusable project starters. |

## Who is Yeast For?

### Developers
Need local VMs for testing, development, or learning Linux? Yeast gives you reproducible environments in seconds.

### System Administrators
Want to test configurations before deploying to production? Yeast lets you validate changes safely.

### Students
Learning Linux, networking, or infrastructure without cloud costs? Yeast runs everything locally.

### Security Researchers
Need isolated lab environments for testing? Yeast's snapshot and reset workflow is perfect for cybersecurity labs.

### DevOps Engineers
Want to automate VM workflows? Yeast's stable JSON and event streams integrate with your tooling.

## Why Yeast Over Alternatives?

### vs. Vagrant
- No VirtualBox dependency (uses native QEMU/KVM)
- Linux-first (not a multi-platform compromise)
- Simpler project model
- Better JSON automation

### vs. Docker
- Real VMs, not containers
- Full kernel isolation
- Better for system-level testing
- Networking that's closer to production

### vs. Manual QEMU
- Project-scoped state management
- Automatic cloud-init generation
- Built-in SSH port management
- Snapshot and restore workflows

### vs. Cloud VMs
- Zero cost
- No network latency
- Works offline
- Faster iteration cycles

## The Yeast Philosophy

> Real infrastructure should be understandable before it becomes scalable.

Yeast hides repetitive pain, but not the mental model:

- You still know there's a project
- You still see the config that defines machines
- You still understand that QEMU/KVM runs the VMs
- You still see that cloud-init prepares the guest

Yeast should feel **simple**, but not **magical in a confusing way**.

## What's Next?

- [Installation](./installation) — Get Yeast running on your system
- [Quickstart](./quickstart) — Your first VM in 5 minutes
- [Configuration](./configuration) — The complete yeast.yaml reference
- [Architecture](./architecture) — How Yeast works under the hood
