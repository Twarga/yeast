# Yeast Quickstart

This guide gets one Ubuntu VM running with Yeast `v0.5`, shows the provisioning and stopped-VM reset flow, and then points to the first two-VM private lab example.

Yeast is still Linux-first and QEMU/KVM-first. `v0.5` adds the first narrow private lab network: one project-level network, one static IPv4 per attached instance, and management SSH kept separate from lab traffic.

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

## 9. Create A Snapshot

Snapshots in `v0.4` are stopped-VM only. Create the baseline after provisioning has finished and the VM is stopped.

```bash
yeast snapshot web clean --description "Provisioned baseline"
yeast snapshots web
```

This stores a snapshot copy under the project runtime directory and records metadata in state.

## 10. Restore A Snapshot

Bring the VM back up, change something, stop it again, then restore:

```bash
yeast up
yeast ssh web
# make a change inside the VM
exit
yeast down
yeast restore web clean
```

After restore, boot the VM again:

```bash
yeast up
```

## 11. Delete A Snapshot

When you no longer need the stored baseline:

```bash
yeast down
yeast delete-snapshot web clean
```

## 12. First Two-VM Private Lab

The first `v0.5` networking example is:

```text
examples/two-vm-lab
```

It shows:

- one project-level private lab network
- two VMs on that network
- separate management SSH ports
- visible `LAB IP` values in `yeast status`

Management SSH still uses host-forwarded ports such as `127.0.0.1:2205`. The private lab network is separate guest-to-guest traffic inside the VMs.

Typical verification:

```bash
yeast ssh attacker
ip addr show yeastlab0
ping -c 2 10.10.10.20
```

## 13. Destroy Runtime Files

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

## First Provisioning And Reset Demo

The reference `v0.4` demo is:

```text
examples/caddy-single-vm
```

It covers:

- package/file/shell provisioning
- manual verification inside the guest
- stopped-VM snapshot create
- break-and-restore reset flow

It installs Caddy, copies a static page, writes a Caddyfile, and starts the service.
