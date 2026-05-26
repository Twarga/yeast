# Yeast JSON Contract

Status: Draft for v1.0.0 stabilization

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

## Stable Command Data Shapes

The following fields are the v1 candidate `yeast.v1` integration surface for LabsBakery, Yeast MCP, scripts, and future automation.

### `doctor --json`

```json
{
  "checks": [
    {
      "name": "qemu-system-x86_64",
      "status": "ok",
      "details": "/usr/bin/qemu-system-x86_64"
    }
  ],
  "blockers": 0,
  "warnings": 0
}
```

Fields:

- `checks`: ordered host checks.
- `checks[].name`: check name.
- `checks[].status`: check status.
- `checks[].details`: human-readable check detail.
- `blockers`: number of blocking failures.
- `warnings`: number of warnings.

### `init --json`

```json
{
  "project_root": "/home/user/project",
  "config_path": "/home/user/project/yeast.yaml",
  "metadata_path": "/home/user/project/.yeast/project.json",
  "project_id": "proj_...",
  "template": "ubuntu-basic",
  "created": true
}
```

Fields:

- `project_root`: initialized project directory.
- `config_path`: generated `yeast.yaml` path.
- `metadata_path`: generated `.yeast/project.json` path.
- `project_id`: stable project id.
- `template`: template name or path when initialized from a template.
- `created`: whether project files were created.

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

Fields:

- `templates`: available built-in templates.
- `templates[].name`: template id.
- `templates[].title`: display title.
- `templates[].description`: short description.
- `templates[].category`: template category.
- `templates[].version`: template version string.
- `templates[].source`: template source, currently `builtin` for built-ins.
- `templates[].path`: template source path.

### `pull --list --json`

```json
{
  "list": true,
  "images": ["ubuntu-22.04", "ubuntu-24.04"]
}
```

Fields:

- `list`: true for image-list responses.
- `images`: supported trusted image names.

### `pull <image> --json`

```json
{
  "image_name": "ubuntu-24.04",
  "image_path": "/home/user/.yeast/cache/images/ubuntu-24.04/image.qcow2",
  "manifest_url": "https://...",
  "sha256": "..."
}
```

Fields:

- `image_name`: trusted image name.
- `image_path`: local cached image path.
- `manifest_url`: upstream manifest URL when available.
- `sha256`: expected image checksum when available.

### `up --json`

```json
{
  "project_id": "proj_...",
  "instances": [
    {
      "name": "web",
      "status": "running",
      "ssh_address": "127.0.0.1:2222",
      "ssh_port": 2222
    }
  ]
}
```

Fields:

- `project_id`: project id.
- `instances`: startup results.
- `instances[].name`: instance name.
- `instances[].status`: resulting status.
- `instances[].ssh_address`: host SSH address when available.
- `instances[].ssh_port`: host SSH port when available.

Use `yeast up --json --events` for progress events before the final result.

### `provision [instance] --json`

```json
{
  "project_id": "proj_...",
  "instance": {
    "name": "web",
    "provisioning_status": "provisioned",
    "ssh_address": "127.0.0.1:2222",
    "ssh_port": 2222,
    "provision_log_path": "/home/user/.yeast/projects/proj_.../instances/web/provision.log",
    "last_error": ""
  }
}
```

Fields:

- `project_id`: project id.
- `instance.name`: instance name.
- `instance.provisioning_status`: provisioning status.
- `instance.ssh_address`: host SSH address when available.
- `instance.ssh_port`: host SSH port when available.
- `instance.provision_log_path`: provisioning log path when available.
- `instance.last_error`: last provisioning error when available.

Use `yeast provision --json --events` for progress events before the final result.

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
      "user": "yeast",
      "lab_ip": "10.10.10.20",
      "runtime_dir": "/home/user/.yeast/projects/proj_.../instances/web",
      "provision_log_path": "/home/user/.yeast/projects/proj_.../instances/web/provision.log",
      "provisioning_status": "provisioned",
      "last_error": ""
    }
  ]
}
```

Fields:

- `project_id`: project id.
- `instances`: status records.
- `instances[].name`: instance name.
- `instances[].status`: tracked status.
- `instances[].pid`: tracked runtime PID.
- `instances[].management_ip`: host management IP when available.
- `instances[].ssh_port`: host SSH port when available.
- `instances[].user`: configured guest user when available.
- `instances[].lab_ip`: private lab IP when available.
- `instances[].runtime_dir`: instance runtime directory when available.
- `instances[].provision_log_path`: provisioning log path when available.
- `instances[].provisioning_status`: provisioning status when available.
- `instances[].last_error`: last tracked error when available.

### `inspect <instance> --json`

```json
{
  "project_id": "proj_...",
  "instance": {
    "name": "web",
    "status": "running",
    "ssh_port": 2222,
    "user": "yeast",
    "lab_ip": "10.10.10.20",
    "runtime_dir": "...",
    "provision_log_path": "...",
    "provisioning_status": "provisioned"
  },
  "snapshot_names": ["clean"],
  "snapshot_count": 1
}
```

Fields:

- `project_id`: project id.
- `instance`: one `status` instance record.
- `snapshot_names`: snapshot names for the instance.
- `snapshot_count`: number of snapshots for the instance.

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

Fields:

- `project_id`: project id.
- `instance`: target instance.
- `run.command`: command string.
- `run.exit_code`: guest command exit code.
- `run.stdout`: guest stdout.
- `run.stderr`: guest stderr.
- `run.started_at`: start timestamp.
- `run.finished_at`: finish timestamp.
- `run.duration`: duration in nanoseconds.
- `run.timed_out`: whether Yeast timed out the command.

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

Fields:

- `project_id`: project id.
- `instance`: target instance.
- `direction`: `to_guest` or `from_guest`.
- `source`: resolved source path.
- `destination`: resolved destination path.
- `started_at`: transfer start timestamp.
- `finished_at`: transfer finish timestamp.
- `duration`: duration in nanoseconds.

### `logs <instance> --json`

```json
{
  "project_id": "proj_...",
  "instance": "web",
  "log_path": "/home/user/.yeast/projects/proj_.../instances/web/vm.log",
  "content": "..."
}
```

Fields:

- `project_id`: project id.
- `instance`: target instance.
- `log_path`: host path to the VM runtime log.
- `content`: log content returned by Yeast.

### `snapshot <instance> <name> --json`

```json
{
  "project_id": "proj_...",
  "instance": "web",
  "snapshot": {
    "name": "clean",
    "created_at": "2026-05-26T12:00:00Z",
    "description": "Clean baseline",
    "disk_path": "...",
    "source_disk_size": "20G"
  }
}
```

Fields:

- `project_id`: project id.
- `instance`: target instance.
- `snapshot.name`: snapshot name.
- `snapshot.created_at`: creation timestamp.
- `snapshot.description`: optional description.
- `snapshot.disk_path`: snapshot disk path.
- `snapshot.source_disk_size`: source disk size when known.

### `restore <instance> <name> --json`

```json
{
  "project_id": "proj_...",
  "instance": "web",
  "snapshot": {
    "name": "clean",
    "disk_path": "..."
  }
}
```

Fields:

- `project_id`: project id.
- `instance`: target instance.
- `snapshot`: restored snapshot metadata.

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

Fields:

- `project_id`: project id.
- `instance`: target instance.
- `snapshots`: snapshot metadata records.

### `delete-snapshot <instance> <name> --json`

```json
{
  "project_id": "proj_...",
  "instance": "web",
  "snapshot": "clean"
}
```

Fields:

- `project_id`: project id.
- `instance`: target instance.
- `snapshot`: deleted snapshot name.

### `down --json`

```json
{
  "project_id": "proj_...",
  "instances": [
    {
      "name": "web",
      "status": "stopped"
    }
  ]
}
```

Fields:

- `project_id`: project id.
- `instances[].name`: instance name.
- `instances[].status`: resulting status.

Use `yeast down --json --events` for progress events before the final result.

### `destroy --json`

```json
{
  "project_id": "proj_...",
  "instances": [
    {
      "name": "web",
      "status": "destroyed"
    }
  ]
}
```

Fields:

- `project_id`: project id.
- `instances[].name`: instance name.
- `instances[].status`: resulting status.

Use `yeast destroy --json --events` for progress events before the final result.

### `version --json`

```json
"v1.0.0"
```

The `data` field is a JSON string containing the Yeast version.

### Commands Without A JSON Data Contract

- `yeast ssh` is interactive and should not be used as a JSON workflow.
- `yeast docs` does not support `--json`.

## Event Envelope

Lifecycle events use JSON Lines. Each event is one JSON object followed by a newline.

Use `--json --events` to enable event streaming. `--events` is intentionally tied to JSON output so event lines are not mixed with human terminal rendering.

Commands with event streaming:

- `yeast up --json --events`
- `yeast provision [instance] --json --events`
- `yeast restore <instance> <name> --json --events`
- `yeast down --json --events`
- `yeast destroy --json --events`

Event output uses this envelope:

```json
{
  "schema_version": "yeast.v1",
  "type": "event",
  "name": "ssh.ready",
  "command": "up",
  "project_id": "proj_...",
  "instance": "web",
  "message": "SSH is ready",
  "time": "2026-05-26T12:00:00Z",
  "data": {
    "ssh_port": 2222
  }
}
```

Fields:

- `schema_version`: JSON contract version.
- `type`: always `event`.
- `name`: stable machine-readable event name.
- `command`: command that emitted the event.
- `project_id`: project id when known.
- `instance`: instance name when the event belongs to one instance.
- `message`: optional human-readable event summary.
- `time`: event timestamp.
- `data`: optional event-specific structured data.

## Stable Event Names

Stable lifecycle event names:

- `project.loaded`
- `config.validated`
- `image.ready`
- `disk.ready`
- `cloud_init.generated`
- `vm.starting`
- `ssh.waiting`
- `ssh.ready`
- `provision.started`
- `provision.finished`
- `snapshot.created`
- `restore.started`
- `restore.finished`
- `instance.ready`
- `instance.stopped`
- `instance.destroyed`
- `workflow.completed`
- `workflow.failed`

These names are intentionally generic. LabsBakery can map them to UI progress without learning QEMU internals.
