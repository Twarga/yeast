# JSON Output

Use `--json` for machine-readable output.

```bash
yeast status --json
```

JSON output uses the `yeast.v1` schema version.

Prefer JSON output for scripts and integrations. Do not scrape human terminal output.

## Success Envelope

Successful command output uses this shape:

```json
{
  "ok": true,
  "schema_version": "yeast.v1",
  "command": "status",
  "data": {}
}
```

`data` changes by command. For example, `status` returns a project id and instance list, while `version` returns the version string.

## Error Envelope

Errors use the same top-level schema and set `ok` to `false`:

```json
{
  "ok": false,
  "schema_version": "yeast.v1",
  "error": {
    "code": "failed_precondition",
    "message": "project metadata not found: /path/to/.yeast/project.json"
  }
}
```

Error details may include a `details` object when Yeast has structured context.

## Useful Commands

```bash
yeast version --json
yeast doctor --json
yeast status --json
yeast inspect web --json
yeast pull --list --json
yeast init --list-templates --json
```

## Rules For Scripts

- Check `ok` first.
- Check `schema_version` before assuming field names.
- Use `data` only after confirming the command you ran.
- Do not parse the human output tables.
- Use `--json --events` when you need progress during long lifecycle commands.
