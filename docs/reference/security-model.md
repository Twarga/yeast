# Security Model

Yeast runs **rootless**: QEMU, virtiofsd, and the egress proxy all run as
your user. Nothing requires `sudo` for normal operation.

## Process sandboxing

Every QEMU process launches with:

```
-sandbox on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny
```

`resourcecontrol` is relaxed only when `display.gpu: virgl` needs it — the
only exception, and visible in `yeast doctor`.

## Sockets & files

| Artifact | Mode |
|---|---|
| QMP control socket | `0600` (full VM control — hardened explicitly) |
| Daemon unix socket | `0600` |
| `.yeast/audit.jsonl` | `0600`, append-only |

Sidecars (virtiofsd, egress proxy) run detached with pidfiles under the
instance runtime dir and are reaped on `down`/`destroy`.

## Egress modes

`egress.mode` in `yeast.yaml` controls what a VM can reach — enforced in
the QEMU netdev, not just policy:

| Mode | Effect |
|---|---|
| `open` | normal `user` networking (default) |
| `offline` | `restrict=on` — no internet, no host; SSH hostfwd + lab net still work |
| `allowlist` | `restrict=on` + `guestfwd` → a local HTTP/CONNECT proxy allowing only listed domains |

Every proxy ALLOW/DENY lands in `network.log` (`yeast logs <vm> --network`).
`yeast run --rm` defaults to **offline** — the safe choice for untrusted
workloads.

## Agent policy

`.yeast/agent-policy.yaml` gates MCP tools server-side — deny lists and
exec-timeout clamps apply regardless of what the client requests
([Agents & MCP](../guides/agents-mcp.md)).

## Daemon TCP access

The daemon's unix socket is primary. `--listen 127.0.0.1:PORT` is opt-in
and requires `--token` — every request must carry `Authorization: Bearer`.

## Audit trail

`.yeast/audit.jsonl` records every mutating operation with actor
(`cli`/`api`/`mcp:<client>`), redacted args, duration, and outcome.
`yeast audit` reads it. Sensitive fields (tokens, passwords) are redacted
before write.

## Honest host checks

`yeast doctor` reports what `yeast up` will actually hit — KVM device,
QEMU version + live-snapshot capability, virgl, virtiofsd — and never
pretends a missing piece is fine.
