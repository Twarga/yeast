---
title: Tutorial 13 - WireGuard VPN Mesh
description: Encrypted VPN tunnels between VMs
---

# Tutorial 13 - WireGuard VPN Mesh

This walkthrough demonstrates setting up encrypted VPN tunnels between VMs using WireGuard.

## Create the project

```bash
mkdir 13-wireguard-vpn-mesh
cd 13-wireguard-vpn-mesh
yeast init
cp /path/to/yeast/examples/wireguard-vpn-mesh/yeast.yaml ./yeast.yaml
```

## Boot and validate

```bash
yeast pull ubuntu-24.04
yeast up
yeast status
```

## Test the VPN mesh

```bash
yeast exec hub -- wg show
yeast exec spoke1 -- ping -c 2 10.200.0.2
yeast exec spoke2 -- ping -c 2 10.200.0.1
```

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to set up WireGuard VPN tunnels
- How to create a hub-and-spoke VPN mesh
- How to configure encrypted communication between VMs
- How to test VPN connectivity

## Next Steps

- [Tutorial 14 - GitOps/CI Lab](./14-gitops-ci-lab) - Gitea + Drone CI pipeline
