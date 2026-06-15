# Snapshots

Yeast snapshots are stopped-VM disk snapshots.

They are useful for creating reset points in local labs.

## Create A Snapshot

```bash
yeast down
yeast snapshot web baseline --description "Clean baseline"
```

## List Snapshots

```bash
yeast snapshots web
```

## Restore

```bash
yeast down
yeast restore web baseline
```

!!! warning
    Restore replaces the current disk with the snapshot disk.

## Delete

```bash
yeast delete-snapshot web baseline
```

## Why VMs Must Be Stopped

Snapshots copy disk state. A running VM can be changing its disk while Yeast is trying to snapshot or restore it.

Stopping first makes the reset point predictable.

## Good Snapshot Names

Use names that describe the reset point:

```bash
yeast snapshot web clean-install
yeast snapshot web before-upgrade
yeast snapshot web baseline
```

## Typical Lab Flow

```bash
yeast up
yeast down
yeast snapshot web baseline --description "Clean baseline"
yeast up
```

Then after experimenting:

```bash
yeast down
yeast restore web baseline
yeast up
```

## What Snapshots Are Not

Snapshots are not project-wide atomic checkpoints in v1.1.

Create and restore snapshots per instance.
