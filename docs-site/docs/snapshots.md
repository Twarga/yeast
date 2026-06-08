---
title: Snapshots
description: Take and restore VM snapshots
---

# Snapshots

This page explains how to use snapshots in Yeast.

## Overview

Snapshots let you save the state of a VM and restore it later. This is useful for:

- Testing changes without risk
- Creating lab baselines
- Rolling back mistakes
- Sharing reproducible environments

## How Snapshots Work

When you take a snapshot:

1. The VM must be stopped
2. Yeast copies the disk image to a snapshot file
3. The snapshot is stored in `~/.yeast/projects/<id>/instances/<name>/snapshots/`

When you restore a snapshot:

1. The VM must be stopped
2. Yeast copies the snapshot back to `disk.qcow2`
3. The VM boots from the restored state

## Taking Snapshots

### Stop the VM

```bash
yeast down
```

### Take the Snapshot

```bash
yeast snapshot web baseline --description "Clean install"
```

### List Snapshots

```bash
yeast snapshots web
```

## Restoring Snapshots

### Stop the VM

```bash
yeast down
```

### Restore the Snapshot

```bash
yeast restore web baseline
```

### Start the VM

```bash
yeast up
```

## Use Cases

### Testing Changes

```bash
# Take a baseline snapshot
yeast down
yeast snapshot web baseline

# Start and test changes
yeast up
yeast ssh web
# ... make changes

# Restore if something breaks
yeast down
yeast restore web baseline
yeast up
```

### Lab Baselines

Create a snapshot for each lab exercise:

```bash
yeast down
yeast snapshot web lab1-complete
yeast up
```

### Full Lab Reset

Reset all VMs to baseline:

```bash
yeast down
yeast restore web baseline
yeast restore db baseline
yeast up
```

## Limitations

- **VM must be stopped** - Can't take snapshots of running VMs
- **Per-instance** - No project-wide atomic snapshots
- **Disk only** - Snapshots don't capture network state

## Best Practices

1. **Snapshot after provisioning** - Capture a clean, working state
2. **Use descriptive names** - `baseline`, `lab1-complete`, `before-update`
3. **Add descriptions** - Document what the snapshot contains

## Next Steps

- [Commands](./commands) - Snapshot commands
- [Configuration](./configuration) - yeast.yaml reference
- [Tutorials](/tutorials/) - Hands-on snapshot exercises
