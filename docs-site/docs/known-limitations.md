---
title: Known Limitations
description: Current limitations in Yeast v1
---

# Known Limitations

This page documents known limitations in Yeast v1. These are not bugs — they are intentional scope boundaries for the current release.

## Platform Limitations

### Linux Only

Yeast only runs on Linux. It does not support:
- macOS (no KVM, different QEMU toolchain)
- Windows native (no KVM, different virtualization stack)
- WSL1 (lacks Linux kernel)

**WSL2:** Partially supported. VMs work but KVM acceleration may not be available without nested virtualization. See [Installation](./installation) for WSL2 setup.

### x86_64 Only

Yeast only supports x86_64 (AMD64) architecture.

ARM64 support is planned for a future release.

### No Nested Virtualization

Yeast VMs cannot run other VMs inside them (no nested KVM).

This is a QEMU/KVM limitation, not a Yeast limitation.

## Networking Limitations

### Single Network Per Project

Yeast v1 supports only one private lab network per project.

If you need multiple isolated networks, create separate projects.

### No DHCP

Private networks use static IPs only. There is no DHCP server.

You must assign `ipv4` addresses in `yeast.yaml`:

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

### No Router

Private networks are flat L2 segments. There is no:
- Router VM
- Gateway configuration
- Inter-network routing

### No Bridge Mode

Yeast uses SLIRP (user networking) for management and socket multicast for lab networks. It does not support bridge mode or direct host network access.

**Implication:** VMs cannot be reached from other machines on your LAN without external port forwarding at the host level.

### No IPv6

Only IPv4 is supported. IPv6 addressing is not available.

## VM Limitations

### No Windows Guests

Yeast does not support Windows VMs.

The focus is on Linux cloud images with cloud-init. Windows would require:
- Different image formats
- VirtIO driver injection
- Cloudbase-Init instead of cloud-init

### No GPU Passthrough

GPU passthrough is not supported.

Use cases requiring GPU acceleration should use bare-metal or other tools.

### Resource Limits

VMs have practical limits:
- **RAM:** Tested up to 8 GB per VM
- **CPUs:** Tested up to 4 per VM
- **Disk:** Tested up to 100 GB per VM

Higher values may work but are not officially supported.

### Manual Images Require User Setup

`yeast pull --list` includes both auto-downloadable cloud images and manual/setup-only image entries. Manual images are searchable and documented, but Yeast cannot automatically download or prepare them because their upstream distribution format, licensing flow, credentials, or cloud-init support differs from standard cloud images.

When a manual image is selected, Yeast prints setup instructions. You must place the prepared disk image in the expected cache location before `yeast up` can use it.

## Provisioning Limitations

### No Idempotent Shell Detection

Shell commands in `provision` run on every provisioning trigger. Yeast does not automatically detect if a command is already applied.

**Workaround:** Write idempotent commands:

```yaml
shell:
  - test -f /etc/nginx/nginx.conf || sudo cp /tmp/nginx.conf /etc/nginx/
  - sudo systemctl is-active nginx || sudo systemctl start nginx
```

### No Remote File Provisioning

Files must be local to the host. You cannot provision from URLs or remote sources.

**Workaround:** Use shell commands to download files:

```yaml
shell:
  - curl -fsSL https://example.com/file.txt -o /tmp/file.txt
```

### No Template Variables

There is no templating system in `yeast.yaml`. You cannot use variables like `${PROJECT_NAME}` or `${IP_RANGE}`.

**Workaround:** Use external templating tools (e.g., `envsubst`, `jinja2`) to generate `yeast.yaml`.

## Snapshot Limitations

### VM Must Be Stopped

You can only take or restore snapshots when the VM is stopped.

**Workflow:**
```bash
yeast down
yeast snapshot web baseline
yeast up
```

### Per-Instance Only

Snapshots are per-instance. There is no project-wide atomic snapshot.

**Workaround:** Snapshot each instance individually:

```bash
yeast snapshot web baseline
yeast snapshot db baseline
```

### No Incremental Snapshots

Each snapshot is a full disk copy, not an incremental diff.

**Implication:** Multiple snapshots consume more disk space than incremental snapshots would.

### Disk Only

Snapshots capture disk state only. They do not capture:
- Running process state
- Network connections
- Memory contents

**Workaround:** Stop the VM before snapshotting to ensure clean disk state.

## Lifecycle Limitations

### No Live Migration

VMs cannot be migrated between hosts while running.

**Workaround:** Use snapshots + `yeast pull` + `yeast up` on the new host.

### No Auto-Start

VMs do not start automatically on host boot.

**Workaround:** Use systemd user services or cron to run `yeast up` on boot.

### No Resource Monitoring

Yeast does not monitor or report host resource usage (CPU, RAM, disk) for VMs.

**Workaround:** Use standard Linux tools (`top`, `htop`, `df`) on the host.

### Full Smoke Tests Need a Real VM Runner

Unit tests and non-VM smoke tests can run quickly on ordinary development hosts. Full lifecycle smoke tests boot real QEMU/KVM VMs, wait for cloud-init and SSH readiness, and may download large cloud images.

For reliable full-smoke validation, use a host or CI runner with:
- writable `/dev/kvm`
- `qemu-system-x86_64` and `qemu-img`
- enough free disk space for image cache and qcow2 overlays
- network access to image mirrors
- enough time for first-boot cloud-init completion

## Security Limitations

### No VM Isolation Policies

Yeast does not implement VM-level security policies (AppArmor, SELinux, seccomp profiles).

**Workaround:** Apply security policies at the host level.

### No Encrypted Disks

VM disk images are not encrypted.

**Workaround:** Use host-level disk encryption (LUKS, BitLocker on host drive).

### No Network Policies

There are no built-in firewall rules or network policies for VMs.

**Workaround:** Use host-level firewall (`iptables`, `nftables`) or configure `ufw` inside VMs.

## What's Not In Scope (Yet)

These features may be addressed in future versions:

| Feature | Status |
|---|---|
| Multiple private networks | Not in v1 |
| DHCP for lab networks | Not in v1 |
| Windows guest support | Not in v1 |
| GPU passthrough | Not in v1 |
| Live snapshots | Not in v1 |
| Project-wide snapshots | Not in v1 |
| Auto-start on boot | Not in v1 |
| Resource monitoring | Not in v1 |
| Remote template registry | Not in v1 |
| Web UI / Dashboard | Not in v1 |
| REST API | Not in v1 |
| Plugin system | Not in v1 |

## Reporting Limitations

If a limitation is causing problems for your use case:

1. Check if there's a workaround in the [Troubleshooting](./troubleshooting) guide
2. Search [GitHub Issues](https://github.com/Twarga/yeast/issues) for related discussions
3. Open a feature request with your use case

## Next Steps

- [Troubleshooting](./troubleshooting) — Common issues and fixes
- [Architecture](./architecture) — How Yeast works under the hood
- [GitHub Issues](https://github.com/Twarga/yeast/issues) — Report bugs and request features
