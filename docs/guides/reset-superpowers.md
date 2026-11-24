# Reset Superpowers — Snapshots, Forks, Instant Restore

The headline Yeast feature: checkpoint a **running** VM (RAM + disk),
destroy the guest, and return to the checkpoint in about two seconds. Plus
copy-on-write forks for "same machine, N parallel copies".

## Three snapshot kinds

| Kind | When | Speed | Space |
|---|---|---|---|
| `overlay` | stopped VM | instant | tiny (qcow2 chain) |
| `live` | running VM | ~40–120 ms pause | RAM image + disk overlay |
| `copy` | legacy (pre-v1.4) | slow | full disk copy |

You don't choose — `yeast snapshot` picks automatically by whether the
instance is running.

## Live checkpoints

```bash
yeast up
yeast snapshot web golden          # RAM+disk checkpoint, VM keeps running
yeast exec web -- rm -rf /         # break the guest
yeast restore web golden           # back to golden, ~2s
```

Under the hood: QMP `stop` → `blockdev-snapshot-sync` → RAM migrate to file
→ `cont` (single `snapshot-save` fallback on new QEMU). Restore respawns
QEMU with `-incoming defer`, streams the memory image back via
`migrate-incoming`, resumes, and resyncs the guest clock over SSH
(`date`/`chronyc`).

`yeast doctor` reports whether your QEMU supports the fast path
(`mapped-ram` needs QEMU ≥ 8.2 roughly; the doctor row tells you what you
actually have).

## Whole-project checkpoints

```bash
yeast snapshot --all golden        # shared checkpoint across every instance
yeast restore --all golden         # all-or-nothing: preflight + safety snapshots
```

`restore --all` verifies every instance has the checkpoint **before**
touching anything, takes per-instance `__pre-restore` safety overlays, and
rolls back on failure.

## Forks

```bash
yeast fork web --count 3 --boot    # 3 COW siblings, fresh ports+MACs
yeast status                       # web-fork-1 … web-fork-3
yeast destroy --forks web          # remove just the fork group
```

Forks freeze the base image into a read-only layer; each fork gets a thin
overlay, auto-allocated SSH/service/VNC ports, its own MAC, and `ForkOf`
metadata. Partial creation rolls back cleanly.

## Cleanup

```bash
yeast snapshots web                # KIND + SIZE columns (real allocated bytes)
yeast delete-snapshot web golden   # chain-safe (commit/rebase the chain)
yeast clean --snapshots            # wipe every snapshot in the project
```

Deleting an overlay snapshot merges its chain safely — `qemu-img commit`
into the parent, or `rebase` the child if a child exists.
