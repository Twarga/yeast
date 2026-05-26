# LabsBakery Integration Contract

Status: Draft for Yeast v0.9.0

Yeast is the VM engine. LabsBakery is the lab product. This contract defines how LabsBakery should call Yeast without reimplementing VM runtime behavior or depending on human terminal output.

## Scope

This contract is for the first local LabsBakery integration:

- one LabsBakery lab maps to one Yeast project directory
- one LabsBakery lab session maps to one copied Yeast project/session directory
- LabsBakery calls the Yeast CLI with `--json`
- LabsBakery may use `--json --events` for progress during long-running actions
- LabsBakery stores product state, instructions, checks, users, and session metadata
- Yeast stores VM state, runtime files, disks, snapshots, provisioning state, and instance status

Out of scope:

- Yeast daemon/API mode
- hosted workers
- multi-user isolation
- billing, teams, courses, RBAC, or marketplace behavior
- LabsBakery-specific state inside Yeast

## Boundary

LabsBakery owns:

- lab catalog
- lab metadata
- lab sessions
- visual topology
- instructions
- checks and scoring
- learner progress
- terminal UX
- import/export

Yeast owns:

- images
- disks
- QEMU/KVM lifecycle
- cloud-init
- provisioning
- private networking
- snapshots and restore
- guest exec/copy/logs/inspect
- JSON envelopes
- lifecycle events

Rule:

```text
LabsBakery never parses Yeast human output.
LabsBakery never calls QEMU directly.
Yeast never stores LabsBakery learner/product state.
```

## Session Directory Model

LabsBakery should create a dedicated working directory for every lab session.

Example:

```text
~/.labsbakery/sessions/<session-id>/
  lab.json
  yeast.yaml
  files/
  .yeast/project.json
```

LabsBakery should run all Yeast commands with the session directory as the current working directory.

This keeps Yeast project identity, state, disks, logs, and snapshots isolated per session.

## Required Yeast Version

LabsBakery should require:

```text
yeast version >= v0.8.0
```

Required contract:

```text
schema_version = yeast.v1
```

LabsBakery should reject or warn on unknown schema versions until explicitly supported.

## Command Contract

### Create Session Project

For a built-in or local Yeast template:

```bash
yeast init --template <template-name-or-path> --json
```

Expected use:

- create `yeast.yaml`
- create `.yeast/project.json`
- copy template files into the session directory

LabsBakery should treat generated files as session-local files. It may modify them before first `up` when baking from a visual builder.

### Start Lab

```bash
yeast up --json --events
```

Expected use:

- create missing disks
- generate cloud-init
- start all configured instances
- wait for SSH readiness
- run provisioning
- emit progress events
- finish with a success envelope

LabsBakery should show progress from event lines and update final state from `status --json`.

### Read Status

```bash
yeast status --json
```

LabsBakery should use this as the source of runtime truth for:

- instance names
- running/stopped status
- management host and SSH port
- lab IPs
- provisioning status
- runtime/provision log paths
- last error when present

LabsBakery should not infer VM status from its own database without confirming through Yeast.

### Inspect Instance

```bash
yeast inspect <instance> --json
```

Expected use:

- terminal setup
- detailed VM view
- snapshot count/name display
- debug panels

### Stop Lab

```bash
yeast down --json
```

Expected use:

- stop all running instances in the session
- keep disks and snapshots

### Destroy Lab

```bash
yeast destroy --json
```

Expected use:

- remove session runtime files managed by Yeast
- leave LabsBakery session records to LabsBakery cleanup policy

### Create Baseline

```bash
yeast down --json
yeast snapshot <instance> clean --description "Clean baseline" --json
```

For multi-VM labs, LabsBakery v0.1 should snapshot each instance one at a time while all instances are stopped.

### Reset Lab

```bash
yeast down --json
yeast restore <instance> clean --json --events
yeast up --json --events
```

For multi-VM labs, LabsBakery v0.1 should restore each instance one at a time while all instances are stopped, then start the full lab again.

LabsBakery should make reset a destructive action in the UI because it discards learner changes after the baseline.

### Read Logs

```bash
yeast logs <instance> --json
```

Expected use:

- debug panels
- support views
- failure details after a failed start/provision

### Run Checks

```bash
yeast exec <instance> --json -- <command...>
```

Expected use:

- lab validation checks
- service health checks
- scoring helpers

LabsBakery should keep check definitions in LabsBakery scenario files, not in Yeast state.

## Terminal Contract

LabsBakery browser terminals should connect through SSH using data from `status --json` or `inspect --json`.

Required fields:

- `management_ip`
- `ssh_port`
- `user`
- `name`

Current default guest user:

```text
yeast
```

LabsBakery should use the `user` value from `status --json` or `inspect --json` when opening browser terminals.

LabsBakery should not run `yeast ssh` for browser terminals because `yeast ssh` is an interactive human command. Use an SSH library or backend PTY bridge pointed at the host/port from Yeast JSON.

## Event Handling

LabsBakery should use JSON Lines events for long-running operations:

```bash
yeast up --json --events
yeast provision --json --events
yeast restore <instance> clean --json --events
```

Required event fields:

- `schema_version`
- `type`
- `name`
- `command`
- `time`

Useful optional fields:

- `project_id`
- `instance`
- `message`
- `data`

LabsBakery should treat unknown event names as informational, not fatal. It should key UI progress off known names and retain the final command envelope as the source of success/failure.

## First UI Progress Mapping

Recommended event-to-UI mapping:

| Yeast event | LabsBakery UI copy |
|---|---|
| `project.loaded` | Loading lab session |
| `config.validated` | Validating lab config |
| `image.ready` | Preparing images |
| `disk.ready` | Preparing VM disks |
| `cloud_init.generated` | Preparing guest bootstrap |
| `vm.starting` | Starting machines |
| `ssh.waiting` | Waiting for guest access |
| `ssh.ready` | Guest access ready |
| `provision.started` | Running lab setup |
| `provision.finished` | Lab setup finished |
| `restore.started` | Restoring clean baseline |
| `restore.finished` | Baseline restored |
| `instance.ready` | Machine ready |
| `workflow.completed` | Lab ready |
| `workflow.failed` | Lab action failed |

## Error Handling

LabsBakery should branch on `error.code`, not `error.message`.

Recommended UI handling:

| Error code | LabsBakery behavior |
|---|---|
| `invalid_argument` | Mark lab/template as invalid and show maintainer-facing details |
| `not_found` | Show missing image/template/snapshot/instance action |
| `conflict` | Refresh status and show action conflict |
| `failed_precondition` | Show setup/state requirement |
| `timeout` | Offer retry and show logs |
| `runtime_error` | Show runtime failure and logs |
| `provisioning_failed` | Show provisioning failure and logs |
| `guest_error` | Show guest command/copy failure |
| `internal` | Show generic failure and collect logs |
| `unknown` | Show generic failure and collect logs |

## First Test Lab Target

The first LabsBakery integration lab should be:

```text
attacker VM
target VM
one private lab network
static lab IPs
simple target service
clean baseline snapshot
browser terminal for both VMs
reset button
destroy button
JSON status/debug panel
```

This target proves the full engine path without cloud hosting, auth, billing, marketplace, or advanced visual builder scope.

## Yeast Gaps To Track During v0.9

These are possible Yeast improvements found by the contract. Add them only when they are generic engine improvements:

- add project-level snapshot/reset helpers if per-instance reset becomes too repetitive
- add event coverage for `down` and `destroy` if LabsBakery needs progress for those actions
- document a first lab package/template folder convention
- make terminal connection fields explicit in the JSON contract

## Minimal Adapter Pseudocode

```text
create_session(template):
  mkdir session_dir
  run(session_dir, "yeast init --template <template> --json")

start_session(session):
  stream(session.dir, "yeast up --json --events")
  return run(session.dir, "yeast status --json")

reset_session(session):
  run(session.dir, "yeast down --json")
  for instance in session.instances:
    stream(session.dir, "yeast restore <instance> clean --json --events")
  stream(session.dir, "yeast up --json --events")
  return run(session.dir, "yeast status --json")

destroy_session(session):
  run(session.dir, "yeast destroy --json")
```
