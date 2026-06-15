# Yeast

Yeast turns a folder into real Linux VMs.

Define machines in `yeast.yaml`, run `yeast up`, and get SSH-ready QEMU/KVM guests with cloud-init, provisioning, snapshots, private networking, and JSON output for automation.

## Start Here

If you are new to Yeast, follow this path:

1. [What Is Yeast?](getting-started/what-is-yeast.md)
2. [Installation](getting-started/installation.md)
3. [Quickstart](getting-started/quickstart.md)
4. [First VM](getting-started/first-vm.md)
5. [Yeast Labs](labs/index.md)

## The Short Version

```bash
mkdir my-lab
cd my-lab
yeast init --template ubuntu-basic
yeast pull ubuntu-24.04
yeast up
yeast ssh web
```

When you are done:

```bash
yeast down
yeast destroy
```

## What Yeast Handles

- project-local VM definitions with `yeast.yaml`
- trusted image discovery and caching
- QEMU/KVM startup
- cloud-init seed generation
- SSH readiness
- package, file, and shell provisioning
- stopped-VM snapshots and restore
- one private lab network per project
- guest control with `ssh`, `exec`, `copy`, `logs`, and `inspect`
- stable JSON output and lifecycle events

## What Yeast Is Not

Yeast is not a cloud platform, container orchestrator, web dashboard, or replacement for Kubernetes.

It is a Linux-first local VM engine for repeatable real-machine labs.

## Current Release

The current docs target Yeast `v1.1.0`.

Start with [Quickstart](getting-started/quickstart.md), or jump to the [command reference](reference/commands.md).
