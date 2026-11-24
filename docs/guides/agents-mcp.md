# Yeast for AI Agents

Yeast is agent-grade infrastructure: every command emits a frozen JSON
contract, a local daemon exposes HTTP + SSE + a WebSocket VNC proxy, and a
native MCP server lets agents drive VMs as tools — with a project-level
policy file gating what they may do.

## Machine output

Every command accepts `--json` (stable `yeast.v1` envelope) and `--events`
(JSON Lines progress stream). Schemas live in `schemas/yeast.v1/` and are
frozen — new fields only ever arrive as `yeast.v2` envelopes.

```bash
yeast up --json
yeast status --json | jq '.data.instances[].name'
```

## `yeast mcp` — Model Context Protocol

A native MCP server speaks JSON-RPC over **stdio** (default) or through the
daemon's `POST /mcp` endpoint (Streamable HTTP). It exposes 18 tools:

`vm_status` `vm_up` `vm_down` `vm_destroy` `vm_exec` `vm_copy` `vm_ssh`
`vm_snapshot` `vm_snapshots` `vm_restore` `vm_delete_snapshot` `vm_fork`
`vm_logs` `vm_inspect` `vm_screenshot` `vm_input` `project_status` `doctor`

```bash
# stdio — plug into any MCP client
yeast mcp

# or via the daemon
curl --unix-socket ~/.yeast/daemon.sock \
     -X POST http://localhost/mcp?project=<id> \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## `.yeast/agent-policy.yaml` — tool gating

A policy file in the project directory restricts which MCP tools run and
clamps dangerous options:

```yaml
deny:
  - vm_destroy
  - vm_down
limits:
  exec_timeout_seconds: 30
```

Denied calls return a policy error before touching the VM. Policy is
enforced **server-side** — a client cannot bypass it.

## `yeast daemon` — the local API

`yeast daemon` starts `yeastd`: a unix-socket (`0600`) HTTP API that shares
the same service layer as the CLI and reconciles with already-running VMs.
Optional `--listen 127.0.0.1:PORT --token …` adds a bearer-authenticated TCP
listener.

| Route | Purpose |
|---|---|
| `GET /v1/projects/{id}/status` | instance statuses |
| `POST /v1/projects/{id}/up` · `down` · `destroy` | lifecycle |
| `POST /v1/projects/{id}/exec` | remote command |
| `GET /v1/projects/{id}/events` | SSE stream (`yeast.v2` JSONL) |
| `GET /v1/projects/{id}/instances/{n}/vnc` | WebSocket→VNC proxy |
| `POST /mcp` | MCP over HTTP |
| `GET /v1/projects/{id}/export` | image export |

Full contract: `api/openapi.yaml`.

## Go SDK

`pkg/yeast` is a typed client over the daemon socket:

```go
c := yeast.New(yeast.DefaultSocket())
status, err := c.Status(ctx, projectID)
```

See `examples/sdk-basic`.

## Audit trail

Every mutating action (from CLI, API, or MCP) appends to
`.yeast/audit.jsonl`: actor (`cli`, `api`, `mcp:<client>`), action,
redacted arguments, duration, outcome. `yeast audit` renders it.

See also: [Security Model](../reference/security-model.md).
