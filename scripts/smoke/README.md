# Tutorial Smoke Harness

This harness runs the tutorial-style Yeast smoke flows from a fresh temp workspace and records one log per lab.

## Usage

Run all labs:

```bash
scripts/smoke/tutorials.sh all
```

Run one lab:

```bash
scripts/smoke/tutorials.sh lab07_templates
scripts/smoke/tutorials.sh repeat_lifecycle
```

Use a specific binary:

```bash
YEAST_BIN=./dist/yeast-linux-amd64 scripts/smoke/tutorials.sh all
```

Use a fixed workspace root instead of a temp dir:

```bash
SMOKE_ROOT=/tmp/yeast-smoke-run scripts/smoke/tutorials.sh lab08_json_events
```

## Behavior

- uses `YEAST_BIN` when set
- otherwise uses `./dist/yeast-linux-amd64` when present
- otherwise builds a local binary automatically
- creates a fresh temp workspace by default
- stores one log per lab under `logs/`
- attempts `yeast destroy` and `yeast clean` on lab failure
- prints a concise summary with runtime and log paths

## Labs

- `lab01_first_vm`
- `lab02_provisioning`
- `lab03_snapshot_restore`
- `lab04_multi_vm_network`
- `lab05_guest_control`
- `lab06_labsbackery_reset`
- `lab07_templates`
- `lab08_json_events`
- `cleanup_broken_yaml`
- `cleanup_orphan_qemu`
- `repeat_lifecycle`

## Notes

- The harness runs real QEMU/KVM flows for the VM labs, so it needs the same host prerequisites as the tutorials.
- A full `all` run takes several minutes.
- The workspace is intentionally left behind on success so logs and generated projects can be inspected.
- Yeast v1.1 does not support general `ports:` forwarding in `yeast.yaml`; use SSH tunnels or guest-side checks instead.
