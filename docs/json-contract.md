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

## Command Data Field Style

Command-specific `data` objects use lower `snake_case` field names.

Do not depend on Go struct field names such as `ProjectID`, `SSHPort`, or `ProvisioningStatus`. The stable JSON fields are `project_id`, `ssh_port`, and `provisioning_status`.

## Stable Core Data Shapes

The following fields are the first stable `yeast.v1` integration surface for LabsBakery and automation.

### `status --json`

```json
{
  "project_id": "proj_...",
  "instances": [
    {
      "name": "web",
      "status": "running",
      "pid": 123,
      "management_ip": "127.0.0.1",
      "ssh_port": 2222,
      "lab_ip": "10.10.10.20",
      "runtime_dir": "/home/user/.yeast/projects/proj_.../instances/web",
      "provision_log_path": "/home/user/.yeast/projects/proj_.../instances/web/provision.log",
      "provisioning_status": "provisioned",
      "last_error": ""
    }
  ]
}
```

### `inspect <instance> --json`

```json
{
  "project_id": "proj_...",
  "instance": {
    "name": "web",
    "status": "running",
    "ssh_port": 2222,
    "lab_ip": "10.10.10.20",
    "runtime_dir": "...",
    "provision_log_path": "...",
    "provisioning_status": "provisioned"
  },
  "snapshot_names": ["clean"],
  "snapshot_count": 1
}
```

### `exec <instance> --json -- <command>`

```json
{
  "project_id": "proj_...",
  "instance": "web",
  "run": {
    "command": "whoami",
    "exit_code": 0,
    "stdout": "yeast\n",
    "stderr": "",
    "started_at": "2026-05-26T12:00:00Z",
    "finished_at": "2026-05-26T12:00:00.2Z",
    "duration": 200000000,
    "timed_out": false
  }
}
```

`duration` is encoded as nanoseconds because it is a Go duration.

### `copy --json`

```json
{
  "project_id": "proj_...",
  "instance": "web",
  "direction": "to_guest",
  "source": "./file.txt",
  "destination": "/home/yeast/file.txt",
  "started_at": "2026-05-26T12:00:00Z",
  "finished_at": "2026-05-26T12:00:01Z",
  "duration": 1000000000
}
```

### `logs <instance> --json`

```json
{
  "project_id": "proj_...",
  "instance": "web",
  "log_path": "/home/user/.yeast/projects/proj_.../instances/web/vm.log",
  "content": "..."
}
```

### `snapshots <instance> --json`

```json
{
  "project_id": "proj_...",
  "instance": "web",
  "snapshots": [
    {
      "name": "clean",
      "created_at": "2026-05-26T12:00:00Z",
      "disk_path": "...",
      "description": "Clean baseline"
    }
  ]
}
```

### `init --list-templates --json`

```json
{
  "templates": [
    {
      "name": "ubuntu-basic",
      "title": "Ubuntu Basic",
      "description": "Minimal Ubuntu VM starter for testing the Yeast lifecycle.",
      "category": "vm",
      "version": "1",
      "source": "builtin",
      "path": "builtin/ubuntu-basic"
    }
  ]
}
```
