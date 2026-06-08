---
title: Tutorial 09 - Nodi Home Lab
description: Complex multi-VM architecture with 4 VMs
---

# Tutorial 09 - Nodi Home Lab

This walkthrough demonstrates building a realistic home lab with 4 VMs, shared storage, web services, and cross-VM file sharing.

## Create the project

```bash
mkdir 09-nodi-home-lab
cd 09-nodi-home-lab
yeast init
cp /path/to/yeast/examples/nodi-home-lab/yeast.yaml ./yeast.yaml
```

## Boot and validate

```bash
yeast pull ubuntu-24.04
yeast up
yeast status
```

## Verify the lab

```bash
yeast ssh gateway -- curl -fsS http://192.168.2.50
yeast ssh alpha -- ls /mnt/nfs
yeast ssh beta -- ls /mnt/smb
```

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to set up a 4-VM home lab
- How to configure shared storage (NFS and SMB)
- How to set up web services across VMs
- How to work with cross-VM dependencies

## Next Steps

- [Tutorial 10 - Load Balancer Lab](./10-load-balancer-lab) - Caddy reverse proxy
