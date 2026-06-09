---
title: Snapshots
description: Take and restore VM snapshots for safe experimentation
---

# Snapshots

Snapshots let you save the state of a VM and restore it later. They are essential for:

- **Testing safely** — Try changes without risk
- **Lab baselines** — Create reproducible starting points
- **Cybersecurity practice** — Break things, then reset
- **Rollback** — Undo mistakes instantly
- **Sharing** — Distribute reproducible environments

## How Snapshots Work

When you take a snapshot:

1. The VM must be **stopped** (`yeast down`)
2. Yeast copies the current `disk.qcow2` to a snapshot file
3. The snapshot is stored in `~/.yeast/projects/<id>/instances/<name>/snapshots/`

When you restore a snapshot:

1. The VM must be **stopped**
2. Yeast copies the snapshot back to `disk.qcow2`
3. The VM boots from the restored state

**Key properties:**
- Snapshots are **per-instance** (no project-wide atomic snapshots yet)
- Snapshots capture **disk state only** (not network state or memory)
- Snapshots are **full copies** (not incremental)
- Creating a snapshot takes 1-5 seconds for typical lab VMs

## Taking Snapshots

### 1. Stop the VM

```bash
yeast down
```

Snapshots can only be taken when the VM is stopped.

### 2. Take the Snapshot

```bash
yeast snapshot web baseline --description "Clean Ubuntu install with nginx"
```

**Arguments:**
- `web` — Instance name
- `baseline` — Snapshot name (choose something descriptive)
- `--description` — Optional description

**Naming tips:**
- `baseline` — Clean starting state
- `lab1-complete` — After finishing an exercise
- `before-experiment` — Before trying something risky
- `2024-06-08-working` — Date-based for long-term projects

### 3. Verify the Snapshot

```bash
yeast snapshots web
```

Expected output:

```
Snapshots for web
  NAME      DESCRIPTION                              CREATED
  baseline  Clean Ubuntu install with nginx          2024-06-08 14:32
```

## Restoring Snapshots

### 1. Stop the VM

```bash
yeast down
```

### 2. Restore the Snapshot

```bash
yeast restore web baseline
```

**Warning:** This overwrites the current disk state. Any changes since the snapshot will be lost.

### 3. Start the VM

```bash
yeast up
```

The VM boots from the restored state. All files, packages, and configuration from the snapshot are back.

## Deleting Snapshots

Remove a snapshot when you no longer need it:

```bash
yeast delete-snapshot web baseline
```

This frees disk space. The current VM state is unaffected.

## Practical Workflows

### Testing Changes Safely

```bash
# 1. Take a baseline snapshot
yeast down
yeast snapshot web baseline --description "Before changes"

# 2. Start and experiment
yeast up
yeast ssh web
# ... make changes, test ideas ...
exit

# 3. If something breaks, restore
yeast down
yeast restore web baseline
yeast up
# Back to clean state!
```

### Lab Exercise Workflow

For a course with multiple exercises:

```bash
# Exercise 1
yeast up
# ... do exercise 1 ...
yeast down
yeast snapshot web exercise1-done

# Exercise 2
yeast up
# ... do exercise 2 ...
yeast down
yeast snapshot web exercise2-done

# Reset to any exercise
yeast restore web exercise1-done
yeast up
```

### Cybersecurity Lab Reset

```bash
# 1. Setup target VM
yeast up
yeast ssh target
# ... configure vulnerable service ...
exit
yeast down
yeast snapshot target vulnerable --description "Vulnerable service running"

# 2. Attacker practices exploits
yeast up
yeast ssh attacker
# ... run exploits against target ...
exit

# 3. Reset target after attack
yeast down target
yeast restore target vulnerable
yeast up target
# Target is back to vulnerable state!
```

## Best Practices

### 1. Snapshot After Provisioning

Always take a snapshot after successful provisioning:

```bash
yeast up          # Provisioning runs
yeast down        # Stop VM
yeast snapshot web provisioned --description "After initial provisioning"
```

This gives you a clean, working baseline to return to.

### 2. Use Descriptive Names

```bash
# Good
yeast snapshot web nginx-working
yeast snapshot web before-database-migration

# Bad
yeast snapshot web a
yeast snapshot web snap1
```

### 3. Add Descriptions

```bash
yeast snapshot web baseline \
  --description "Ubuntu 24.04 with nginx, PostgreSQL, and Node.js installed"
```

### 4. Clean Up Old Snapshots

Snapshots consume disk space. Periodically review and delete unused ones:

```bash
yeast snapshots web
yeast delete-snapshot web old-snapshot-name
```

### 5. Don't Snapshot Running VMs

Yeast v1 requires VMs to be stopped before snapshotting. If you try to snapshot a running VM:

```bash
yeast snapshot web baseline
# Error: Instance must be stopped before taking a snapshot
```

## Storage Considerations

Each snapshot is a full copy of the disk:

| VM Disk Size | Snapshot Size | Notes |
|---|---|---|
| 2 GB (default) | ~500 MB - 2 GB | Depends on actual data |
| 20 GB | ~2-10 GB | After provisioning |

**Tips for saving space:**
- Use smaller `disk_size` for lab VMs
- Delete snapshots you no longer need
- Use `yeast clean` to remove orphaned resources

## Limitations

Current snapshot limitations in Yeast v1:

- **VM must be stopped** — Can't snapshot running VMs
- **Per-instance only** — No project-wide atomic snapshots
- **Disk only** — Snapshots don't capture network state or running processes
- **Full copies** — Each snapshot is a complete disk copy (not incremental)
- **Local only** — Snapshots stay on the local machine

## Next Steps

- [Commands](./commands) — Complete CLI reference
- [Configuration](./configuration) — yeast.yaml reference
- [Troubleshooting](./troubleshooting) — Common issues and fixes
- [Tutorials](/tutorials/) — Hands-on snapshot exercises
