---
title: Known Limitations
description: What Yeast doesn't support yet
---

# Known Limitations

This page documents known limitations in Yeast v1.

## Platform Limitations

### Linux Only

Yeast only runs on Linux. It does not support:
- macOS
- Windows
- WSL2 (experimental)

### x86_64 Only

Yeast only supports x86_64 (AMD64) architecture.

### No Nested Virtualization

Yeast VMs cannot run other VMs inside them.

## Networking Limitations

### Single Network

Yeast v1 supports only one private lab network per project.

### No DHCP

Private networks use static IPs only.

### No Router

Private networks are flat L2 segments.

### No Bridge Mode

Yeast uses SLIRP for management and socket multicast for lab networks.

## VM Limitations

### No Windows Guests

Yeast does not support Windows VMs.

### No GPU Passthrough

Yeast does not support GPU passthrough.

### Limited Resource Allocation

VMs are limited to:
- Maximum 8 GB RAM per VM
- Maximum 4 CPUs per VM
- Maximum 100 GB disk per VM

## Provisioning Limitations

### No Idempotent Shell Commands

Shell commands in `provision` run on every provisioning trigger.

### No Remote File Provisioning

Files must be local to the host.

### No Template Variables

There is no templating system in `yeast.yaml`.

## Snapshot Limitations

### VM Must Be Stopped

You can only take or restore snapshots when the VM is stopped.

### Per-Instance Only

Snapshots are per-instance.

### No Incremental Snapshots

Each snapshot is a full disk copy.

## Lifecycle Limitations

### No Live Migration

VMs cannot be migrated between hosts while running.

### No Auto-Start

VMs do not start automatically on boot.

### No Resource Monitoring

Yeast does not monitor host resources.

## Future Improvements

These limitations may be addressed in future versions:

- Multiple private networks
- DHCP support
- Windows guest support (experimental)
- GPU passthrough
- Live snapshots
- Project-wide snapshots
- Auto-start on boot
- Resource monitoring

## Next Steps

- [Troubleshooting](./troubleshooting) - Common issues and fixes
- [Architecture](./architecture) - How Yeast works under the hood
- [Configuration](./configuration) - yeast.yaml reference
