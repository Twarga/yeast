# Yeast Quickstart

This guide gets one Ubuntu VM running with Yeast `v0.7`, shows the provisioning and stopped-VM reset flow, proves the first guest-control commands, and then points to the first two-VM private lab template.

Yeast is still Linux-first and QEMU/KVM-first. `v0.7` keeps the narrow `v0.6` VM engine and adds project starters through `yeast init --template`.

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

## 3. Or Start From A Template

List built-in templates:

```bash
yeast init --list-templates
```

Create a Caddy project from a built-in template:

```bash
mkdir yeast-caddy-demo
cd yeast-caddy-demo
yeast init --template caddy-single-vm
```

Current built-ins:

- `ubuntu-basic`
- `caddy-single-vm`
- `two-vm-lab`

You can also initialize from a local template directory:

```bash
yeast init --template ../my-template
```

Templates are project starters. After initialization, `yeast.yaml` and generated files are normal editable project files.

## 4. Pull A Trusted Image

List supported images:

```bash
yeast pull --list
```

Pull Ubuntu 24.04:

```bash
yeast pull ubuntu-24.04
```

Yeast downloads the image into the shared image cache and verifies its SHA-256 checksum before storing it.

## 5. Start The VM

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

## 6. Check Status

```bash
yeast status
```

For scripts:

```bash
yeast status --json
```

## 7. SSH Into The VM

```bash
yeast ssh web
```

If the project has exactly one running VM, `yeast ssh` can be run without an instance name.

## 8. Rerun Provisioning

If you edit provisioned files or shell commands and want to apply them again without recreating the VM:

```bash
yeast provision web
```

`yeast provision`:

- requires a running reachable VM
- reruns the same merged project-level and instance-level plan as `yeast up`
- does not recreate disks or reboot the guest by itself

## 9. Stop The VM

```bash
yeast down
```

This stops tracked running VMs but keeps runtime files and disks.

## 10. Create A Snapshot

Snapshots in `v0.4` are stopped-VM only. Create the baseline after provisioning has finished and the VM is stopped.

```bash
yeast snapshot web clean --description "Provisioned baseline"
yeast snapshots web
```

This stores a snapshot copy under the project runtime directory and records metadata in state.

## 11. Restore A Snapshot

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

## 12. Delete A Snapshot

When you no longer need the stored baseline:

```bash
yeast down
yeast delete-snapshot web clean
```

## 13. First Guest-Control Commands

Run one command inside the guest:

```bash
yeast exec web -- whoami
```

Copy a file into the guest:

```bash
yeast copy web --to-guest ./artifact.txt /home/yeast/artifact.txt
```

Copy a file back out:

```bash
yeast copy web --from-guest /home/yeast/artifact.txt ./artifact-out.txt
```

Inspect one instance:

```bash
yeast inspect web
```

Read the VM runtime log:

```bash
yeast logs web --tail 20
```

`yeast ssh` is still the interactive terminal flow. The new `v0.6` commands are for one-shot operations and structured automation.

## 14. First Two-VM Private Lab

The first private networking template is:

```bash
mkdir yeast-two-vm-lab
cd yeast-two-vm-lab
yeast init --template two-vm-lab
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

## 15. Destroy Runtime Files

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
