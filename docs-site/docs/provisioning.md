---
title: Provisioning
description: Install packages, copy files, and configure VMs
---

# Provisioning

This page explains how to provision VMs in Yeast.

## Overview

Provisioning is the process of configuring a VM after it boots. Yeast supports two phases:

1. **Cloud-init** (first boot only) - Creates users, sets hostname, configures networking
2. **Post-boot provisioning** (every boot) - Installs packages, copies files, runs commands

## Cloud-Init

Cloud-init runs automatically on first boot. It handles:

- Creating the `yeast` user
- Adding SSH keys
- Setting the hostname
- Configuring the lab network
- Setting sudo policy

## Post-Boot Provisioning

After cloud-init finishes and SSH is ready, Yeast runs the `provision` section.

### Packages

Install packages with the system package manager:

```yaml
instances:
  - name: web
    provision:
      packages:
        - nginx
        - curl
        - git
```

### Files

Copy files from host to guest:

```yaml
instances:
  - name: web
    provision:
      files:
        - source: ./files/index.html
          destination: /var/www/html/index.html
          permissions: "0644"
```

### Shell Commands

Run shell commands during provisioning:

```yaml
instances:
  - name: web
    provision:
      shell:
        - sudo systemctl enable nginx
        - sudo systemctl start nginx
```

## Complete Example

```yaml
instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 2
    ssh_port: 2222
    user: yeast
    sudo: nopasswd
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

## Provisioning Behavior

### First Boot

On first boot, Yeast runs:
1. Cloud-init (user, SSH, hostname, network)
2. Post-boot provisioning (packages, files, shell)

### Subsequent Boots

On subsequent boots, Yeast checks:
1. Has `yeast.yaml` changed?
2. If changed, re-run post-boot provisioning
3. If not changed, skip provisioning

### Force Reprovisioning

Force re-run of provisioning:

```bash
yeast up --reprovision
```

## Debugging Provisioning

### Check Provisioning Logs

```bash
yeast logs web
```

### SSH and Inspect

```bash
yeast ssh web
dpkg -l | grep nginx
ls -la /var/www/html/
systemctl status nginx
```

## Best Practices

1. **Keep provisioning idempotent** - Commands should work if run multiple times
2. **Use retries for network operations** - Package installs may fail temporarily
3. **Test provisioning** - Run `yeast up --reprovision` to verify

## Next Steps

- [Configuration](./configuration) - yeast.yaml reference
- [Architecture](./architecture) - How provisioning works under the hood
- [Troubleshooting](./troubleshooting) - Common issues and fixes
