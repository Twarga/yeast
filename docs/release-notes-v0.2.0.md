# Yeast v0.2.0 Release Notes

Status: Draft
Release type: disk size support release
Target platform: Linux amd64

## Summary

Yeast v0.2.0 documents and verifies `disk_size` as a desired-state setting for instance overlay disk creation.

The release keeps the v0.1 local VM lifecycle intact and does not add provisioning, private networking, snapshots, or guest-control features.

## What Changed

- `disk_size` is part of the supported instance config shape.
- Config validation rejects unsupported disk size values before runtime starts.
- Disk sizes are normalized before the app builds runtime plans.
- `yeast up` passes the configured disk size into the runtime disk plan.
- QEMU disk preparation passes the requested size to `qemu-img create` when creating a new overlay disk.
- Existing overlay disks are kept as-is and are not resized by `yeast up`.

## Supported Format

`disk_size` accepts whole-number sizes with optional `K`, `M`, `G`, `T`, or `P` suffixes. A trailing `B` and spaces are accepted and normalized.

Examples:

```yaml
disk_size: 20G
disk_size: 25600M
disk_size: 10737418240
```

## Verification

Automated checks:

```bash
go test ./...
```

Covered behavior:

- config parsing and validation for `disk_size`
- default/normalization behavior
- app-level wiring from config into runtime plans
- QEMU command construction with and without requested size
- existing disk behavior when a requested size is present

Manual host-dependent verification is still recommended before tagging a public release:

```bash
yeast init
yeast pull ubuntu-24.04
yeast up
qemu-img info ~/.yeast/projects/<project-id>/instances/web/disk.qcow2
yeast down
yeast destroy
```

## Not Included

- provisioning packages/files/shell workflows
- snapshots and restore
- private VM-to-VM lab networking
- guest `exec`, `copy`, `logs`, or `inspect`
- disk resizing for already-created overlay disks
