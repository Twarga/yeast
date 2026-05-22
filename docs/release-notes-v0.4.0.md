# Yeast v0.4.0 Release Notes

Status: Draft
Release type: Minor feature release
Target platform: Linux amd64

## Summary

Yeast v0.4.0 adds the first narrow snapshot and reset workflow to the local VM engine.

It keeps the product intentionally conservative. Snapshot create and restore are stopped-VM only, per-instance only, and disk-copy based. That is deliberate. The goal of this release is not to solve every lab-reset problem. The goal is to make one reliable loop real:

- provision a useful VM
- stop it
- snapshot it
- break it later
- stop it
- restore it
- boot it again

This release adds:

- `yeast snapshot <instance> <name>`
- `yeast snapshots <instance>`
- `yeast restore <instance> <name>`
- `yeast delete-snapshot <instance> <name>`
- snapshot metadata persisted in project state
- snapshot runtime file-copy helpers for the QEMU backend
- single-VM reset docs and smoke coverage

This release does **not** add project-wide reset, live snapshots, live restore, private networking, guest control commands, LabsBackery integration, MCP integration, or cloud worker behavior.

## What Changed

### Snapshot Contract

`v0.4` snapshot behavior is intentionally narrow:

- snapshot create is stopped-VM only
- restore is stopped-VM only
- snapshot scope is one instance at a time
- snapshot storage is a copied qcow2 disk file
- metadata is tracked in project state

Snapshot metadata includes:

- `name`
- `created_at`
- `description`
- `disk_path`
- `source_disk_size`

### New Snapshot Commands

This release adds:

```bash
yeast snapshot <instance> <name> --description "..."
yeast snapshots <instance>
yeast restore <instance> <name>
yeast delete-snapshot <instance> <name>
```

These commands reuse the existing human renderer and JSON envelope behavior.

### Reset Workflow

The first supported reset flow is:

```text
up -> provision -> down -> snapshot -> up -> break -> down -> restore -> up
```

This is enough for single-VM resettable demos and the first lab-like workflows, but not for multi-VM atomic reset.

### Docs And Example

This release updates:

- `examples/caddy-single-vm`
- `docs/quickstart.md`
- `docs/known-limitations.md`
- `README.md`

The Caddy example now shows the first real stopped-VM reset workflow.

## Verification

### Automated

The following checks should pass before release:

```bash
go test ./... -count=1
git diff --check
bash -n scripts/manual-smoke.sh
```

### Smoke Validation

The smoke script now supports:

```bash
TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=positive ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

The positive/full path now covers:

- `doctor`
- `init`
- `pull`
- `up`
- provisioning verification
- `yeast provision`
- `down`
- `yeast snapshot`
- `yeast snapshots`
- guest break step
- `yeast restore`
- restored content verification
- `yeast delete-snapshot`
- `destroy`

On the current Linux/KVM host validation path, the post-restore boot may need a different host `ssh_port` if the previous forwarded port is still settling after a recent stop. The smoke script handles that explicitly so the snapshot workflow itself is still validated.

The negative path still covers the existing JSON error-contract checks for invalid config, unsupported images, and broken project state.

## Not Included

- project-wide snapshot/restore
- live snapshots
- live restore
- memory-state snapshots
- deduplicated snapshot storage
- private VM-to-VM networking
- guest `exec`, `copy`, `logs`, or `inspect`
- template workflows
- LabsBackery runtime contract
- Yeast MCP contract
- Twarga Cloud worker mode
