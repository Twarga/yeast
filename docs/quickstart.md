# Yeast Quickstart

This guide gets one Ubuntu VM running with Yeast v0.1.

Yeast v0.1 is Linux-first and QEMU/KVM-first. It does not support snapshots, provisioning workflows, private lab networking, or templates yet.

## 1. Check The Host

```bash
yeast doctor
```

All blocker checks must pass before `yeast up` can work.

Yeast checks for:

- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage` or `mkisofs`
- `ssh`
- `/dev/kvm`
- an SSH public key at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`
- Yeast image cache directory

The cache directory warning is not fatal. Yeast creates it during image pull.

## 2. Create A Project

```bash
mkdir yeast-ubuntu-basic
cd yeast-ubuntu-basic
yeast init
```

This creates:

- `yeast.yaml`
- `.yeast/project.json`

The project metadata gives this folder a stable project ID. Runtime files are stored under `~/.yeast/projects/<project-id>/`.

## 3. Pull A Trusted Image

List supported images:

```bash
yeast pull --list
```

Pull Ubuntu 24.04:

```bash
yeast pull ubuntu-24.04
```

Yeast downloads the image into the shared image cache and verifies its SHA-256 checksum before storing it.

## 4. Start The VM

```bash
yeast up
```

Yeast will:

- load `yeast.yaml`
- lock project state
- check the cached image exists
- render cloud-init `user-data` and `meta-data`
- create a seed ISO
- create a qcow2 overlay disk
- start QEMU/KVM
- wait for SSH readiness
- save runtime state

## 5. Check Status

```bash
yeast status
```

For scripts:

```bash
yeast status --json
```

## 6. SSH Into The VM

```bash
yeast ssh web
```

If the project has exactly one running VM, `yeast ssh` can be run without an instance name.

## 7. Stop The VM

```bash
yeast down
```

This stops tracked running VMs but keeps runtime files and disks.

## 8. Destroy Runtime Files

```bash
yeast destroy
```

This removes tracked runtime files for the project. It does not delete the shared image cache.

## JSON Output

Most non-interactive commands support `--json`:

```bash
yeast doctor --json
yeast pull --list --json
yeast status --json
```

`yeast ssh` is interactive and should not be used as a JSON workflow.
