# Yeast v1.0.0 Release Notes

Status: Prepared for release

Release date: 2026-05-29

## Summary

Yeast v1.0.0 is the first stable local engine release.

This release stabilizes the complete local VM loop that Yeast has been building toward: project-based QEMU/KVM machines, trusted cloud images, cloud-init bootstrap, post-boot provisioning, stopped-VM snapshot and restore, one private lab network, SSH-backed guest control, built-in/local templates, versioned JSON output, lifecycle events, and the first LabsBakery local-engine contract.

Yeast v1.0.0 does not turn Yeast into a cloud platform, web UI, or full lab product. It freezes the local engine that those products can now build on.

## Install

Recommended install command:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

Explicit v1.0.0 install after the tag is published:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | \
  YEAST_REF=v1.0.0 YEAST_EXPECTED_VERSION=v1.0.0 bash
```

Verify:

```bash
yeast version
yeast doctor
```

Expected version:

```text
v1.0.0
```

## Upgrade

Run the installer again:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

The installer overwrites the `yeast` binary at `YEAST_INSTALL_DIR/yeast`. It does not delete project directories, cached images, disks, snapshots, or state under `~/.yeast`.

If the installer adds your user to the `kvm` group, log out and back in before running `yeast up`.

## Stable v1 Surface

Yeast v1.0.0 stabilizes these user-facing surfaces:

- core CLI commands documented in `docs/command-reference.md`
- `yeast.yaml` fields documented in `docs/config-reference.md`
- JSON envelope shape with `schema_version: "yeast.v1"`
- documented command JSON fields
- documented error codes
- JSON Lines lifecycle events for supported workflows
- local template behavior
- LabsBakery local-engine integration boundary

## Features Included

### Project Lifecycle

- `yeast doctor`
- `yeast init`
- `yeast pull`
- `yeast up`
- `yeast status`
- `yeast ssh`
- `yeast down`
- `yeast destroy`
- project-scoped runtime directories
- project identity metadata
- state locking and reconciliation

### Images And Disks

- trusted built-in image manifest
- shared image cache under `~/.yeast/cache/images`
- SHA-256 verification for downloaded images
- per-instance qcow2 disks
- configurable `disk_size`

### Provisioning

- project-level and instance-level provisioning
- package provisioning
- file provisioning
- shell provisioning
- automatic provisioning during `yeast up`
- manual reruns with `yeast provision`
- provisioning status and logs

### Snapshots And Reset

- `yeast snapshot`
- `yeast snapshots`
- `yeast restore`
- `yeast delete-snapshot`
- stopped-VM baseline workflow
- snapshot metadata in state

### Private Lab Networking

- one project-level private lab network
- static per-instance lab IPv4
- separate host-forwarded SSH management path
- `LAB IP` in human status output
- lab IPs in JSON status output

### Guest Control

- `yeast exec`
- `yeast copy --to-guest`
- `yeast copy --from-guest`
- `yeast logs`
- `yeast inspect`
- structured JSON command results

### Templates

- `yeast init --list-templates`
- `yeast init --template <name-or-path>`
- built-in `ubuntu-basic`
- built-in `caddy-single-vm`
- built-in `two-vm-lab`
- local filesystem templates

### Automation Contract

- `schema_version: "yeast.v1"` on JSON success and error envelopes
- stable error codes
- stable documented JSON fields
- JSON Lines event streams for long-running workflows
- human output and JSON output kept separate

### LabsBakery Foundation

- LabsBakery integration contract
- LabsBakery lab package convention
- first attacker/target example package
- browser-terminal-friendly status metadata
- local session directory model

## Compatibility

Yeast v1.0.0 is Linux-first and QEMU/KVM-first.

Required host basics:

- Linux
- `/dev/kvm`
- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage`, `mkisofs`, or compatible ISO builder
- `ssh`
- SSH public key

Supported built-in images:

- `ubuntu-22.04`
- `ubuntu-24.04`

Supported CPU architectures for the installer:

- `amd64`
- `arm64`

## Known Limitations

Yeast v1.0.0 intentionally does not include:

- Windows host support
- macOS host support
- VirtualBox backend
- libvirt backend
- daemon mode
- web API
- remote workers
- Twarga Cloud features
- LabsBakery web UI
- Yeast MCP server
- remote template registry
- packaged `.lbz` import/export
- project-wide atomic snapshot/reset helper
- live snapshots
- live restore
- memory-state snapshots
- multiple private lab networks
- bridge mode
- DHCP lab guests
- recursive directory sync
- log streaming/follow mode
- built-in service health-check DSL

These are future product layers or later engine expansions. They are not part of the v1.0 local engine contract.

## Verification

The v1.0.0 release candidate passed:

- release artifact build with version injection
- checksum verification
- full Go test suite
- shell syntax checks
- whitespace diff check
- `go vet`
- `golangci-lint`
- `gosec`
- installer path validation through a non-system install harness
- full real-host KVM smoke test

The full real-host smoke covered:

- built-in template list
- Caddy template initialization
- LabsBakery attacker/target package materialization
- Ubuntu image pull
- QEMU/KVM VM boot
- provisioning to `provisioned`
- SSH readiness
- Caddy service check
- guest HTTP content check
- `exec`
- `copy` to guest
- `copy` from guest
- `inspect`
- `logs`
- provisioning rerun
- down/up lifecycle
- snapshot create/list/delete
- break-and-restore reset flow
- two-VM private lab boot
- static lab IP reporting
- guest-side lab NIC checks
- guest-to-guest TCP reachability
- destroy cleanup
- negative JSON error cases

See `docs/release-checklist-v1.0.0.md` for the detailed validation record.

## Migration Notes

Projects created during the v0.x line should continue to work if they use the documented v1 config fields.

Recommended before upgrading important projects:

```bash
yeast down
yeast status --json
```

Then upgrade the binary and run:

```bash
yeast doctor
yeast status
```

If a project has hand-edited state files or undocumented config fields, bring it back to the documented `yeast.yaml` surface before treating it as a v1 project.

## What Comes Next

After v1.0.0, the next work should build on the stable local engine instead of changing it casually:

- LabsBakery implementation using Yeast as the VM engine
- Yeast MCP using the JSON/guest-control surface
- project-wide reset helpers if LabsBakery needs them
- richer templates and lab packages
- more networking modes when real lab requirements justify them
