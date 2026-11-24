# Reset Spike — measured live-snapshot timings

Measured on this host with `scripts/bench-reset.sh`. Re-run any time:

```sh
./scripts/bench-reset.sh [memory-mb]   # default 512
```

## Host under test

- QEMU emulator version **11.1.3** (qemu-11.1.3-1.fc45), qemu-img 11.1.3
- `/dev/kvm` present and writable → `accel=kvm`
- No guest OS booted (paused machine, scratch qcow2) — numbers measure QEMU's
  migration machinery + RAM write/read, not OS boot.

## QMP primitives available on this QEMU

| Command | Present |
|---|---|
| `snapshot-save` / `snapshot-load` | yes (job API, internal fallback path) |
| `blockdev-snapshot-sync` | yes (external disk snapshot) |
| `migrate` / `migrate-incoming` | yes |
| `mapped-ram` capability | yes (must be enabled on **both** ends) |
| `multifd` capability | yes |

## Measured numbers

| Operation | 512 MiB | 2048 MiB |
|---|---|---|
| `migrate file:` (RAM save, mapped-ram+multifd) | **41–44 ms** | **120 ms** |
| `migrate-incoming file:` + guest restored | **41 ms** | **46 ms** |

Scaled to a realistic 4 GiB desktop VM: expect ~0.2–0.3 s of RAM I/O — far
inside the ~2 s restore target. Guest-visible pause time equals the RAM save
window (`stop` → `cont`), so ~50–300 ms in practice.

## Spike conclusions adopted in Phase 3

1. **Method A (`file:` migration) is the primary path.** It is ~10–50× faster
   than required and simpler than `exec:cat` piping.
2. **`mapped-ram` + `multifd` are enabled when the host QEMU reports them**
   (`query-migrate-capabilities`), on both the save side and the
   `-incoming defer` restore side. QEMU < 8.2 lacks `mapped-ram`; the
   plain `file:`/`exec:` migration still works as the fallback path.
3. **`migrate-incoming` requires `-incoming defer`** on the spawned QEMU.
   The restored guest lands `paused` when the source was snapshotted under
   `stop` — `query-status` must accept `paused`, not just `running`.
4. **Capability mismatch is the main restore failure mode** ("received
   capability is on"): restore must set the same migration capabilities it
   saved with.
5. **`snapshot-save`/`snapshot-load` remain as the internal-snapshot
   fallback** for older QEMU builds (job API, all QEMU ≥ 6.0).
6. **Stopped overlay snapshots via `qemu-img create -b`** are effectively
   instant (<50 ms) and need no QMP — used for stopped-VM snapshots and
   `yeast fork`.

## Failure modes observed during the spike

- `query-status` polling for `"running"` hangs forever after restoring a
  stopped source — accept `paused`.
- Forgetting `migrate-set-capabilities` on the destination fails the whole
  incoming migration at the post-load hook.
