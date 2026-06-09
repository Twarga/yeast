---
title: Architecture
description: How Yeast works under the hood
---

# Architecture Overview

This page explains how Yeast works at a high level — what happens when you run commands, where files are stored, and how the different components interact.

## Mental Model

Yeast is a **desired state** system for local VMs. You declare what you want in `yeast.yaml`, and Yeast makes it happen.

```
Desired State          Yeast Engine           Actual State
yeast.yaml      ────►  Load + Validate   ────►  QEMU/KVM VMs
```

The core loop:
1. Read `yeast.yaml` to learn what should exist
2. Read `.yeast/state.json` to learn what already exists
3. Reconcile: create, modify, or remove resources to match desired state

## Components

### 1. Project Identity

Every Yeast project has a unique ID stored in `.yeast/project.json`. This ID:
- Creates isolated namespaces for instances
- Derives unique multicast addresses for private networks
- Prevents collisions between projects

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "version": "1.0"
}
```

### 2. Configuration Layer

The config layer reads and validates `yeast.yaml`:
- Schema validation (required fields, types)
- Cross-reference validation (port uniqueness, IP in CIDR range)
- File existence checks (provision sources)

If validation fails, Yeast stops with a clear error message pointing to the exact field.

### 3. Image Cache

Base images are cached at `~/.yeast/cache/images/<image-name>/`:
- Downloaded once, shared across all projects
- QCOW2 format for copy-on-write overlays
- Verified on pull, cached until explicitly removed

```
~/.yeast/cache/images/
  ubuntu-24.04/
    disk.qcow2          # Base image
    manifest.json       # Metadata (version, checksum, date)
```

### 4. Instance Disks

Each VM gets a **copy-on-write (QCOW2)** overlay disk:
- Base image is never modified
- Changes go to the overlay
- Multiple VMs from the same image share the base (saves disk space)
- Disks live in `~/.yeast/projects/<id>/instances/<name>/disk.qcow2`

### 5. Cloud-Init

Yeast uses cloud-init for first-boot provisioning:
- Generates `user-data`, `meta-data`, and `network-config` files
- Packages them into a `seed.iso` (ISO 9660 filesystem)
- Attaches the ISO to QEMU as a CD-ROM
- cloud-init reads the ISO on first boot and configures the VM

**What cloud-init does:**
- Creates the configured user
- Adds your SSH public key
- Sets the hostname
- Configures static IP addresses on lab networks
- Sets up sudo policy

### 6. QEMU Runtime

Yeast starts QEMU with KVM acceleration:

**Default devices:**
- VirtIO block device (for the disk)
- VirtIO network device (for management network)
- Additional VirtIO network device per lab network
- Serial console (for logging)
- Cloud-init ISO (first boot only)

**Management network:**
- Uses QEMU's user networking (SLIRP)
- No root privileges required
- Port forwarding for SSH and declared ports
- VMs can reach the internet but are isolated from host network

**Lab networks:**
- Uses QEMU's socket netdev with multicast
- All VMs on the same network see each other's Layer 2 frames
- Completely isolated from host
- No DHCP — static IPs only

### 7. State Management

Yeast tracks runtime state in `state.json`:

```json
{
  "project_id": "a1b2c3d4...",
  "instances": {
    "web": {
      "status": "running",
      "pid": 12345,
      "ssh_port": 2222,
      "disk_path": ".../disk.qcow2",
      "seed_path": ".../seed.iso"
    }
  }
}
```

This enables:
- Finding running VMs without scanning processes
- Clean shutdown (send ACPI signal to PID)
- Reconciliation (detect orphaned processes)
- Concurrent access (file locking)

## Lifecycle Flows

### Startup Flow (`yeast up`)

```
1. Load .yeast/project.json
   └── Get project ID

2. Load yeast.yaml
   └── Validate configuration

3. For each instance in yeast.yaml:
   a. Check if image exists in cache
      └── If not: error (run yeast pull first)
   
   b. Check/create disk.qcow2
      └── If missing: create overlay from base image
      └── If exists: reuse
   
   c. Generate cloud-init files
      └── user-data (users, SSH keys, packages)
      └── meta-data (instance-id, hostname)
      └── network-config (static IPs, routes)
   
   d. Build seed.iso from cloud-init files
   
   e. Build QEMU command line
      └── CPU, memory, disk, network, cloud-init ISO
   
   f. Start QEMU process
      └── Record PID in state.json
   
   g. Wait for SSH to become available
      └── Poll 127.0.0.1:<ssh_port> with timeout
   
   h. If provisioning configured:
      └── Install packages
      └── Copy files
      └── Run shell commands

4. Report results
```

### Shutdown Flow (`yeast down`)

```
1. Load state.json
2. For each running instance:
   a. Send ACPI shutdown signal to QEMU process
   b. Wait for process to exit (with timeout)
   c. If timeout: force kill (SIGKILL)
   d. Update state.json (mark as stopped)
3. Report results
```

### Destroy Flow (`yeast destroy`)

```
1. Stop all running instances (same as yeast down)
2. For each instance:
   a. Remove disk.qcow2
   b. Remove seed.iso
   c. Remove cloud-init files
   d. Remove log files
   e. Remove state.json
3. Remove snapshots directory
4. Report results
```

## Storage Layout

```
~/.yeast/
├── cache/
│   └── images/
│       ├── ubuntu-24.04/
│       │   ├── disk.qcow2          # Downloaded base image
│       │   └── manifest.json       # Image metadata
│       └── ubuntu-22.04/
│           ├── disk.qcow2
│           └── manifest.json
└── projects/
    └── <project-id>/
        ├── .yeast/
        │   └── project.json        # Project identity
        ├── yeast.yaml              # User configuration
        ├── state.json              # Runtime state
        └── instances/
            └── <name>/
                ├── disk.qcow2      # VM overlay disk
                ├── seed.iso         # Cloud-init ISO
                ├── user-data        # Cloud-init config
                ├── meta-data        # Cloud-init metadata
                ├── network-config   # Cloud-init network
                ├── vm.log           # QEMU console log
                └── snapshots/
                    └── <name>.qcow2 # Snapshot disks
```

## Key Design Decisions

### Project-Scoped State

Every project is completely isolated:
- Two projects can both have a VM named `web`
- Snapshots are per-project
- State files don't interfere

This means you can safely run multiple labs side by side.

### Copy-on-Write Disks

VMs use QCOW2 overlays instead of copying the full base image:
- 20 GB base image + 500 MB changes = 500 MB disk usage for the VM
- Multiple VMs from the same image share the base
- Snapshots are just copies of the overlay (fast)

### No Background Daemon

Yeast does not run a persistent service:
- Each command starts, does its work, and exits
- State is stored in files, not a database
- Works without systemd, Docker, or any background process

This makes Yeast predictable and debuggable.

### SLIRP Networking

Yeast uses QEMU's user-mode networking (SLIRP) for management:
- No root required
- No bridge configuration
- Works on any Linux system
- VMs get NAT internet access automatically

The tradeoff: VMs can't be reached from other machines on your network (unless you forward ports).

## Security Model

- VMs run as your user (not root)
- Disk images are owned by your user
- SSH access is key-based only (no passwords)
- Lab networks are completely isolated (no host access)
- Port forwarding is explicit (must be declared in yeast.yaml)

## Performance Characteristics

| Operation | Typical Time |
|---|---|
| `yeast pull ubuntu-24.04` | 2-5 minutes (first time) |
| `yeast up` (first boot) | 30-60 seconds |
| `yeast up` (subsequent) | 5-15 seconds |
| `yeast down` | 5-10 seconds |
| `yeast snapshot` | 1-5 seconds |
| `yeast restore` | 1-5 seconds |

## Next Steps

- [Configuration](./configuration) — yeast.yaml reference
- [Commands](./commands) — CLI command reference
- [Troubleshooting](./troubleshooting) — Common issues and fixes
- [Networking](./networking) — How networking works
