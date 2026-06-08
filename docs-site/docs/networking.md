---
title: Networking
description: VM networking and private lab networks
---

# Networking

This page explains how networking works in Yeast.

## Overview

Yeast provides two types of networking:

1. **Management Network** (SLIRP) - For host-to-VM access
2. **Private Lab Network** - For VM-to-VM communication

## Management Network (SLIRP)

When you start a VM, Yeast creates a management network using QEMU's SLIRP (user networking).

**How it works:**
- SLIRP is a userspace TCP/IP emulator
- It runs inside the QEMU process
- It translates guest packets to real host sockets

**Key properties:**
- No root required
- Works on any Linux system
- Safe (VMs can't be reached from outside unless you forward ports)

### Port Forwarding

Declare port forwards in `yeast.yaml`:

```yaml
instances:
  - name: web
    ports:
      - host_port: 8080
        guest_port: 80
```

## Private Lab Network

For VM-to-VM communication, Yeast uses a private lab network:

```yaml
networks:
  - name: lab
    cidr: 192.168.2.0/24

instances:
  - name: web
    networks:
      - name: lab
        ipv4: 192.168.2.10
```

**How it works:**
- Yeast uses QEMU's socket netdev with multicast
- All VMs on the same network see each other's Layer 2 frames
- The multicast address is derived from project ID and network name

**Key properties:**
- Isolated from host
- No DHCP server (static IPs only)
- No router (flat L2 segment)
- Unique per project

### Static IP Assignment

Assign static IPs in `yeast.yaml`:

```yaml
instances:
  - name: web
    networks:
      - name: lab
        ipv4: 192.168.2.10
```

### VM-to-VM Communication

VMs on the same network can communicate directly:

```bash
yeast exec web -- ping 192.168.2.11
```

## Troubleshooting

### Can't SSH into VM

1. Check if VM is running: `yeast status`
2. Check if SSH port is correct: `ss -tlnp | grep 2222`
3. Check VM logs: `yeast logs web`

### VMs Can't Ping Each Other

1. Check both VMs have the lab NIC
2. Check IP addresses match `yeast.yaml`
3. Check multicast address

### Port Conflict

If `yeast up` fails with "port already in use":

```bash
ss -tlnp | grep 8080
```

## Next Steps

- [Configuration](./configuration) - yeast.yaml reference
- [Provisioning](./provisioning) - Provisioning guide
- [Troubleshooting](./troubleshooting) - Common issues and fixes
