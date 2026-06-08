---
title: Tutorial 01 - Your First VM
description: Learn the basics of creating, starting, and managing a VM
---

# Tutorial 01 - Your First VM

This walkthrough uses a simple Ubuntu VM configuration.

## Create the project

```bash
mkdir 01-first-vm
cd 01-first-vm
yeast init
cp /path/to/yeast/examples/ubuntu-basic/yeast.yaml ./yeast.yaml
```

## Run the lifecycle

```bash
yeast doctor
yeast pull ubuntu-24.04
yeast up
yeast status
yeast ssh web
```

Inside the guest, verify the VM is real:

```bash
uname -a
exit
```

## Stop and remove it

```bash
yeast down
yeast destroy
```

Expected result:

- `yeast up` reports one running VM
- `yeast ssh web` opens a working shell as `yeast`
- `yeast down` and `yeast destroy` both complete without orphaning QEMU

## What You Learned

- How to initialize a Yeast project
- How to pull a base image
- How to start, access, and stop a VM
- How to clean up resources

## Next Steps

- [Tutorial 02 - Provisioning](./02-provisioning) - Install packages and configure VMs
