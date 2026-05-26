# Yeast JSON Contract

Status: Draft for v0.8.0

Yeast JSON output is for tools, scripts, LabsBakery, Yeast MCP, and future automation.

Human terminal output is not a stable integration surface. Use `--json` for automation.

## Schema Version

All JSON envelopes include:

```json
{
  "schema_version": "yeast.v1"
}
```

The current schema version is:

```text
yeast.v1
```

Yeast should not make breaking changes inside `yeast.v1`. If a breaking change is required later, introduce a new schema version.

## Success Envelope

Successful command output uses this envelope:

```json
{
  "ok": true,
  "schema_version": "yeast.v1",
  "command": "status",
  "data": {}
}
```

Fields:

- `ok`: always `true` for success.
- `schema_version`: JSON contract version.
- `command`: command name that produced the response.
- `data`: command-specific response body.

## Error Envelope

Failed command output uses this envelope:

```json
{
  "ok": false,
  "schema_version": "yeast.v1",
  "error": {
    "code": "invalid_argument",
    "message": "instance web has invalid ssh_port 70000"
  }
}
```

Fields:

- `ok`: always `false` for errors.
- `schema_version`: JSON contract version.
- `error.code`: stable machine-readable error code.
- `error.message`: human-readable explanation.
- `error.details`: optional structured details.

## Current Error Codes

Current codes:

- `invalid_argument`
- `not_found`
- `conflict`
- `failed_precondition`
- `internal`
- `unknown`

More specific v0.8 codes may be added for timeouts, runtime failures, provisioning failures, and guest operation failures.

## Compatibility Rule

For `yeast.v1`:

- existing envelope fields should not be removed
- existing field meanings should not change
- new optional fields may be added
- command-specific `data` objects should be documented before LabsBakery or MCP depend on them
