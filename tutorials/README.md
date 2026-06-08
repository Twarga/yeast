# Yeast Tutorial Series

> **Goal**: Learn Yeast by doing. Start with a single VM and progress to multi-machine labs, provisioning, snapshots, guest control, and automation.

These tutorials are designed for the owner/developer who wants to see Yeast work in the real world before building on top of it. Each tutorial builds on the previous one.

---

## Prerequisites

Before starting, ensure your host is ready:

```bash
yeast doctor
```

Required:
- Linux host
- `/dev/kvm` accessible
- `qemu-system-x86_64`, `qemu-img`
- `genisoimage` or `mkisofs`
- `ssh` and a public key at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`

Install dependencies if needed:

```bash
# Ubuntu / Debian
sudo apt install qemu-system-x86 qemu-utils genisoimage

# Fedora / RHEL
sudo dnf install qemu-system-x86 qemu-img genisoimage

# Arch Linux
sudo pacman -S qemu-base cdrtools
```

Ensure your user is in the `kvm` group:

```bash
sudo usermod -aG kvm $USER
# Log out and back in for this to take effect
```

---

## Tutorial Map

| # | Tutorial | What You Will Learn | Time |
|---|---|---|---|
| 01 | [Your First VM](01-first-vm.md) | Create, start, SSH into, stop, and destroy a basic Ubuntu VM | 10 min |
| 02 | [Provisioning](02-provisioning.md) | Automate setup with packages, files, and shell commands | 15 min |
| 03 | [Snapshots and Reset](03-snapshots.md) | Save a clean baseline and restore it after changes | 10 min |
| 04 | [Multi-VM Lab](04-multi-vm-lab.md) | Create a private network of VMs with static IPs | 15 min |
| 05 | [Guest Control](05-guest-control.md) | Run commands, copy files, read logs, and inspect VMs | 10 min |
| 06 | [LabsBackery Lab](06-labsbackery-lab.md) | Build an attacker/target cybersecurity lab package | 20 min |
| 07 | [Templates](07-templates.md) | Use built-in templates and create your own | 10 min |
| 08 | [JSON and Events](08-json-automation.md) | Drive Yeast from scripts and tools with `--json` and `--events` | 10 min |
| 09 | [Nodi Home Lab](09-nodi-home-lab.md) | Build a complex 4-VM service architecture with shared storage, web services, and cross-VM networking | 60 min |
| 10 | [Load Balancer Lab](10-load-balancer-lab.md) | Build a 3-VM reverse proxy with round-robin load balancing, health checks, and failure resilience | 30 min |
| 11 | [Database + App Stack](11-database-app-stack.md) | Build a 2-VM two-tier application with PostgreSQL persistence, REST API, and snapshot data reset | 35 min |
| 12 | [Monitoring Stack](12-monitoring-stack.md) | Build a 4-VM observability stack with Prometheus scraping and Grafana dashboards | 40 min |
| 13 | [WireGuard VPN Mesh](13-wireguard-vpn-mesh.md) | Build an encrypted overlay network with WireGuard hub-and-spoke topology and NAT routing | 30 min |
| 14 | [GitOps / CI Lab](14-gitops-ci-lab.md) | Build a 3-VM GitOps pipeline with Gitea, Drone CI, and a Docker registry | 50 min |

---

## How to Use These Tutorials

1. Create a dedicated workspace folder for tutorials:

```bash
mkdir ~/yeast-tutorials
cd ~/yeast-tutorials
```

2. Each tutorial creates its own project folder. You can safely `yeast destroy` at the end of each one without affecting others.

3. If something goes wrong, run `yeast status --json` to inspect state, and `yeast logs <instance>` to see the VM runtime log.

4. The `--json` flag is mentioned throughout so you can see the machine-readable contract Yeast exposes to tools.

5. Expect the first `yeast up` in a project to be the slow one. That is the cold boot path: cloud-init, first SSH readiness, and any first-run provisioning.

6. Expect later `yeast up` runs on the same unchanged project to be much faster. That is the warm boot path: Yeast reuses the disk and skips provisioning when the fingerprint is unchanged.

7. Use `yeast up --reprovision` when you want the next boot to rerun provisioning on purpose. Use `yeast up --no-provision` when you want a fast boot without package/file/shell provisioning.

---

## Expected Outcomes

After completing this series, you will understand:

- How Yeast turns a `yeast.yaml` into a real QEMU/KVM VM
- How cloud-init prepares the guest on first boot
- How post-boot provisioning automates setup
- How snapshots create safe reset points for labs
- How private lab networking connects VMs
- How guest control commands let you operate inside VMs without SSH sessions
- How Yeast's JSON/events contract enables automation and future LabsBackery/MCP integration
- How to build a realistic multi-VM service lab with shared storage, web UIs, and cross-VM dependencies
- How reverse proxies distribute traffic and handle backend failures
- How database persistence works with snapshots and restoration
- How metrics scraping and dashboards provide observability
- How VPN tunnels create encrypted overlays on existing networks
- How Git, CI runners, and registries form a GitOps pipeline

---

## Cleanup Note

Each tutorial ends with a `yeast destroy` step. If you skip it, old projects remain under `~/.yeast/projects/` and continue to occupy disk space. To clean everything:

```bash
# List all tracked projects
ls ~/.yeast/projects/

# Remove a specific project runtime data
rm -rf ~/.yeast/projects/<project-id>

# Or from inside the project folder:
cd ~/yeast-tutorials/<tutorial-folder>
yeast destroy
```

If a tutorial leaves a stuck VM, stale port, broken `yeast.yaml`, or orphaned QEMU process behind, run:

```bash
yeast clean
```

If a port is still busy afterward, inspect the host listener directly:

```bash
ss -tlnp | grep <port>
```

Useful recovery/debug commands while working through tutorials:

```bash
yeast status
yeast inspect <instance>
yeast logs <instance>
```

Those commands show the current SSH address, forwarded ports, runtime paths, and log paths for the instance.

---

Next: [Tutorial 01 - Your First VM](01-first-vm.md)
