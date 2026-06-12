---
title: Introduction
description: What is Yeast and why it exists
---

# Introduction to Yeast

Yeast is a Linux-first local VM orchestration tool for QEMU/KVM. It lets you define virtual machines in a single YAML file, provision them with cloud-init, and manage their lifecycle with simple commands.

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

## Core Workflow

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

## Key Ideas

- You always work inside a project directory
- `yeast.yaml` is the source of truth
- QEMU/KVM runs the VMs locally
- cloud-init prepares the guest on first boot
- Yeast keeps runtime state and project data separate

## What to Read Next

- [Installation](./installation) — Get Yeast running on your system
- [Quickstart](./quickstart) — Your first VM in 5 minutes
- [Configuration](./configuration) — The complete yeast.yaml reference
- [Architecture](./architecture) — How Yeast works under the hood
