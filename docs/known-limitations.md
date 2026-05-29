# Yeast Known Limitations

This document describes the current Yeast `v1.0` limits.

Yeast is intentionally narrow. The goal of v1.0 is a reliable local VM engine before adding daemon mode, cloud workers, advanced networking, richer guest health checks, and product-specific LabsBakery UI behavior.

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

Yeast `v1.0` supports:

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

Yeast `v1.0` supports the first narrow private lab network:

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
- full smoke coverage validates guest-to-guest TCP reachability, but deeper topology debugging is still manual from inside the VMs

## Snapshot Limits

Yeast `v1.0` supports the first narrow reset loop:

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

Yeast `v1.0` supports the first narrow guest-control surface:

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

Yeast `v1.0` supports the first narrow template surface:

- `yeast init --list-templates`
- `yeast init --template <name-or-path>`
- built-in templates
- local filesystem templates

Current limits:

- templates are project starters only
- generated projects are normal editable Yeast projects
- no remote template downloads
- no template registry/search/update workflow
- no complex variable engine
- no hidden provisioning bundles outside normal `yeast.yaml`
- Yeast can materialize LabsBakery-style local template packages, but LabsBakery package import/export is still a LabsBakery product concern

## UI And Integration Limits

- No daemon.
- No web API.
- No TUI progress view yet.
- First LabsBakery local-engine integration contract is documented.
- No MCP server yet.

The current integration surface is the CLI plus `--json` and `--json --events`.

## Stability Limits

Yeast v1.0 stabilizes the local engine surface documented in the command, config, and JSON references.

Still expected to evolve after v1.0:

- future config fields for features outside the current v1 scope
- state internals, with migrations when needed
- new JSON fields, added compatibly where possible
- new event names for future workflows
- image manifest format

Existing v1 command names, config fields, JSON envelope shape, and documented error codes should not be broken casually.
