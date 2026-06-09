---
title: Commands
description: Complete CLI command reference for Yeast
---

# Command Reference

This page documents all Yeast CLI commands with examples and common flags.

## Command Syntax

```bash
yeast <command> [arguments] [flags]
```

## Core Commands

### yeast init

Initialize a new Yeast project in the current directory.

```bash
yeast init
```

**Creates:**
- `.yeast/project.json` — Project identity
- `yeast.yaml` — Default configuration

**Options:**

```bash
# List built-in templates
yeast init --list-templates

# Initialize from a built-in template
yeast init --template ubuntu-basic
yeast init --template caddy-single-vm
yeast init --template two-vm-lab

# Initialize from a local template directory
yeast init --template /path/to/template
```

**Output:**
```
Project initialized
  ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890
  Config: yeast.yaml
```

---

### yeast pull

Download a base image to the local cache.

```bash
yeast pull ubuntu-24.04
```

**Available images:**
- `ubuntu-24.04`
- `ubuntu-22.04`

**Options:**

```bash
# List available images
yeast pull --list
```

**Output:**
```
Pulling ubuntu-24.04...
Image downloaded successfully.
  Size: 2.3 GB
  Cache: ~/.yeast/cache/images/ubuntu-24.04/
```

**Note:** Images are cached and shared across all projects. You only need to pull once.

---

### yeast up

Start all VMs in the project.

```bash
yeast up
```

**What it does:**
1. Validates `yeast.yaml`
2. Creates disks if missing
3. Generates cloud-init
4. Starts QEMU processes
5. Waits for SSH
6. Runs provisioning

**Options:**

```bash
# Force re-run provisioning
yeast up --reprovision

# JSON output
yeast up --json

# JSON with event stream
yeast up --json --events
```

**Output:**
```
OK Instances ready
  RUN  web  127.0.0.1:2222
  RUN  db   127.0.0.1:2223
```

---

### yeast down

Stop all VMs in the project.

```bash
yeast down
```

**What it does:**
1. Sends ACPI shutdown to each VM
2. Waits for graceful shutdown (with timeout)
3. Force-kills if timeout exceeded
4. Updates state

**Options:**

```bash
# JSON output
yeast down --json

# JSON with event stream
yeast down --json --events
```

**Output:**
```
OK Instances stopped
  web  stopped
  db   stopped
```

---

### yeast destroy

Remove all project resources.

```bash
yeast destroy
```

**What it removes:**
- VMs and their disks
- Cloud-init ISOs
- State files
- Snapshots

**Warning:** This permanently deletes data. Cannot be undone.

**Options:**

```bash
# JSON output
yeast destroy --json

# JSON with event stream
yeast destroy --json --events
```

---

## Access Commands

### yeast ssh

SSH into a running VM.

```bash
yeast ssh web
```

**Opens an interactive SSH session.**

**Options:**

```bash
# Run a command without interactive session
yeast ssh web -- ls -la /

# Note: Use -- to separate Yeast flags from SSH command
```

**Note:** `yeast ssh` does not support `--json` (it's interactive).

---

### yeast exec

Execute a command in a VM without opening an interactive session.

```bash
yeast exec web -- whoami
```

```bash
yeast exec web -- sudo apt update
```

```bash
yeast exec db -- psql -c "SELECT version();"
```

**Options:**

```bash
# JSON output
yeast exec web --json -- whoami
```

---

### yeast copy

Copy files between host and VM.

**Copy to guest:**

```bash
yeast copy web --to-guest ./local-file.txt /home/yeast/remote-file.txt
```

**Copy from guest:**

```bash
yeast copy web --from-guest /home/yeast/remote-file.txt ./local-file.txt
```

**Options:**

```bash
# JSON output
yeast copy web --json --to-guest ./file.txt /home/yeast/file.txt
```

---

## Status Commands

### yeast status

Show the status of all VMs in the project.

```bash
yeast status
```

**Output:**
```
Project status
  NAME  STATUS   SSH              LAB IP
  web   running  127.0.0.1:2222   192.168.2.10
  db    running  127.0.0.1:2223   192.168.2.20
```

**Options:**

```bash
# JSON output
yeast status --json
```

---

### yeast inspect

Show detailed information about a specific VM.

```bash
yeast inspect web
```

**Output:**
```
Instance: web
  Status: running
  PID: 12345
  SSH: 127.0.0.1:2222
  Image: ubuntu-24.04
  Memory: 512 MB
  CPUs: 1
  Disk: ~/.yeast/projects/.../instances/web/disk.qcow2
  Log: ~/.yeast/projects/.../instances/web/vm.log
```

**Options:**

```bash
# JSON output
yeast inspect web --json
```

---

### yeast logs

Show logs for a VM.

```bash
yeast logs web
```

**Shows:** QEMU console output, cloud-init logs, provisioning output.

**Options:**

```bash
# Show last N lines
yeast logs web --tail 50

# Follow logs (like tail -f)
yeast logs web --follow
```

---

## Snapshot Commands

### yeast snapshot

Create a snapshot of a VM.

```bash
yeast snapshot web baseline --description "Clean install"
```

**Requirements:** VM must be stopped (`yeast down`).

---

### yeast snapshots

List snapshots for an instance.

```bash
yeast snapshots web
```

**Output:**
```
Snapshots for web
  NAME      DESCRIPTION           CREATED
  baseline  Clean install         2024-06-08 14:32
```

---

### yeast restore

Restore an instance from a snapshot.

```bash
yeast restore web baseline
```

**Requirements:** VM must be stopped.

**Warning:** Overwrites current disk state.

---

### yeast delete-snapshot

Delete a snapshot.

```bash
yeast delete-snapshot web baseline
```

---

## Utility Commands

### yeast doctor

Check system requirements and diagnose issues.

```bash
yeast doctor
```

**Checks:**
- Operating system
- KVM support
- QEMU installation
- Required packages
- SSH key availability
- System resources

**Output:**
```
System Check
  OS:        linux
  KVM:       available
  QEMU:      found at /usr/bin/qemu-system-x86_64
  SSH key:   found at ~/.ssh/id_ed25519.pub
  Resources: sufficient
```

---

### yeast clean

Clean up stale state and orphaned resources.

```bash
yeast clean
```

**What it does:**
- Removes stale state files
- Kills orphaned QEMU processes
- Removes broken instance directories

**When to use:**
- After a crash
- When `yeast status` shows wrong information
- Before starting fresh after manual QEMU intervention

---

### yeast version

Show the Yeast version.

```bash
yeast version
```

**Output:**
```
yeast version 1.0.1
```

---

## Global Flags

| Flag | Description | Example |
|---|---|---|
| `--json` | Output JSON instead of human-readable | `yeast status --json` |
| `--events` | Include event stream in JSON output | `yeast up --json --events` |
| `--help`, `-h` | Show help for a command | `yeast up --help` |
| `--version`, `-v` | Show version | `yeast --version` |

## JSON Output

All commands (except `ssh`) support `--json` for machine-readable output:

```bash
yeast status --json
```

**Envelope format:**

```json
{
  "schema_version": "yeast.v1",
  "status": "success",
  "data": {
    "instances": [
      {
        "name": "web",
        "status": "running",
        "ssh_port": 2222
      }
    ]
  }
}
```

**Error format:**

```json
{
  "schema_version": "yeast.v1",
  "status": "error",
  "error": {
    "code": "PORT_CONFLICT",
    "message": "Port 2222 is already in use",
    "instance": "web",
    "field": "ssh_port"
  }
}
```

## Environment Variables

| Variable | Description |
|---|---|
| `YEAST_DEBUG` | Enable debug logging |
| `YEAST_JSON` | Default to JSON output for all commands |

## Next Steps

- [Configuration](./configuration) — yeast.yaml reference
- [Troubleshooting](./troubleshooting) — Common issues and fixes
- [Tutorials](/tutorials/) — Step-by-step guided labs
