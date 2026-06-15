# Yeast Labs

The Yeast labs are a short public tutorial path for learning Yeast itself.

They are not a DevOps course. They teach the Yeast workflows you will use in real projects: projects, images, lifecycle, cloud-init, provisioning, status, JSON output, snapshots, private networking, and templates.

## Learning Path

Follow the labs in order:

| # | Lab | What You Learn |
|---:|---|---|
| 1 | [First VM, First SSH](01-first-vm-first-ssh.md) | the basic Yeast loop |
| 2 | [Cloud-Init Basics](02-cloud-init-basics.md) | first boot, user, hostname, SSH keys |
| 3 | [Provisioning After Boot](03-provisioning-after-boot.md) | packages, files, shell commands |
| 4 | [Status, Logs, Inspect, JSON](04-status-logs-inspect-json.md) | observing VMs and automation output |
| 5 | [Snapshots And Restore](05-snapshots-and-restore.md) | stopped-VM reset points |
| 6 | [Multi-VM Private Networking](06-multi-vm-private-networking.md) | one private project network |
| 7 | [Templates And Reusable Labs](07-templates-and-reusable-labs.md) | built-in and local templates |

## Before You Start

You need Yeast installed on a Linux host with QEMU/KVM.

Run:

```bash
yeast doctor
```

Fix blockers before starting Lab 01.

## How To Use These Labs

Each lab is meant to be copy/paste friendly and safe to clean up.

Use a new folder for each lab. That keeps project state, VM disks, and snapshots separate.

Most labs end with:

```bash
yeast down
yeast destroy
```

`yeast down` stops VMs and keeps disks. `yeast destroy` removes tracked runtime files and disks for that lab project.

## What These Labs Do Not Teach

These labs do not teach Nginx, Docker, Kubernetes, CI/CD, or production DevOps.

Those topics belong in a separate course. Here, the job is simpler: become comfortable with Yeast as the local VM engine.
