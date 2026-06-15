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
