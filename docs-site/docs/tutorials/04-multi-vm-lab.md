---
title: Tutorial 04 - Multi-VM Lab
description: Multiple VMs with private networking
---

# Tutorial 04 - Multi-VM Lab

This walkthrough demonstrates setting up multiple VMs with private networking.

## Create the project

```bash
mkdir 04-multi-vm-lab
cd 04-multi-vm-lab
yeast init
cp /path/to/yeast/examples/two-vm-lab/yeast.yaml ./yeast.yaml
```

## Boot both machines

```bash
yeast up
yeast status
```

## Verify the private network

```bash
yeast ssh attacker -- bash -lc 'ip -4 addr show yeastlab0 && ping -c 2 10.10.10.20'
yeast ssh target -- bash -lc 'ip -4 addr show yeastlab0 && ping -c 2 10.10.10.10'
```

Expected result:

- `yeast status` shows `LAB IP` values
- both guests can ping each other on the lab network

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to define multiple VMs in yeast.yaml
- How to set up private networking between VMs
- How to assign static IPs to VMs
- How VMs can communicate on the lab network

## Next Steps

- [Tutorial 05 - Guest Control](./05-guest-control) - exec, copy, logs, inspect
