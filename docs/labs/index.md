# Yeast Labs

The Yeast labs are public documentation tutorials.

They teach Yeast itself, not DevOps.

Follow them in order:

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

Install Yeast and run:

```bash
yeast doctor
```

Fix blockers before starting the labs.

## Style

Each lab is practical and short enough to finish. You should be able to copy the commands, understand what Yeast is doing, verify the result, and clean up safely.
