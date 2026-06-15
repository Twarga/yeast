# Lab 05: Snapshots And Restore

Create a stopped-VM snapshot, change the VM, then restore the clean baseline.

You will learn:

- why snapshots require stopped VMs
- how `yeast snapshot` creates a reset point
- how `yeast snapshots` lists reset points
- how `yeast restore` replaces the current disk state
- why restore is intentionally explicit

## What You Will Build

```text
yeast-lab-05/
└── web VM
    ├── baseline snapshot
    ├── changed guest disk
    └── restored clean disk
```

## Before You Start

Run:

```bash
yeast doctor
```

## Step 1: Create And Start

```bash
mkdir yeast-lab-05
cd yeast-lab-05
yeast init --template ubuntu-basic
yeast up
```

## Step 2: Create A Baseline

Snapshots require the VM to be stopped.

```bash
yeast down
yeast snapshot web baseline --description "Clean baseline"
yeast snapshots web
```

Expected result:

- a snapshot named `baseline`
- metadata showing the snapshot belongs to `web`

## Step 3: Change The VM

```bash
yeast up
yeast exec web -- touch /home/yeast/marker
yeast exec web -- test -e /home/yeast/marker
```

The marker file proves the guest disk has changed after the baseline.

## Step 4: Restore The Baseline

```bash
yeast down
yeast restore web baseline
yeast up
yeast exec web -- test ! -e /home/yeast/marker
```

If the last command succeeds, restore worked.

!!! warning
    Restore replaces the current instance disk with the snapshot disk. Any changes made after the snapshot are removed.

## Step 5: Delete The Snapshot

```bash
yeast down
yeast delete-snapshot web baseline
yeast snapshots web
```

Use this when you no longer need a reset point.

## Clean Up

```bash
yeast destroy
```

## What You Learned

Snapshots are per-instance reset points. They are safest when the VM is stopped because the disk is not changing underneath Yeast.

## Next Lab

Continue with [Multi-VM Private Networking](06-multi-vm-private-networking.md).
