# Yeast v0.3.0 Release Notes

Status: Ready
Release type: Minor feature release
Target platform: Linux amd64

## Summary

Yeast v0.3.0 is the first provisioning release of the v2 rebuild.

It keeps the product Linux-first and intentionally narrow, but it closes the gap between "VM booted" and "VM is actually useful" by adding post-boot provisioning and a rerun command.

This release adds:

- project-level and instance-level `provision` config
- package installation over SSH
- file copy over SSH
- shell command execution over SSH
- automatic provisioning during `yeast up`
- `yeast provision [instance]` for reruns
- provisioning status and per-instance `provision.log`
- a real Caddy demo and provisioning smoke coverage

This release does **not** add snapshots, private networking, guest control commands, LabsBackery integration, MCP integration, or cloud worker behavior.

## What Changed

### Provisioning Contract

Provisioning now supports:

- `provision.packages`
- `provision.files`
- `provision.shell`

Provisioning can be declared:

- once at the top level for project-wide steps
- per instance for instance-specific steps

Merge order:

```text
project packages -> instance packages
project files    -> instance files
project shell    -> instance shell
```

Execution model:

- cloud-init remains bootstrap only
- packages/files/shell run post-boot over SSH
- file sources resolve relative to the project root unless absolute

### Automatic Provisioning In `yeast up`

`yeast up` now:

- boots the guest
- waits for SSH readiness
- runs the merged provisioning plan
- writes `provision.log`
- stores provisioning state in project state

Provisioning states now include:

- `not_started`
- `running`
- `provisioned`
- `failed`

On provisioning failure:

- the instance remains running
- state is marked `failed`
- `last_error` is persisted
- the failure is visible for rerun and debugging

### `yeast provision`

`yeast provision [instance]` reruns the same merged post-boot plan against an existing running reachable VM.

It:

- requires a running instance
- does not recreate disks
- does not regenerate cloud-init
- does not reboot the guest unless a user-authored shell command does that

### Example And Docs

This release adds:

- `examples/caddy-single-vm`
- updated quickstart for provisioning
- updated config reference for live provisioning behavior
- updated known limitations for the current product state

## Verification

### Automated

The following checks passed before release prep:

```bash
go test ./... -count=1
git diff --check
bash -n scripts/manual-smoke.sh
```

### Smoke Validation

The smoke script now supports:

```bash
TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=positive ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

The full smoke workflow covers:

- `doctor`
- `init`
- `pull`
- `up`
- `status`
- `status --json`
- direct SSH checks
- Caddy service verification
- guest HTTP content verification
- `yeast provision`
- `down`
- restart
- `destroy`
- negative-path JSON checks for:
  - uninitialized `status`
  - repeated `init`
  - unsupported `pull`
  - corrupt project metadata
  - state project mismatch
  - missing config
  - invalid `disk_size`
  - invalid `hostname`
  - invalid `ssh_port`
  - duplicate requested `ssh_port`
  - missing provision source file

## Example v0.3.0 Config

```yaml
version: 1
provision:
  packages:
    - caddy
  files:
    - source: ./site/index.html
      destination: /var/www/html/index.html
      permissions: "0644"
    - source: ./site/Caddyfile
      destination: /etc/caddy/Caddyfile
      permissions: "0644"
  shell:
    - sudo systemctl enable caddy
    - sudo systemctl restart caddy
instances:
  - name: web
    hostname: caddy-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    ssh_port: 2205
    user: yeast
    sudo: nopasswd
```

## Not Included

- snapshots and restore
- private VM-to-VM networking
- guest `exec`, `copy`, `logs`, or `inspect`
- template workflows
- LabsBackery runtime contract
- Yeast MCP contract
- Twarga Cloud worker mode
