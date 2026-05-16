# Yeast Architecture Overview

Yeast is designed as an engine with a CLI on top.

The CLI should stay thin. Real behavior belongs in internal application workflows and supporting packages.

## High-Level Flow

```text
yeast command
  -> cmd/yeast
  -> internal/app workflow
  -> config/project/state/images/cloud-init/runtime/output
```

Example for `yeast up`:

```text
CLI
  -> app.Up
     -> load project metadata
     -> resolve ~/.yeast paths
     -> lock state
     -> load yeast.yaml
     -> reconcile state
     -> verify cached image
     -> render cloud-init
     -> create seed ISO
     -> prepare qcow2 disk
     -> start QEMU/KVM
     -> wait for SSH
     -> save state
```

## Package Responsibilities

### `cmd/yeast`

Owns CLI command definitions.

Responsibilities:

- parse arguments and flags
- call `internal/app`
- choose human output or JSON output

It should not know QEMU details, state file internals, or cloud-init rendering details.

### `internal/app`

Owns user-facing workflows.

Current workflows:

- `Init`
- `Doctor`
- `Pull`
- `Up`
- `Status`
- `SSH`
- `Down`
- `Destroy`

The app layer coordinates other packages but should avoid low-level implementation details.

### `internal/project`

Owns project identity and paths.

Responsibilities:

- `.yeast/project.json`
- project IDs
- `~/.yeast` path resolution
- project runtime directories
- instance runtime directories

### `internal/config`

Owns `yeast.yaml`.

Responsibilities:

- load YAML
- validate schema
- apply defaults
- normalize values

### `internal/state`

Owns project runtime state.

Responsibilities:

- `state.json`
- `state.lock`
- atomic save
- stale lock handling
- process reconciliation

State records what Yeast believes exists. It is not the source of desired configuration.

### `internal/images`

Owns trusted base images.

Responsibilities:

- built-in trusted image manifest
- cache paths
- downloads
- SHA-256 verification

### `internal/provision/cloudinit`

Owns first-boot cloud-init bootstrap.

Responsibilities:

- SSH public key discovery
- `user-data`
- `meta-data`
- seed ISO creation through `genisoimage` or `mkisofs`

### `internal/runtime`

Defines the runtime boundary.

The app layer talks to this interface, not directly to QEMU.

### `internal/runtime/qemu`

Owns QEMU/KVM implementation.

Responsibilities:

- `qemu-img` disk preparation
- QEMU argv construction
- process start/stop/inspect/destroy
- VM log file wiring

### `internal/guest`

Owns host-to-guest control helpers.

Current responsibilities:

- TCP readiness checks
- SSH command argument construction
- running interactive SSH

### `internal/output`

Owns output rendering.

Responsibilities:

- human output
- JSON output
- success/error envelopes
- terminal styling for human mode

Important rule:

Human output can be pretty. JSON output must stay stable and unstyled.

## Storage Layout

Project folder:

```text
project/
  yeast.yaml
  .yeast/
    project.json
```

Yeast home:

```text
~/.yeast/
  cache/
    images/
      ubuntu-24.04/
        image.qcow2
  projects/
    <project-id>/
      state.json
      state.lock
      instances/
        web/
          disk.qcow2
          seed.iso
          user-data
          meta-data
          vm.log
```

## Output Model

Human output and JSON output share the same workflow results.

Human output:

- styled with Lip Gloss
- optimized for terminal readability
- allowed to change before v1

JSON output:

- plain JSON
- no ANSI styling
- intended for scripts, LabsBackery, and future MCP integrations
- should become stable before v1

## Future Architecture Direction

Planned future layers:

- lifecycle events
- live Bubble Tea progress UI
- provisioning workflows
- snapshots and reset
- private networking
- guest exec/copy/logs
- LabsBackery integration contract
- Yeast MCP integration
- remote worker mode for Twarga Cloud

These should build on the current app/runtime/output boundaries instead of bypassing them.
