# Yeast

Yeast turns a folder into real Linux VMs.

Define machines in `yeast.yaml`, run `yeast up`, and get SSH-ready QEMU/KVM guests with cloud-init, provisioning, snapshots, private networking, and JSON output for automation.

## Twarga Academy

The DevOps bootcamp lives in its own course area so the Yeast product docs stay clean.

- [Academy Home](academy/index.md) - start the course
- [Curriculum](academy/curriculum/index.md) - see the full learning path
- [Labs](academy/labs/index.md) - open the mirrored lab set
- [Source Contract](academy/support/index.md) - read the course rules and source docs

## Learn Yeast In Order

If you are new, follow this path. It is ordered so each page teaches one layer before the next one.

| Step | Read | You learn |
|---:|---|---|
| 1 | [What Is Yeast?](getting-started/what-is-yeast.md) | What problem Yeast solves |
| 2 | [Installation](getting-started/installation.md) | How to get the CLI and check your host |
| 3 | [Quickstart](getting-started/quickstart.md) | The shortest working loop |
| 4 | [Write `yeast.yaml`](getting-started/write-yeast-yaml.md) | How to edit RAM, CPU, disk, image, users, provisioning, and networking |
| 5 | [First VM](getting-started/first-vm.md) | A slower first-machine walkthrough |
| 6 | [Yeast Labs](labs/index.md) | Guided tutorials that build confidence |

## The Short Version

Run these commands from a new project folder:

```bash
mkdir my-lab
cd my-lab
yeast init --template ubuntu-basic
yeast up
yeast ssh web
```

When you are done:

```bash
yeast down
yeast destroy
```

`yeast up` downloads supported cloud images automatically when they are missing.

## The File You Edit Most

Most Yeast work happens in `yeast.yaml`.

This small file says what machines you want:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
```

Common edits:

| Want to change | Edit |
|---|---|
| RAM | `memory: 2048` |
| CPU count | `cpus: 2` |
| disk size | `disk_size: 30G` |
| image | `image: debian-12` |
| VM name | `name: web` |
| host SSH port | `ssh_port: 2222` |

Read [Write `yeast.yaml`](getting-started/write-yeast-yaml.md) when you want to understand every field without jumping straight into reference docs.

## What Yeast Handles

- project-local VM definitions with `yeast.yaml`
- trusted image discovery and caching
- QEMU/KVM startup
- cloud-init seed generation
- SSH readiness
- package, file, and shell provisioning
- stopped-VM snapshots and restore
- one private lab network per project
- guest control with `ssh`, `exec`, `copy`, `logs`, and `inspect`
- stable JSON output and lifecycle events

## What Yeast Is Not

Yeast is not a cloud platform, container orchestrator, web dashboard, or replacement for Kubernetes.

It is a Linux-first local VM engine for repeatable real-machine labs.

## Current Release

The current docs target Yeast `v1.1.2`.

Start with [Quickstart](getting-started/quickstart.md), or jump to the [command reference](reference/commands.md).
