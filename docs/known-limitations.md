# Yeast Known Limitations

This document describes Yeast `v0.7` planning limits and the shipped `v0.6` behavior.

Yeast is intentionally narrow right now. The goal is a reliable local VM core before adding LabsBackery, MCP, cloud, advanced networking, and richer guest-control workflows.

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

Yeast `v0.5` now supports the first narrow private lab network:

- one project-level private lab network
- one static IPv4 per attached instance
- management SSH stays separate from lab traffic
- `yeast status` exposes the configured `LAB IP`

Current limits:

- management still uses QEMU user-mode networking with host port forwarding
- bridge mode config in `yeast.yaml` is not supported
- DHCP for lab guests is not supported
- custom port forwarding rules are not supported
- multiple private networks in one project are not supported
- multi-network lab topologies are not supported
- the current guest-side lab NIC shape is one deterministic interface (`yeastlab0`) only
- deeper guest-to-guest validation is still manual from inside the VMs

## Snapshot Limits

Yeast `v0.4` now supports the first narrow reset loop:

- per-instance snapshot create
- per-instance snapshot list
- per-instance restore
- per-instance snapshot delete
- snapshot metadata tracked in state

Current limits:

- snapshot create is stopped-VM only
- restore is stopped-VM only
- project-wide snapshot/restore is not implemented yet
- snapshot storage is full disk-copy based; there is no deduplication
- live snapshots are not supported
- live restore is not supported
- memory-state snapshots are not supported
- multi-VM atomic reset is not supported

This is enough for the first single-VM reset workflows, but still not enough for full multi-machine classroom lab reset.

## Guest Control Limits

Yeast `v0.6` now supports the first narrow guest-control surface:

- `yeast exec`
- `yeast copy`
- `yeast logs`
- `yeast inspect`

Current limits:

- SSH-backed only
- one selected instance per command
- `copy` is file-oriented only; recursive directory sync is not implemented
- `logs` reads the VM runtime log file and does not stream/follow
- `inspect` is state-based and does not yet expose deeper guest health or service details
- no log streaming/follow mode yet
- no service health checks yet

## Template Limits

Yeast `v0.7` will add the first narrow template surface:

- `yeast init --list-templates`
- `yeast init --template <name-or-path>`
- built-in templates
- local filesystem templates

Current planned limits:

- templates are project starters only
- generated projects are normal editable Yeast projects
- no remote template downloads
- no template registry/search/update workflow
- no complex variable engine
- no hidden provisioning bundles outside normal `yeast.yaml`
- no LabsBackery-specific package format yet

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
