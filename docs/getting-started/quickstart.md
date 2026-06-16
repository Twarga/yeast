# Quickstart

This guide starts one Ubuntu VM with Yeast.

You will:

- create a Yeast project
- let `yeast up` download the trusted cloud image if needed
- boot a real QEMU/KVM VM
- SSH into the guest
- stop and destroy the project safely

Expected time: 5-10 minutes after the image is cached.

## What You Need

- a Linux host
- working KVM access
- Yeast installed
- an SSH public key

If you are not sure, start with:

```bash
yeast doctor
```

## 1. Check The Host

```bash
yeast doctor
```

Fix any blockers before continuing.

## 2. Create A Project

Run this from a normal host terminal:

```bash
mkdir my-lab
cd my-lab
yeast init --template ubuntu-basic
```

This creates:

| Path | Purpose |
|---|---|
| `yeast.yaml` | VM configuration |
| `.yeast/project.json` | project identity |

## 3. Look At `yeast.yaml`

```bash
sed -n '1,120p' yeast.yaml
```

You should see one VM named `web`.

The important part looks like this:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
```

This means:

| Field | Meaning |
|---|---|
| `name: web` | The VM is called `web`. Commands use this name, for example `yeast ssh web`. |
| `image: ubuntu-24.04` | The VM uses the trusted Ubuntu 24.04 cloud image. |
| `memory: 1024` | The VM gets 1024 MiB of RAM. |
| `cpus: 1` | The VM gets one vCPU. |
| `disk_size: 20G` | The VM disk starts at 20 GiB when the disk is created. |

## 4. Start The VM

```bash
yeast up
```

Yeast will:

1. validate `yeast.yaml`
2. create the VM disk
3. generate cloud-init data
4. start QEMU/KVM
5. wait for SSH
6. run provisioning if configured

If the image is not cached yet, Yeast downloads it during this step.

## 5. Check Status

```bash
yeast status
```

You should see one running instance named `web`.

For scripts:

```bash
yeast status --json
```

## 6. SSH Into The VM

```bash
yeast ssh web
```

Inside the VM:

```bash
hostname
whoami
ip addr
exit
```

Expected result:

- `hostname` prints `web` unless you changed `hostname`
- `whoami` prints `yeast` unless you changed `user`
- `ip addr` shows normal Linux network interfaces

## 7. Stop Or Destroy

Stop the VM but keep its disk:

```bash
yeast down
```

Delete the project runtime files and disk:

```bash
yeast destroy
```

!!! warning
    `yeast destroy` removes tracked VM runtime files and disks for this project.

## Common Next Checks

Read VM logs:

```bash
yeast logs web --tail 80
```

Inspect one VM:

```bash
yeast inspect web
```

See terminal docs:

```bash
yeast docs --list
```

List supported images:

```bash
yeast pull --list
```

Manual `yeast pull <image>` is optional for supported auto-download images. Use it when you want to warm the cache before `yeast up`.

## Next Step

Read [Write `yeast.yaml`](write-yeast-yaml.md) next. That page explains how to edit RAM, CPU, disk size, images, provisioning, users, and networks.
