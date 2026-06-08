---
title: Commands
description: Complete CLI command reference
---

# Command Reference

This page documents all Yeast CLI commands.

## Overview

Yeast commands follow this pattern:

```bash
yeast <command> [arguments] [flags]
```

## Core Commands

### yeast init

Initialize a new Yeast project.

```bash
yeast init
```

### yeast pull

Download a base image to the local cache.

```bash
yeast pull <image>
```

### yeast up

Start all VMs in the project.

```bash
yeast up [--reprovision] [--json]
```

### yeast down

Stop all VMs in the project.

```bash
yeast down [--json]
```

### yeast destroy

Remove all project resources.

```bash
yeast destroy [--json]
```

## Access Commands

### yeast ssh

SSH into a running VM.

```bash
yeast ssh <instance> [--command <cmd>]
```

### yeast exec

Execute a command in a VM without opening an interactive session.

```bash
yeast exec <instance> -- <command>
```

### yeast copy

Copy files between host and VM.

```bash
yeast copy <instance>:<path> <local-path>
yeast copy <local-path> <instance>:<path>
```

## Status Commands

### yeast status

Show the status of all VMs in the project.

```bash
yeast status [--json]
```

### yeast inspect

Show detailed information about a specific VM.

```bash
yeast inspect <instance> [--json]
```

### yeast logs

Show logs for a VM.

```bash
yeast logs <instance> [--tail <lines>]
```

## Snapshot Commands

### yeast snapshot

Create a snapshot of a VM.

```bash
yeast snapshot <instance> <name> [--description <desc>]
```

### yeast snapshots

List snapshots for an instance.

```bash
yeast snapshots <instance>
```

### yeast restore

Restore an instance from a snapshot.

```bash
yeast restore <instance> <name>
```

## Utility Commands

### yeast doctor

Check system requirements and diagnose issues.

```bash
yeast doctor
```

### yeast clean

Clean up stale state and orphaned resources.

```bash
yeast clean
```

### yeast version

Show the Yeast version.

```bash
yeast version
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output JSON instead of human-readable text |
| `--help`, `-h` | Show help for a command |
| `--version`, `-v` | Show version |

## JSON Output

All commands support `--json` flag for machine-readable output:

```bash
yeast status --json
```

## Next Steps

- [Configuration](./configuration) - yeast.yaml reference
- [Troubleshooting](./troubleshooting) - Common issues and fixes
- [Tutorials](/tutorials/) - Step-by-step guided labs
