# New Yeast Docs Plan

Status: in progress
Audience: public Yeast users and contributors
Scope: public documentation, public Yeast tutorial labs, embedded terminal docs

## Goal

Remake the Yeast documentation so a new user can understand, install, run, and automate Yeast without guessing.

The docs should be:

- accurate to the current code
- easy to follow
- organized around user journeys
- copy/paste safe
- honest about limitations
- friendly without becoming academic
- maintainable across releases

## Non-Goals

The public Yeast docs are not the private DevOps bootcamp.

Public Yeast tutorials teach Yeast itself:

- projects
- `yeast.yaml`
- images
- VM lifecycle
- cloud-init
- provisioning
- status/logs/inspect
- JSON output
- snapshots
- private networking
- templates

They should not become long-form DevOps course chapters.

## Current Problems To Fix

### Multiple Documentation Surfaces

Current docs exist in several places:

- `docs-site/` for the VitePress site
- `docs/` for legacy/static docs
- `tutorials/` for older long tutorials
- `internal/docs/embedded/` for terminal docs
- `README.md` for project overview

This creates drift. The new docs should make one public source of truth.

### Stale Or Unsupported Features

Several existing docs and examples mention:

```yaml
ports:
  - host_port: 8080
    guest_port: 80
```

The current `internal/config/model.go` does not define `ports`, `host_port`, or `guest_port`.

Decision for this docs pass:

- Do not teach `ports:` in public docs.
- Remove or archive public examples that depend on `ports:` until the feature exists.
- Track `ports:` as a future product feature if desired.

### Invalid Commands

Public docs should not reference commands or flags that do not exist in the current CLI.

Known corrections:

- Use `yeast version`, not `yeast --version`.
- Do not document `yeast clean` unless a public command is implemented.
- Do not document `yeast logs --follow`; current logs support `--tail`.

## Recommended Docs Engine

Recommendation: MkDocs Material.

Why:

- Markdown-first
- excellent navigation and search
- strong fit for CLI/project docs
- simple GitHub Pages deployment
- less frontend overhead than VitePress
- good admonitions, tabs, code blocks, and strict builds

Migration should happen after the content structure is agreed.

## Target Public Docs Tree

```text
docs/
  index.md

  getting-started/
    what-is-yeast.md
    installation.md
    quickstart.md
    first-vm.md

  concepts/
    projects.md
    images.md
    lifecycle.md
    cloud-init.md
    provisioning.md
    networking.md
    snapshots.md
    guest-control.md
    state-and-files.md

  guides/
    templates.md
    caddy-single-vm.md
    two-vm-lab.md
    rerun-provisioning.md
    automate-with-json.md
    manage-image-cache.md
    update-yeast.md
    release-smoke-v1.1.0.md

  reference/
    commands.md
    yeast-yaml.md
    images.md
    json-output.md
    events.md
    limitations.md

  troubleshooting/
    index.md
    kvm.md
    ssh.md
    images.md
    provisioning.md
    networking.md
    snapshots.md

  labs/
    index.md
    01-first-vm-first-ssh.md
    02-cloud-init-basics.md
    03-provisioning-after-boot.md
    04-status-logs-inspect-json.md
    05-snapshots-and-restore.md
    06-multi-vm-private-networking.md
    07-templates-and-reusable-labs.md

  releases/
    v1.1.0.md
    archive.md

  archive/
```

## V1.1 Feature Coverage Checklist

Every v1.1 feature must have a public docs home before the rewrite is considered complete.

### CLI Commands

| Command | Docs Home | Required Coverage |
|---|---|---|
| `yeast doctor` | `getting-started/installation.md`, `troubleshooting/index.md` | host readiness, KVM, QEMU, SSH key checks |
| `yeast init` | `getting-started/quickstart.md`, `concepts/projects.md`, `guides/templates.md`, `reference/commands.md` | default project init, built-in templates, local templates |
| `yeast pull` | `concepts/images.md`, `reference/images.md`, `reference/commands.md` | `--list`, `--cached`, auto images, manual images |
| `yeast images clean` | `guides/manage-image-cache.md`, `reference/commands.md` | one-image cleanup, `--all`, `--dry-run` |
| `yeast up` | `concepts/lifecycle.md`, `reference/commands.md`, labs | normal boot, `--no-provision`, `--reprovision`, `--sequential`, `--profile` |
| `yeast down` | `concepts/lifecycle.md`, `reference/commands.md`, labs | graceful stop, preserves project disks |
| `yeast destroy` | `concepts/lifecycle.md`, `reference/commands.md`, labs | destructive cleanup warning |
| `yeast status` | `concepts/guest-control.md`, `labs/04-status-logs-inspect-json.md`, `reference/commands.md` | human status, JSON status |
| `yeast inspect` | `concepts/guest-control.md`, `labs/04-status-logs-inspect-json.md`, `reference/commands.md` | detailed instance state |
| `yeast logs` | `concepts/guest-control.md`, `troubleshooting/index.md`, `reference/commands.md` | runtime logs, `--tail` |
| `yeast ssh` | `getting-started/first-vm.md`, `concepts/guest-control.md`, `reference/commands.md` | interactive guest shell |
| `yeast exec` | `concepts/guest-control.md`, `labs/05-snapshots-and-restore.md`, `reference/commands.md` | one-shot commands, `--timeout` |
| `yeast copy` | `concepts/guest-control.md`, `labs/04-status-logs-inspect-json.md`, `reference/commands.md` | `--to-guest`, `--from-guest`, `--timeout` |
| `yeast provision` | `concepts/provisioning.md`, `guides/rerun-provisioning.md`, `reference/commands.md` | rerun provisioning on running instances |
| `yeast snapshot` | `concepts/snapshots.md`, `labs/05-snapshots-and-restore.md`, `reference/commands.md` | stopped-VM snapshot, description |
| `yeast snapshots` | `concepts/snapshots.md`, `reference/commands.md` | list snapshots |
| `yeast restore` | `concepts/snapshots.md`, `labs/05-snapshots-and-restore.md`, `reference/commands.md` | stopped-VM restore, destructive warning |
| `yeast delete-snapshot` | `concepts/snapshots.md`, `reference/commands.md` | remove snapshot metadata and disk copy |
| `yeast update` | `guides/update-yeast.md`, `reference/commands.md` | `--check`, `--force`, `--version`, checksum-verified release update |
| `yeast docs` | `getting-started/quickstart.md`, `reference/commands.md` | terminal docs topics, `--list`, no JSON support |
| `yeast completion` | `reference/commands.md` | shell completion generation |
| `yeast version` | `getting-started/installation.md`, `reference/commands.md` | installed version check |

### Global Flags

| Flag | Docs Home | Required Coverage |
|---|---|---|
| `--json` | `reference/json-output.md`, `labs/04-status-logs-inspect-json.md` | machine-readable output |
| `--events` | `reference/events.md`, `labs/04-status-logs-inspect-json.md` | JSON Lines lifecycle events, requires JSON |
| `--quiet`, `-q` | `reference/commands.md` | suppress progress output |
| `--help`, `-h` | `reference/commands.md` | per-command help |

### `yeast.yaml` Fields

| Field | Docs Home | Required Coverage |
|---|---|---|
| `version` | `reference/yeast-yaml.md` | required, currently `1` |
| `management_host` | `reference/yeast-yaml.md`, `concepts/networking.md` | management bind host, valid IPv4, default behavior |
| `networks` | `concepts/networking.md`, `reference/yeast-yaml.md` | at most one private network |
| `networks[].name` | `reference/yeast-yaml.md` | unique project network name |
| `networks[].cidr` | `concepts/networking.md`, `reference/yeast-yaml.md` | IPv4 CIDR only |
| `provision` | `concepts/provisioning.md`, `reference/yeast-yaml.md` | top-level provision shared by instances |
| `provision.packages` | `concepts/provisioning.md`, `reference/yeast-yaml.md` | package install step |
| `provision.files` | `concepts/provisioning.md`, `reference/yeast-yaml.md` | host-to-guest file copy |
| `provision.files[].source` | `reference/yeast-yaml.md` | local source path |
| `provision.files[].destination` | `reference/yeast-yaml.md` | guest destination path |
| `provision.files[].permissions` | `reference/yeast-yaml.md` | octal file mode |
| `provision.shell` | `concepts/provisioning.md`, `reference/yeast-yaml.md` | shell commands run after package/file steps |
| `instances` | `concepts/projects.md`, `reference/yeast-yaml.md` | one or more VM definitions |
| `instances[].name` | `reference/yeast-yaml.md` | unique instance name, naming rules |
| `instances[].hostname` | `concepts/cloud-init.md`, `reference/yeast-yaml.md` | guest hostname |
| `instances[].image` | `concepts/images.md`, `reference/yeast-yaml.md` | supported image name |
| `instances[].memory` | `reference/yeast-yaml.md` | MiB, minimum 128 when set |
| `instances[].cpus` | `reference/yeast-yaml.md` | vCPU count, minimum 1 when set |
| `instances[].disk_size` | `reference/yeast-yaml.md` | overlay disk size, applies on first disk creation |
| `instances[].ssh_port` | `concepts/networking.md`, `reference/yeast-yaml.md` | host management SSH port |
| `instances[].user` | `concepts/cloud-init.md`, `reference/yeast-yaml.md` | Linux user, default behavior |
| `instances[].sudo` | `concepts/cloud-init.md`, `reference/yeast-yaml.md` | `none`, `password`, `nopasswd` |
| `instances[].env` | `concepts/provisioning.md`, `reference/yeast-yaml.md` | environment values for guest/cloud-init behavior |
| `instances[].user_data` | `concepts/cloud-init.md`, `reference/yeast-yaml.md` | custom cloud-init user data behavior and cautions |
| `instances[].networks` | `concepts/networking.md`, `reference/yeast-yaml.md` | at most one private network attachment |
| `instances[].networks[].name` | `reference/yeast-yaml.md` | attached network name |
| `instances[].networks[].ipv4` | `concepts/networking.md`, `reference/yeast-yaml.md` | static IPv4 inside network CIDR |
| `instances[].provision` | `concepts/provisioning.md`, `reference/yeast-yaml.md` | instance-specific provision steps |

Unsupported in v1.1 public docs:

- `ports`
- `host_port`
- `guest_port`

These appear in some older docs/examples but are not part of the current config model. Do not teach them as supported until implemented.

## Current Supported Images

This list must be checked against `internal/images/manifest.go` and `yeast pull --list` during the docs rewrite.

### Auto-Download Images

These images have direct URLs and checksums in the manifest. They are intended to be downloadable with `yeast pull <image>`.

| Image | Category | Cloud-Init | Approx Size | Description |
|---|---|---:|---:|---|
| `debian-12` | General Purpose | yes | ~400MB | Debian 12 Bookworm, stable/minimal |
| `debian-13` | General Purpose | yes | ~400MB | Debian 13 Trixie, newer packages |
| `ubuntu-22.04` | General Purpose | yes | ~500MB | Ubuntu 22.04 LTS, legacy LTS |
| `ubuntu-24.04` | General Purpose | yes | ~600MB | Ubuntu 24.04 LTS, default choice |
| `fedora-41` | DevOps & Cloud | yes | ~500MB | Fedora 41, balanced stable/bleeding-edge |
| `fedora-42` | DevOps & Cloud | yes | ~500MB | Fedora 42, newer tooling/kernels |
| `alma-9` | Enterprise | yes | ~1GB | AlmaLinux 9, RHEL-compatible |
| `centos-stream-9` | Enterprise | yes | ~800MB | CentOS Stream 9, upstream RHEL |
| `rocky-9` | Enterprise | yes | ~1GB | Rocky Linux 9, RHEL-compatible |

### Manual/Setup-Only Images

These images are searchable and listed by Yeast, but require manual download or preparation before use.

| Image | Category | Cloud-Init | Approx Size | Notes |
|---|---|---:|---:|---|
| `amazon-linux-2023` | Enterprise | yes | ~1.4GB | Manual QCOW2 download from Amazon Linux |
| `opensuse-leap-15.6` | Enterprise | no | ~1GB | Manual setup/conversion required |
| `kali-2026.1` | Security | no | ~3.6GB | Manual QEMU image download, default `kali/kali` credentials |
| `parrot-security-7.1` | Security | no | ~11.7GB | Manual QCOW2 setup |
| `alpine-3.21` | Minimal | no | ~50MB | Manual ISO/QCOW2 setup |
| `arch-linux` | Niche | no | ~800MB | Manual arch-boxes or custom QCOW2 setup |
| `nixos-24.11` | Niche | no | ~1GB | Manual NixOS generator/download setup |

### Image Docs Requirements

The new image docs must explain:

- `yeast pull --list` shows all supported images.
- `yeast pull --cached` shows locally cached images.
- Auto-download images can be pulled directly.
- Manual images print setup instructions instead of downloading.
- Cloud-init support affects automatic user, SSH, and provisioning behavior.
- Image cache cleanup uses `yeast images clean`.

## Public Yeast Mini Bootcamp

The public Yeast mini bootcamp is a 7-lab documentation tutorial path.

Purpose:

```text
Teach Yeast.
```

Not DevOps.

### Lab 01: First VM, First SSH

Teaches:

- creating a project
- initializing from a template
- pulling an image
- booting one VM
- connecting with SSH
- stopping and destroying safely

Core commands:

```bash
mkdir yeast-lab-01
cd yeast-lab-01
yeast init --template ubuntu-basic
yeast pull ubuntu-24.04
yeast up
yeast status
yeast ssh web
yeast down
yeast destroy
```

### Lab 02: Cloud-Init Basics

Teaches:

- how Yeast prepares first boot
- hostname
- default user
- SSH key access
- sudo policy
- what cloud-init does before provisioning

Core files:

- `yeast.yaml`
- generated cloud-init data, explained conceptually

### Lab 03: Provisioning After Boot

Teaches:

- top-level `provision`
- instance-level `provision`
- `packages`
- `files`
- `shell`
- `yeast provision`
- `yeast up --reprovision`
- `yeast up --no-provision`

Core commands:

```bash
yeast init --template caddy-single-vm
yeast up
yeast exec web -- systemctl is-active caddy
yeast logs web --tail 80
yeast provision web
yeast down
yeast destroy
```

### Lab 04: Status, Logs, Inspect, JSON

Teaches:

- checking project state
- reading VM logs
- inspecting one instance
- machine-readable JSON output
- JSON events for automation

Core commands:

```bash
yeast status
yeast logs web --tail 50
yeast inspect web
yeast status --json
yeast up --json --events
```

### Lab 05: Snapshots And Restore

Teaches:

- stopped-VM snapshots
- per-instance snapshot metadata
- restore workflow
- reset points
- destructive restore warning

Core commands:

```bash
yeast down
yeast snapshot web baseline --description "Clean baseline"
yeast snapshots web
yeast up
yeast exec web -- touch /home/yeast/marker
yeast down
yeast restore web baseline
yeast up
yeast exec web -- test ! -e /home/yeast/marker
```

### Lab 06: Multi-VM Private Networking

Teaches:

- one project private network
- static IPv4 assignment
- VM-to-VM connectivity
- management SSH versus private lab traffic

Core commands:

```bash
yeast init --template two-vm-lab
yeast up
yeast status
yeast exec attacker -- ping -c 2 10.10.10.20
yeast exec target -- ping -c 2 10.10.10.10
yeast down
yeast destroy
```

### Lab 07: Templates And Reusable Labs

Teaches:

- listing built-in templates
- initializing from a template
- understanding copied files
- editing the generated project
- local template concept

Core commands:

```bash
yeast init --list-templates
mkdir reusable-lab
cd reusable-lab
yeast init --template caddy-single-vm
find . -maxdepth 3 -type f | sort
sed -n '1,160p' yeast.yaml
```

## Public Lab Writing Style

Use documentation tutorial style:

- direct
- friendly
- practical
- short enough to finish
- enough explanation to understand Yeast
- no deep DevOps course content

Every lab should include:

- goal
- what you will build
- before you start
- steps
- expected result
- verification
- cleanup
- what you learned
- next lab

Template:

```markdown
# Lab NN: Title

Short intro.

You will learn:

- Yeast concept
- Yeast command
- Yeast workflow

## What You Will Build

Small diagram.

## Before You Start

Run:

```bash
yeast doctor
```

## Step 1: Create The Project

## Step 2: Start The VM Or VMs

## Step 3: Verify It Worked

## Step 4: Try One Small Change

## Step 5: Clean Up

## What You Learned

## Next Lab
```

## Source Of Truth

Docs must be checked against:

- `cmd/yeast/*.go` for commands and flags
- `internal/config/model.go` for config fields
- `internal/config/validate.go` for validation rules
- `internal/images/manifest.go` for image names and manual/auto behavior
- `internal/templates/builtin/` for built-in templates
- `internal/output/` for human and JSON output behavior
- `internal/docs/embedded/` for terminal docs topics

## Embedded Terminal Docs

Keep `yeast docs` short.

Terminal docs are offline survival docs, not a full website replacement.

Recommended topics:

- `quickstart`
- `installation`
- `config`
- `troubleshooting`
- `release-smoke`

## Migration Phases

### Phase 1: Stabilize Existing Docs

- Commit current trust fixes.
- Remove invalid commands.
- Stop hiding broken VitePress links.
- Remove unsupported `ports:` from public docs or mark as future.

### Phase 2: Create New Public Docs Tree

- Add new docs folders.
- Create the 7 public Yeast labs.
- Create core reference pages.
- Archive old stale docs.

### Phase 3: Replace VitePress

- Add `mkdocs.yml`.
- Add Material theme config.
- Move source docs into MkDocs structure.
- Remove or deprecate `docs-site/`.
- Update GitHub Pages workflow.

Status: active. MkDocs Material is now the public docs engine, the new docs source lives in `docs/`, and the Pages workflow should publish MkDocs under `/docs/` with the custom landing page at the site root.

### Phase 4: Validate

Run:

```bash
mkdocs build --strict
go test ./internal/docs -count=1
git diff --check
```

Later:

- link checker
- spell checker
- Playwright browser smoke
- screenshot visual QA

## Acceptance Criteria

The new docs are ready when:

- a new user can install and start a VM in under 10 minutes
- public examples are copy/paste safe
- command docs match `yeast help`
- config docs match the Go config model
- unsupported features are not documented as real
- public Yeast labs teach Yeast, not DevOps
- the docs build fails on broken links
- embedded docs still pass tests
