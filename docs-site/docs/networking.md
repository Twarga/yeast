---
title: Networking
description: VM networking and private lab networks explained
---

# Networking

This page explains how networking works in Yeast — how VMs communicate with your host, with each other, and with the internet.

## Overview

Yeast provides **two types** of networking:

| Type | Purpose | Visibility |
|---|---|---|
| **Management Network (SLIRP)** | Host ↔ VM access | Host can reach VM via forwarded ports |
| **Private Lab Network** | VM ↔ VM communication | Isolated from host and internet |

```
┌─────────────────────────────────────────────────────────────┐
│                         YOUR LAPTOP                          │
│                                                              │
│   ┌──────────────┐    SSH Port    ┌──────────────┐          │
│   │   Host OS    │◄──────────────►│     web      │          │
│   │              │    2222       │  127.0.0.1   │          │
│   │  Port 8080   │◄──────────────►│  Port 80     │          │
│   └──────────────┘                └──────────────┘          │
│         ▲                              │                     │
│         │      ┌───────────────────────┘                     │
│         │      │                                              │
│         │      ▼                                              │
│         │   ┌──────────────┐                                │
│         └──►│     db       │                                │
│             │  192.168.2.20│◄──── Private Lab Network        │
│             └──────────────┘       192.168.2.0/24            │
└─────────────────────────────────────────────────────────────┘
```

## Management Network (SLIRP)

When you start a VM, Yeast creates a management network using QEMU's SLIRP (user networking).

### What is SLIRP?

SLIRP is a userspace TCP/IP stack that runs inside the QEMU process:
- It translates guest packets to real host sockets
- No kernel networking modules required
- No root privileges needed

### How It Works

```
Guest VM ──► SLIRP (in QEMU process) ──► Host sockets ──► Internet
```

### Key Properties

| Property | Value |
|---|---|
| Requires root | No |
| Works on any Linux | Yes |
| Guest can reach internet | Yes (via NAT) |
| Host can reach guest | Only via forwarded ports |
| Other machines can reach guest | No |

### Port Forwarding

To access guest services from your host, declare port forwards in `yeast.yaml`:

```yaml
instances:
  - name: web
    ports:
      - host_port: 8080
        guest_port: 80
      - host_port: 8443
        guest_port: 443
```

**Access from host:**

```bash
# HTTP to guest's port 80
curl http://127.0.0.1:8080

# HTTPS to guest's port 443
curl https://127.0.0.1:8443
```

**Rules:**
- `host_port` must be unique across all instances in the project
- Both TCP and UDP are forwarded
- No additional host firewall configuration needed

### Default SSH Access

Every instance gets automatic SSH forwarding:

```yaml
instances:
  - name: web
    ssh_port: 2222  # Host port for SSH
```

Access from host:

```bash
ssh -p 2222 yeast@127.0.0.1
# Or use Yeast's shortcut:
yeast ssh web
```

## Private Lab Network

For VM-to-VM communication, Yeast creates a private lab network.

### How It Works

Yeast uses QEMU's socket netdev with multicast:
- All VMs on the same network join a multicast group
- Ethernet frames are sent to the multicast address
- All VMs on the network receive the frames

**How the multicast address is derived:**
```
Hash(project_id + network_name) → Unique multicast IP + port
```

This ensures:
- Different projects don't see each other's VMs
- Same network name in different projects = different multicast groups
- No configuration needed beyond `yeast.yaml`

### Key Properties

| Property | Value |
|---|---|
| Isolated from host | Yes |
| Isolated from internet | Yes |
| Requires root | No |
| DHCP | No (static IPs only) |
| Router | No (flat L2 segment) |
| Unique per project | Yes |

### Declaring a Lab Network

```yaml
networks:
  - name: lab
    cidr: 192.168.2.0/24

instances:
  - name: web
    networks:
      - name: lab
        ipv4: 192.168.2.10

  - name: db
    networks:
      - name: lab
        ipv4: 192.168.2.20
```

### Static IP Assignment

Assign static IPs in `yeast.yaml`:

```yaml
instances:
  - name: web
    networks:
      - name: lab
        ipv4: 192.168.2.10
```

**Important:**
- IPs must be within the network's CIDR range
- Each IP must be unique within the network
- Yeast does not provide DHCP — you must assign static IPs
- If you omit `ipv4`, the VM gets a link-local address

### VM-to-VM Communication

VMs on the same network can communicate directly:

```bash
# From web, ping db
yeast exec web -- ping -c 3 192.168.2.20

# From db, check web's HTTP server
yeast exec db -- curl -fsS http://192.168.2.10

# SSH from one VM to another (using SSH keys)
yeast exec web -- ssh 192.168.2.20
```

### Viewing Lab Network Info

```bash
# Check lab IP in status
yeast status

# Expected output:
# Project status
#   NAME  STATUS   SSH              LAB IP
#   web   running  127.0.0.1:2222   192.168.2.10
#   db    running  127.0.0.1:2223   192.168.2.20

# Check network interface inside VM
yeast ssh web
ip addr show yeastlab0
# Shows: inet 192.168.2.10/24
```

## Network Configuration Examples

### Single VM with Web Access

```yaml
version: 1

instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2222
    ports:
      - host_port: 8080
        guest_port: 80
```

Access: SSH on 2222, HTTP on 8080.

### Two-VM Lab with Private Network

```yaml
version: 1

networks:
  - name: lab
    cidr: 10.10.10.0/24

instances:
  - name: attacker
    image: ubuntu-24.04
    ssh_port: 2222
    networks:
      - name: lab
        ipv4: 10.10.10.10

  - name: target
    image: ubuntu-24.04
    ssh_port: 2223
    networks:
      - name: lab
        ipv4: 10.10.10.20
```

Access: SSH via host ports. VMs communicate on 10.10.10.x.

### Web + Database with Private Network

```yaml
version: 1

networks:
  - name: backend
    cidr: 192.168.100.0/24

instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2222
    ports:
      - host_port: 8080
        guest_port: 80
    networks:
      - name: backend
        ipv4: 192.168.100.10

  - name: db
    image: ubuntu-24.04
    ssh_port: 2223
    networks:
      - name: backend
        ipv4: 192.168.100.20
```

Web app connects to database at `192.168.100.20`.

## Troubleshooting

### VMs Can't Ping Each Other

1. **Check both VMs have the lab NIC:**
   ```bash
   yeast exec web -- ip addr show
   # Should show an interface with the lab IP
   ```

2. **Check IP addresses match `yeast.yaml`:**
   ```bash
   yeast exec web -- ip addr show yeastlab0
   yeast exec db -- ip addr show yeastlab0
   ```

3. **Check multicast address (should be same for both):**
   ```bash
   ps aux | grep qemu | grep mcast
   ```

4. **Restart both VMs:**
   ```bash
   yeast down && yeast up
   ```

### Can't SSH into VM

1. **Check if VM is running:**
   ```bash
   yeast status
   ```

2. **Check if SSH port is listening on host:**
   ```bash
   ss -tlnp | grep 2222
   ```

3. **Check VM logs:**
   ```bash
   yeast logs web
   ```

4. **Try manual SSH:**
   ```bash
   ssh -p 2222 -v yeast@127.0.0.1
   ```

### Port Conflict

If `yeast up` fails with "port already in use":

```bash
# Find what's using the port
ss -tlnp | grep 8080

# Either kill the process or change the port in yeast.yaml
```

### No Internet Access in VM

SLIRP provides NAT internet access by default. If VMs can't reach the internet:

1. **Check if the management network is working:**
   ```bash
   yeast exec web -- ping -c 3 8.8.8.8
   ```

2. **Check DNS:**
   ```bash
   yeast exec web -- nslookup google.com
   ```

3. **Restart the VM:**
   ```bash
   yeast down && yeast up
   ```

## Limitations

Current networking limitations in Yeast v1:

- **One network per project** — Can't have multiple isolated lab networks
- **No DHCP** — Static IPs only
- **No bridge mode** — Can't bridge to host's physical network
- **No IPv6** — IPv4 only
- **No custom routes** — Flat L2 segment
- **Port forwards are static** — Can't forward port ranges dynamically

## Next Steps

- [Configuration](./configuration) — Full yeast.yaml reference
- [Multi-VM Lab Tutorial](./tutorials/04-multi-vm-lab) — Build your first networked lab
- [Troubleshooting](./troubleshooting) — More network debugging tips
