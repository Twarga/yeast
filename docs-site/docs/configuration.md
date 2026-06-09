---
title: Configuration
description: Complete yeast.yaml reference
---

# Configuration Reference

This page documents all fields in `yeast.yaml`.

## Overview

A `yeast.yaml` file defines your Yeast project. It is a single source of truth for:

1. **Version** — Configuration schema version
2. **Networks** — Private lab networks for VM-to-VM communication
3. **Instances** — Virtual machine definitions

Yeast reads this file on every command and reconciles the desired state with the actual state.

## Minimal Example

The simplest possible `yeast.yaml`:

```yaml
version: 1

instances:
  - name: web
    image: ubuntu-24.04
```

This creates one VM with all defaults:
- 512 MB RAM
- 1 CPU
- 2 GB disk
- Auto-assigned SSH port
- Username `yeast`
- Passwordless sudo

## Complete Example

A fully-featured `yeast.yaml` for a multi-VM lab:

```yaml
version: 1

networks:
  - name: lab
    cidr: 192.168.2.0/24

instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 2
    disk_size: 20G
    ssh_port: 2222
    user: yeast
    sudo: nopasswd
    ports:
      - host_port: 8080
        guest_port: 80
    networks:
      - name: lab
        ipv4: 192.168.2.10
    env:
      - MY_VAR=value
      - APP_ENV=development
    provision:
      packages:
        - nginx
        - curl
      files:
        - source: ./files/index.html
          destination: /var/www/html/index.html
          permissions: "0644"
      shell:
        - sudo systemctl enable nginx
        - sudo systemctl start nginx

  - name: db
    hostname: db-lab
    image: ubuntu-24.04
    memory: 2048
    cpus: 2
    ssh_port: 2223
    networks:
      - name: lab
        ipv4: 192.168.2.20
    provision:
      packages:
        - postgresql
```

## Top-Level Fields

### version

**Required:** Yes  
**Type:** Number  
**Value:** `1`

The configuration schema version. Always `1` for Yeast v1.

```yaml
version: 1
```

### networks

**Required:** No  
**Type:** Array  
**Default:** `[]`

Defines private lab networks for VM-to-VM communication.

```yaml
networks:
  - name: lab
    cidr: 192.168.2.0/24
```

#### networks[].name

**Required:** Yes  
**Type:** String

The network name. Must be unique within the project. Used to attach instances.

#### networks[].cidr

**Required:** Yes  
**Type:** String

The CIDR notation for the network address space (e.g., `192.168.2.0/24`, `10.10.10.0/24`).

**Important:** Choose a range that doesn't conflict with your host's network.

## instances

**Required:** Yes  
**Type:** Array

Defines the virtual machines in your project.

### instances[].name

**Required:** Yes  
**Type:** String

The instance name. Must be unique within the project. Used for:
- SSH access (`yeast ssh web`)
- State tracking
- Snapshot names
- Log files

**Naming rules:**
- Must start with a letter
- Can contain letters, numbers, hyphens, and underscores
- Must be 1-63 characters

### instances[].hostname

**Required:** No  
**Type:** String  
**Default:** Same as `name`

The guest OS hostname. Written by cloud-init during first boot.

```yaml
hostname: web-lab
```

### instances[].image

**Required:** Yes  
**Type:** String

The base image to use. Must be pulled first with `yeast pull <image>`.

**Available images:**
- `ubuntu-24.04`
- `ubuntu-22.04`

```yaml
image: ubuntu-24.04
```

### instances[].memory

**Required:** No  
**Type:** Number  
**Default:** `512`

RAM in megabytes (MiB).

```yaml
memory: 1024  # 1 GB
```

**Recommended values:**
- Minimal VM: 512 MB
- Web server: 1024 MB
- Database: 2048+ MB

### instances[].cpus

**Required:** No  
**Type:** Number  
**Default:** `1`

Number of virtual CPUs.

```yaml
cpus: 2
```

### instances[].disk_size

**Required:** No  
**Type:** String  
**Default:** `"2G"`

Disk size for the overlay image. Accepts:
- Whole numbers: `20`, `50`
- With suffix: `512M`, `20G`, `1T`

```yaml
disk_size: 20G
```

**Important:** This only applies on first disk creation. Existing disks are kept as-is even if you change this value.

### instances[].ssh_port

**Required:** No  
**Type:** Number  
**Default:** Auto-assigned starting at 2222

The host port for SSH access. Must be unique across all instances in the project.

```yaml
ssh_port: 2222
```

**Tip:** Explicitly set `ssh_port` for projects you access frequently. This makes SSH URLs predictable.

### instances[].user

**Required:** No  
**Type:** String  
**Default:** `"yeast"`

The username created by cloud-init. This user gets SSH key access and sudo privileges.

```yaml
user: admin
```

### instances[].sudo

**Required:** No  
**Type:** String  
**Default:** `"nopasswd"`

Sudo policy for the user.

**Options:**
- `"none"` — No sudo access
- `"pass"` — Sudo with password (you must set a password via cloud-init)
- `"nopasswd"` — Sudo without password

```yaml
sudo: nopasswd
```

### instances[].ports

**Required:** No  
**Type:** Array  
**Default:** `[]`

Host-to-guest port forwarding rules. These let you access guest services from your host.

```yaml
ports:
  - host_port: 8080
    guest_port: 80
    description: "HTTP web server"
  - host_port: 8443
    guest_port: 443
    description: "HTTPS web server"
```

**Important:** `host_port` must be unique across all instances in the project.

### instances[].networks

**Required:** No  
**Type:** Array  
**Default:** `[]`

Network attachments for the instance. Each entry attaches the VM to a private lab network.

```yaml
networks:
  - name: lab
    ipv4: 192.168.2.10
```

#### networks[].name

The network name (must match a network defined in the top-level `networks` array).

#### networks[].ipv4

Static IPv4 address for the VM on this network. Must be within the network's CIDR range.

**Important:** Yeast does not provide DHCP. You must assign static IPs.

### instances[].env

**Required:** No  
**Type:** Array of strings  
**Default:** `[]`

Environment variables written to `/etc/profile.d/yeast-env.sh` in the guest. Available to all users after login.

```yaml
env:
  - APP_ENV=production
  - DATABASE_URL=postgres://localhost:5432/mydb
```

### instances[].provision

**Required:** No  
**Type:** Object

Post-boot provisioning configuration. See [Provisioning](./provisioning) for details.

```yaml
provision:
  packages:
    - nginx
    - curl
    - git
  files:
    - source: ./files/index.html
      destination: /var/www/html/index.html
      permissions: "0644"
  shell:
    - sudo systemctl enable nginx
    - sudo systemctl start nginx
```

#### provision.packages

Packages to install with the system package manager (`apt` on Ubuntu/Debian).

#### provision.files

Files to copy from host to guest.

| Field | Required | Description |
|---|---|---|
| `source` | Yes | Host path (relative to project root) |
| `destination` | Yes | Guest path (absolute) |
| `permissions` | No | Octal permissions (e.g., `"0644"`, `"0755"`) |

#### provision.shell

Shell commands to execute during provisioning. Commands run as the configured user.

**Important:** Use `sudo` for commands requiring root.

## Validation

Yeast validates `yeast.yaml` on every command:

- **Required fields** — `version`, `instances[].name`, `instances[].image`
- **Unique names** — Instance names and SSH ports must be unique
- **Valid CIDR** — Network CIDRs must be valid
- **Valid IPs** — Static IPs must be within their network's CIDR range
- **Port conflicts** — `host_port` values must not collide between instances
- **File existence** — `provision.files[].source` paths must exist

Validation errors show exactly which field is wrong and why.

## Environment Variables

Yeast supports these environment variables:

| Variable | Description |
|---|---|
| `YEAST_DEBUG` | Enable debug logging |
| `YEAST_JSON` | Default to JSON output |

## Next Steps

- [Architecture](./architecture) — How Yeast works under the hood
- [Provisioning](./provisioning) — Detailed provisioning guide
- [Networking](./networking) — Network configuration
- [Troubleshooting](./troubleshooting) — Common issues and fixes
