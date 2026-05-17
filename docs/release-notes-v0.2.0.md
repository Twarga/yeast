# Yeast v0.2.0 Release Notes

Status: Ready
Release type: Minor feature release
Target platform: Linux amd64

## Summary

Yeast v0.2.0 is the first post-lifecycle hardening release of the v2 rebuild.

It keeps the local QEMU/KVM workflow intentionally narrow, but makes that workflow more useful and more trustworthy by adding explicit instance controls and by tightening the app-level JSON error contract.

This release adds:

- `disk_size`
- `hostname`
- `ssh_port`
- broader negative-path validation through the smoke suite
- stable error-code coverage across the main app command surface

This release does **not** add provisioning, snapshots, private networking, guest control, LabsBackery integration, MCP integration, or cloud worker behavior.

## What Changed

### Instance Config Controls

#### `disk_size`

`disk_size` is now a supported instance field in `yeast.yaml`.

It is:

- parsed and normalized during config loading
- validated before runtime start
- passed into the runtime disk plan
- used during overlay disk creation

Existing overlay disks are not resized automatically. `yeast up` only applies `disk_size` when creating a new overlay disk.

#### `hostname`

`hostname` is now a supported instance field in `yeast.yaml`.

It is:

- validated with the same safe-name rules used for instance names
- defaulted to the instance name when omitted
- passed through cloud-init user-data
- passed through cloud-init meta-data

This means the guest hostname can now be controlled explicitly and verified from inside the VM.

#### `ssh_port`

`ssh_port` is now a supported instance field in `yeast.yaml`.

It is:

- validated before runtime start
- preserved through normal restart flows
- shown in human `status`
- shown in JSON `status`
- checked for duplicate requested values across instances

Duplicate requested ports now fail early with `invalid_argument`.

## Error Contract Cleanup

The main command surface now returns stable app error codes more consistently.

Covered commands:

- `yeast up`
- `yeast status`
- `yeast ssh`
- `yeast pull`
- `yeast init`
- `yeast down`
- `yeast destroy`

Important result:

- unsupported image errors now preserve `invalid_argument` in `--json`
- uninitialized-project flows return `failed_precondition`
- repeated init returns `conflict`
- invalid config cases return `invalid_argument`
- corrupted or mismatched state/metadata surfaces return `internal`

## Verification

### Automated

The following checks passed before release packaging:

```bash
go test ./... -count=1
go build ./...
git diff --check
bash -n scripts/manual-smoke.sh
```

### Real Host Validation

The real Linux/KVM smoke workflow passed with:

```bash
TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

That smoke run covered:

- `doctor`
- `init`
- `pull`
- `up`
- `status`
- `status --json`
- direct SSH checks
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

## Example v0.2.0 Config

```yaml
version: 1
instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    ssh_port: 2205
    user: yeast
    sudo: none
```

## Not Included

- provisioning packages/files/shell workflows
- snapshots and restore
- private VM-to-VM networking
- guest `exec`, `copy`, `logs`, or `inspect`
- template workflows
- LabsBackery runtime contract
- Yeast MCP contract
- Twarga Cloud worker mode
