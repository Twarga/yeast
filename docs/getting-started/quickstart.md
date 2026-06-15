# Quickstart

This guide starts one Ubuntu VM with Yeast.

You will:

- create a Yeast project
- download a trusted cloud image
- boot a real QEMU/KVM VM
- SSH into the guest
- stop and destroy the project safely

Expected time: 5-10 minutes after the image is cached.

## 1. Check The Host

```bash
yeast doctor
```

Fix any blockers before continuing.

## 2. Create A Project

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

## 3. Pull An Image

```bash
yeast pull ubuntu-24.04
```

Images are cached under `~/.yeast/cache/images` and reused across projects.

You can list available images with:

```bash
yeast pull --list
```

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

## 5. Check Status

```bash
yeast status
```

You should see one running instance named `web`.

## 6. SSH Into The VM

```bash
yeast ssh web
```

Inside the VM:

```bash
hostname
whoami
exit
```

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

## Next Step

Follow [First VM](first-vm.md) for a slower walkthrough, or start the [Yeast Labs](../labs/index.md).
