---
title: Configuration
description: Complete yeast.yaml reference
---

# Configuration Reference

This page documents all fields in `yeast.yaml`.

## Overview

A `yeast.yaml` file defines your Yeast project. It contains:

1. **Version** - Configuration schema version
2. **Networks** - Private lab networks
3. **Instances** - Virtual machine definitions

## Example

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
```

## Top-Level Fields

### version

**Required**: Yes  
**Type**: Number  
**Value**: `1`

The configuration schema version. Always `1` for Yeast v1.

## networks

**Required**: No  
**Type**: Array

Defines private lab networks for VM-to-VM communication.

### networks[].name

**Required**: Yes  
**Type**: String

The network name. Must be unique within the project.

### networks[].cidr

**Required**: Yes  
**Type**: String

The CIDR notation for the network address space.

## instances

**Required**: Yes  
**Type**: Array

Defines the virtual machines in your project.

### instances[].name

**Required**: Yes  
**Type**: String

The instance name. Must be unique within the project.

### instances[].hostname

**Required**: No  
**Type**: String

The guest OS hostname. Written by cloud-init.

### instances[].image

**Required**: Yes  
**Type**: String

The base image to use. Must be pulled first with `yeast pull`.

### instances[].memory

**Required**: No  
**Type**: Number  
**Default**: 512

RAM in megabytes (MiB).

### instances[].cpus

**Required**: No  
**Type**: Number  
**Default**: 1

Number of virtual CPUs.

### instances[].disk_size

**Required**: No  
**Type**: String  
**Default**: "2G"

Disk size for the overlay image.

### instances[].ssh_port

**Required**: No  
**Type**: Number  
**Default**: Auto-assigned

The host port for SSH access. Must be unique across all instances.

### instances[].user

**Required**: No  
**Type**: String  
**Default**: "yeast"

The username created by cloud-init.

### instances[].sudo

**Required**: No  
**Type**: String  
**Default**: "nopasswd"

Sudo policy for the user.

### instances[].ports

**Required**: No  
**Type**: Array

Host-to-guest port forwarding rules.

### instances[].networks

**Required**: No  
**Type**: Array

Network attachments for the instance.

### instances[].env

**Required**: No  
**Type**: Array

Environment variables written to `/etc/profile.d/yeast-env.sh`.

### instances[].provision

**Required**: No  
**Type**: Object

Post-boot provisioning configuration.

#### instances[].provision.packages

**Required**: No  
**Type**: Array

Packages to install with the system package manager.

#### instances[].provision.files

**Required**: No  
**Type**: Array

Files to copy from the host to the guest.

#### instances[].provision.shell

**Required**: No  
**Type**: Array

Shell commands to execute during provisioning.

## Field Summary

| Field | Required | Type | Default | Description |
|-------|----------|------|---------|-------------|
| `version` | Yes | Number | - | Config schema version |
| `networks` | No | Array | - | Private lab networks |
| `instances` | Yes | Array | - | VM definitions |
| `instances[].name` | Yes | String | - | Instance name |
| `instances[].image` | Yes | String | - | Base image |
| `instances[].memory` | No | Number | 512 | RAM in MiB |
| `instances[].cpus` | No | Number | 1 | Virtual CPUs |
| `instances[].disk_size` | No | String | "2G" | Disk size |
| `instances[].ssh_port` | No | Number | Auto | SSH host port |
| `instances[].user` | No | String | "yeast" | Username |
| `instances[].sudo` | No | String | "nopasswd" | Sudo policy |
| `instances[].ports` | No | Array | - | Port forwarding |
| `instances[].networks` | No | Array | - | Network attachments |
| `instances[].env` | No | Array | - | Environment variables |
| `instances[].provision` | No | Object | - | Provisioning config |

## Next Steps

- [Architecture](./architecture) - How Yeast works under the hood
- [Provisioning](./provisioning) - Detailed provisioning guide
- [Networking](./networking) - Network configuration
- [Troubleshooting](./troubleshooting) - Common issues and fixes
