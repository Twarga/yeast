# Yeast Quickstart

This guide gets one Ubuntu VM running with Yeast `v0.3` and then shows the first provisioning workflow.

Yeast is still Linux-first and QEMU/KVM-first. `v0.3` adds post-boot provisioning, but snapshots, private lab networking, and templates are still out of scope.

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
- run post-boot provisioning when the config contains `provision`
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

## 7. Rerun Provisioning

If you edit provisioned files or shell commands and want to apply them again without recreating the VM:

```bash
yeast provision web
```

`yeast provision`:

- requires a running reachable VM
- reruns the same merged project-level and instance-level plan as `yeast up`
- does not recreate disks or reboot the guest by itself

## 8. Stop The VM

```bash
yeast down
```

This stops tracked running VMs but keeps runtime files and disks.

## 9. Destroy Runtime Files

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

## First Provisioning Demo

The reference `v0.3` demo is:

```text
examples/caddy-single-vm
```

It installs Caddy, copies a static page, writes a Caddyfile, and starts the service.
