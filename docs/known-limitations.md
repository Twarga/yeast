# Yeast Known Limitations

This document describes Yeast `v0.3` limits.

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

Yeast `v0.3` now supports:

- `provision.packages`
- `provision.files`
- `provision.shell`
- automatic provisioning during `yeast up`
- manual reruns with `yeast provision`

Current limits:

- package provisioning is currently tuned for Ubuntu and Debian guests and uses `apt-get`
- there is no service or HTTP health-check stage yet
- file provisioning copies host files one by one; directory sync and templating are not implemented
- shell steps always rerun and must be authored to be safe on repeat execution
- per-step timeout tuning is not exposed in config yet
- no Ansible integration or reusable provisioning bundles yet

Raw `user_data` still fully replaces Yeast-generated cloud-init.

## Networking Limits

Yeast uses QEMU user-mode networking with SSH port forwarding for management.

It does not yet support:

- private VM-to-VM networks
- static lab IPs
- bridge mode config in `yeast.yaml`
- custom port forwarding rules
- multi-network lab topologies

## Snapshot Limits

Snapshots and restore are not implemented yet.

This means Yeast `v0.3` is not yet enough for full resettable cybersecurity labs. LabsBackery needs snapshot/reset support before serious classroom use.

## Guest Control Limits

Yeast supports interactive SSH and provisioning-time SSH automation.

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
