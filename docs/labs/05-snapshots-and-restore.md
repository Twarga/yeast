# Lab 05: Snapshots And Restore

In this lab, you will create a stopped-VM snapshot and restore it.

You will learn:

- `yeast snapshot`
- `yeast snapshots`
- `yeast restore`
- why snapshots require stopped VMs

## Create And Start

```bash
mkdir yeast-lab-05
cd yeast-lab-05
yeast init --template ubuntu-basic
yeast up
```

## Create A Baseline

Snapshots require the VM to be stopped.

```bash
yeast down
yeast snapshot web baseline --description "Clean baseline"
yeast snapshots web
```

## Change The VM

```bash
yeast up
yeast exec web -- touch /home/yeast/marker
yeast exec web -- test -e /home/yeast/marker
```

## Restore The Baseline

```bash
yeast down
yeast restore web baseline
yeast up
yeast exec web -- test ! -e /home/yeast/marker
```

If the last command succeeds, restore worked.

!!! warning
    Restore replaces the current disk state with the snapshot disk state.

## Clean Up

```bash
yeast down
yeast destroy
```

Next: [Multi-VM Private Networking](06-multi-vm-private-networking.md).
