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
- `timeout`
- `runtime_error`
- `provisioning_failed`
- `guest_error`
- `internal`
- `unknown`

Meanings:

- `invalid_argument`: user input, flags, config, or command arguments are invalid.
- `not_found`: a requested image, instance, template, snapshot, file, or log is missing.
- `conflict`: the requested action conflicts with existing state.
- `failed_precondition`: the project, host, instance, or state is not ready for the action.
- `timeout`: an operation exceeded its configured or default timeout.
- `runtime_error`: the VM runtime failed while preparing, starting, stopping, snapshotting, or destroying infrastructure.
- `provisioning_failed`: provisioning ran but failed.
- `guest_error`: an operation inside or against a guest failed, such as SSH exec or file copy.
- `internal`: Yeast hit an unexpected internal failure.
- `unknown`: a non-Yeast error reached the JSON renderer without classification.

## Compatibility Rule

For `yeast.v1`:

- existing envelope fields should not be removed
- existing field meanings should not change
- new optional fields may be added
- command-specific `data` objects should be documented before LabsBakery or MCP depend on them
