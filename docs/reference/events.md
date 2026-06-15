# Events

Use `--json --events` to stream lifecycle events as JSON Lines.

```bash
yeast up --json --events
```

Events are useful for tools that need progress updates while a workflow runs.

`--events` requires `--json`.

## Event Shape

Each line is one JSON object:

```json
{
  "schema_version": "yeast.v1",
  "type": "event",
  "name": "ssh.ready",
  "command": "up",
  "project_id": "project_example",
  "instance": "web",
  "message": "SSH is ready",
  "time": "2026-06-15T12:00:00Z",
  "data": {}
}
```

Optional fields are omitted when they do not apply.

## Commands That Emit Events

Lifecycle events are used by long-running workflows:

| Command | Typical Use |
|---|---|
| `yeast up --json --events` | boot progress, image readiness, SSH readiness, provisioning |
| `yeast provision --json --events web` | provisioning progress |
| `yeast restore --json --events web baseline` | restore progress |
| `yeast down --json --events` | stop progress |
| `yeast destroy --json --events` | cleanup progress |

## Event Names

Current event names include:

| Event | Meaning |
|---|---|
| `project.loaded` | Yeast loaded project metadata |
| `config.validated` | `yeast.yaml` passed validation |
| `image.pulling` | Yeast is downloading an image |
| `image.ready` | Base image is ready |
| `disk.ready` | Instance disk is ready |
| `cloud_init.generated` | Cloud-init seed data was generated |
| `vm.starting` | QEMU startup is beginning |
| `ssh.waiting` | Yeast is waiting for SSH |
| `ssh.ready` | Guest SSH is reachable |
| `provision.started` | Provisioning started |
| `provision.finished` | Provisioning finished |
| `provision.skipped` | Provisioning was skipped |
| `snapshot.created` | Snapshot metadata/disk copy was created |
| `restore.started` | Restore started |
| `restore.finished` | Restore finished |
| `instance.ready` | One instance is ready |
| `instance.stopped` | One instance stopped |
| `instance.destroyed` | One instance was destroyed |
| `workflow.completed` | The command workflow completed |
| `workflow.failed` | The command workflow failed |

## Script Pattern

Read events line by line. Do not wait for the command to finish before processing output.

```bash
yeast up --json --events | while IFS= read -r line; do
  printf '%s\n' "$line"
done
```
