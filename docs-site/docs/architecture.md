---
title: Architecture
description: How Yeast works under the hood
---

# Architecture Overview

This page explains how Yeast works at a high level.

## Mental Model

Yeast is a **desired state** system for local VMs. You declare what you want in `yeast.yaml`, and Yeast makes it happen.

```
yeast.yaml (desired state)
    ↓
    Yeast
    ↓
QEMU/KVM (actual state)
```

## Components

### Project Identity

Every Yeast project has a unique ID stored in `.yeast/project.json`.

### Image Cache

Base images are cached at `~/.yeast/cache/images/`.

### Instance Disks

Each VM gets a copy-on-write (QCOW2) overlay disk.

### Cloud-Init

Yeast uses cloud-init for first-boot provisioning.

### QEMU Runtime

Yeast starts QEMU with KVM acceleration and two network interfaces.

### State Management

Yeast tracks runtime state in `state.json`.

## Lifecycle

### Startup Flow

1. Load `.yeast/project.json`
2. Load `yeast.yaml`
3. Validate configuration
4. For each instance:
   - Check/create disk.qcow2
   - Generate cloud-init files
   - Build seed.iso
   - Start QEMU
   - Wait for SSH
   - Run provisioning

### Shutdown Flow

1. Load state.json
2. For each instance:
   - Send ACPI shutdown
   - Wait for process to exit
   - Update state

### Destroy Flow

1. Stop all instances
2. For each instance:
   - Remove disk.qcow2
   - Remove seed.iso
   - Remove state
   - Remove snapshots

## Storage Layout

```
~/.yeast/
  cache/
    images/
      ubuntu-24.04/
        disk.qcow2
  projects/
    <project-id>/
      .yeast/
        project.json
      yeast.yaml
      instances/
        <name>/
          disk.qcow2
          seed.iso
          state.json
          snapshots/
```

## Next Steps

- [Configuration](./configuration) - yeast.yaml reference
- [Commands](./commands) - CLI command reference
- [Troubleshooting](./troubleshooting) - Common issues and fixes
