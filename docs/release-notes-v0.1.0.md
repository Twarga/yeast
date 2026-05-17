# Yeast v0.1.0 Release Notes

Status: Prerelease
Release type: first v2 lifecycle release for early testers
Target platform: Linux amd64

## Summary

Yeast v0.1.0 is the first clean lifecycle release of the v2 rebuild.

It gives Linux users a project-based QEMU/KVM workflow:

```text
init -> pull -> up -> status -> ssh -> down -> destroy
```

The goal of this release is not to ship the full Yeast vision. The goal is to prove the local VM engine: project metadata, config loading, trusted image cache, cloud-init bootstrap, QEMU runtime lifecycle, SSH readiness, state tracking, human output, and JSON output.

## Install

The intended v0.1.0 user path is:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/v0.1.0/install.sh | YEAST_REF=v0.1.0 bash
```

The install script passed the `M10-T3A` hardening gate for supported Linux package-manager paths, but the full manual KVM lifecycle checklist is still required before treating this as broadly proven.

Manual source build:

```bash
git clone https://github.com/Twarga/yeast.git
cd yeast
go build -o yeast ./cmd/yeast
sudo install -m 0755 yeast /usr/local/bin/yeast
```

Release artifact build:

```bash
bash scripts/build-release.sh v0.1.0
```

Expected local artifact files:

```text
dist/yeast-linux-amd64
dist/yeast-linux-amd64.sha256
```

The `dist/` directory is not committed to git. Upload these files to the GitHub release.

## Requirements

- Linux host
- KVM access
- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage` or `mkisofs`
- `ssh`
- SSH public key at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`
- Go 1.25+ for source builds

## What Is Included

### Core Commands

- `yeast doctor`
- `yeast init`
- `yeast pull --list`
- `yeast pull <image>`
- `yeast up`
- `yeast status`
- `yeast ssh [instance]`
- `yeast down`
- `yeast destroy`
- `yeast version`

### VM Lifecycle

Yeast can:

- create project metadata
- read and validate `yeast.yaml`
- pull trusted Ubuntu cloud images
- verify image checksums
- create qcow2 overlay disks
- render cloud-init data
- create seed ISOs
- start QEMU/KVM VMs
- wait for SSH readiness
- track runtime state
- reconcile dead processes
- stop running VMs
- destroy tracked runtime files

### Supported Images

- `ubuntu-22.04`
- `ubuntu-24.04`

### Output

Human output is styled with Lip Gloss.

JSON output is available for core non-interactive commands:

```bash
yeast status --json
```

JSON output remains plain JSON with no terminal styling.

## Example

```bash
mkdir yeast-demo
cd yeast-demo
yeast init
yeast doctor
yeast pull ubuntu-24.04
yeast up
yeast status
yeast ssh web
yeast down
yeast destroy
```

Example project:

- `examples/ubuntu-basic`

## Known Limitations

This is an early release.

Not included yet:

- provisioning packages/files/shell workflows
- snapshots and restore
- private VM-to-VM lab networking
- guest `exec`, `copy`, `logs`, or `inspect`
- templates
- daemon or web API
- LabsBackery integration contract
- Yeast MCP
- Twarga Cloud worker mode
- Windows/macOS host support
- VirtualBox backend

## Verification

Automated checks:

```bash
bash scripts/test-fast.sh
go test ./... -count=1
go build ./...
git diff --check
```

Manual Linux/KVM checklist still required on a real host:

```bash
yeast doctor
yeast init
yeast pull ubuntu-24.04
yeast up
yeast status
yeast ssh web
yeast down
yeast up
yeast destroy
```

This release is marked as a prerelease because the manual host-dependent checklist is still pending.

## Documentation

- `README.md`
- `docs/quickstart.md`
- `docs/installation.md`
- `docs/config-reference.md`
- `docs/troubleshooting.md`
- `docs/known-limitations.md`
- `docs/architecture-overview.md`
- `docs/charm-cli-plan.md`
