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
- `debian-12`, `debian-13`
- `ubuntu-24.04`, `ubuntu-22.04`
- `fedora-41`, `fedora-42`
- `alma-9`, `amazon-linux-2023`, `centos-stream-9`, `rocky-9`
- Manual/setup-only entries such as `kali-2026.1`, `parrot-security-7.1`, `alpine-3.21`, `arch-linux`, `nixos-24.11`

**Options:**

```bash
# List available images
yeast pull --list

# Show locally cached images
yeast pull --cached
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

# Skip provisioning for this boot
yeast up --no-provision

# Boot VMs one at a time for easier debugging
yeast up --sequential

# Print a boot-time profile
yeast up --profile

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

# Fail if the command takes too long
yeast exec web --timeout 30s -- systemctl status ssh
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

# Fail if the transfer takes too long
yeast copy web --timeout 30s --from-guest /var/log/syslog ./syslog
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
```

---

## Provisioning Commands

### yeast provision

Rerun provisioning for a reachable running instance.

```bash
# Re-provision the only running instance
yeast provision

# Re-provision a specific instance
yeast provision web
```

**Use when:** you changed `provision` steps in `yeast.yaml` and want to apply them without recreating the VM.

**Options:**

```bash
# JSON output
yeast provision web --json

# JSON with event stream
yeast provision web --json --events
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

### yeast images clean

```bash
# Remove one cached image
yeast images clean ubuntu-24.04

# Preview image cache cleanup
yeast images clean ubuntu-24.04 --dry-run

# Remove all cached images
yeast images clean --all
```

**What it removes:**
- Cached base images under `~/.yeast/cache/images`
- Only image cache data, not project VM disks

**When to use:**
- To free disk space after testing multiple images
- Before re-downloading a corrupted cached image

---

### yeast update

Update the installed Yeast binary from GitHub releases.

```bash
# Check whether an update is available
yeast update --check

# Install the latest release
yeast update

# Install a specific version
yeast update --version v1.1.0

# Reinstall even if the version matches
yeast update --force --version v1.1.0
```

**What it does:**
- Downloads the release tarball for your platform
- Verifies it against `SHA256SUMS.txt`
- Extracts the `yeast` binary
- Replaces the currently running binary atomically

**When to use:**
- After a new release is published
- During release smoke testing
- To recover from an older installed binary

---

### yeast version

Show the Yeast version.

```bash
yeast version
```

**Output:**
```
v1.1.0
```

---

### yeast docs

Render bundled docs in the terminal.

```bash
# List embedded docs topics
yeast docs --list

# Open a topic
yeast docs quickstart
yeast docs release-smoke
```

**Note:** `yeast docs` does not support `--json`.

---

### yeast completion

Generate shell completion scripts.

```bash
yeast completion bash
yeast completion zsh
yeast completion fish
yeast completion powershell
```

---

## Global Flags

| Flag | Description | Example |
|---|---|---|
| `--json` | Output JSON instead of human-readable | `yeast status --json` |
| `--events` | Include event stream in JSON output | `yeast up --json --events` |
| `--quiet`, `-q` | Suppress progress output | `yeast up --quiet` |
| `--help`, `-h` | Show help for a command | `yeast up --help` |

## JSON Output

Most workflow commands support `--json` for machine-readable output. Interactive or terminal-rendering commands such as `ssh`, `docs`, and `completion` do not.

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
- [Tutorials](./tutorials/) — Step-by-step guided labs
