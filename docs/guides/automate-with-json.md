# Automate With JSON

Use JSON when another program needs to read Yeast output.

Do not scrape human tables or styled terminal output.

## Basic Pattern

```bash
yeast status --json
yeast inspect web --json
```

JSON responses use the `yeast.v1` schema version.

## Check Success

Successful output starts like this:

```json
{
  "ok": true,
  "schema_version": "yeast.v1",
  "command": "status",
  "data": {}
}
```

Errors still use JSON:

```json
{
  "ok": false,
  "schema_version": "yeast.v1",
  "error": {
    "code": "failed_precondition",
    "message": "project metadata not found"
  }
}
```

## Event Streams

Use lifecycle events when a workflow takes time:

```bash
yeast up --json --events
```

Events are JSON Lines. Read them one line at a time:

```bash
yeast up --json --events | while IFS= read -r event; do
  printf '%s\n' "$event"
done
```

## Useful JSON Commands

```bash
yeast version --json
yeast doctor --json
yeast init --templates --json
yeast pull --list --json
yeast status --json
yeast inspect web --json
```

## Rules For Tools

- check `ok` before reading `data`
- check `schema_version`
- treat `data` as command-specific
- use events for progress
- do not use `--json` with `yeast docs`; terminal docs are human-only

See [JSON Output](../reference/json-output.md) and [Events](../reference/events.md) for the reference shapes.

## Stability Promise

`yeast.v1` output stays readable through all of v2. Tools that consume `yeast.v1` envelopes and events keep working — new capabilities arrive as additive `data` fields and new event names, never as breaking changes to existing keys. Machine-readable schemas for every command payload and the shared envelope/event live in [`schemas/yeast.v1/`](../../schemas/yeast.v1/) and are enforced by contract tests.
