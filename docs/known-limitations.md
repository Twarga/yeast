# Yeast Known Limitations

This document describes Yeast v0.1 limits.

Yeast is intentionally narrow right now. The goal is a reliable local VM core before adding LabsBackery, MCP, cloud, snapshots, and advanced provisioning.

## Platform Limits

- Linux host only.
- QEMU/KVM only.
- No VirtualBox backend.
- No Windows host support.
- No macOS host support.
- No remote worker mode.

## VM Lifecycle Limits

- `yeast up` starts all configured project instances.
- There is no per-instance `up` command yet.
- `yeast down` stops tracked instances but does not delete disks.
- `yeast destroy` removes tracked runtime data but not the shared image cache.
- There is no restart command yet.

## Image Limits

- Only built-in trusted images are supported.
- The manifest is compiled into Yeast.
- Custom image registries are not supported yet.
- Image updates require a code/manifest change.

Current images:

- `ubuntu-22.04`
- `ubuntu-24.04`

## Provisioning Limits

Yeast v0.1 uses cloud-init for base bootstrap.

It does not yet provide:

- `provision.packages`
- `provision.files`
- `provision.shell`
- `yeast provision`
- Ansible integration
- reusable provisioning bundles

Raw `user_data` exists, but it fully replaces Yeast-generated cloud-init.

## Networking Limits

Yeast v0.1 uses QEMU user-mode networking with SSH port forwarding for management.

It does not yet support:

- private VM-to-VM networks
- static lab IPs
- bridge mode config in `yeast.yaml`
- custom port forwarding rules
- multi-network lab topologies

## Snapshot Limits

Snapshots and restore are not implemented yet.

This means Yeast v0.1 is not yet enough for full resettable cybersecurity labs. LabsBackery needs snapshot/reset support before serious classroom use.

## Guest Control Limits

Yeast v0.1 supports interactive SSH.

It does not yet support:

- `yeast exec`
- `yeast copy`
- `yeast logs`
- `yeast inspect`
- service health checks

These are required later for Yeast MCP and richer LabsBackery workflows.

## UI And Integration Limits

- No daemon.
- No web API.
- No TUI progress view yet.
- No LabsBackery integration contract yet.
- No MCP server yet.

The current integration surface is the CLI plus `--json`.

## Stability Limits

The project is still pre-1.0.

Expected to change before v1:

- config schema details
- state schema details
- JSON schemas
- error codes
- image manifest format

The goal is to stabilize these before `v1.0`.
