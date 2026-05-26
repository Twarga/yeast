# Yeast v1 Contract Audit

Status: Draft for v1.0.0 stabilization

This audit defines the public Yeast surfaces that should be frozen before `v1.0.0`.

The goal is not to add new product scope. The goal is to make the existing engine reliable enough that users, LabsBakery, scripts, and future Yeast MCP integrations can depend on it.

## Contract Rule

For v1.0, Yeast should preserve:

- documented command names and argument shapes
- documented config fields and validation meanings
- documented JSON envelope fields
- documented command-specific JSON fields
- documented error codes
- documented event names
- documented example/template behavior

Yeast may add optional fields later. Yeast should not remove or change the meaning of stable v1 fields without a new schema version, migration path, or clearly documented breaking release.

Human terminal formatting is not a stable contract. Automation should use `--json`.

## Public CLI Commands

These commands are in the v1 surface and should remain available through the v1 line.

| Command | Stable syntax | Notes |
|---|---|---|
| `yeast doctor` | `yeast doctor` | Host readiness check. |
| `yeast init` | `yeast init [--template <name-or-path>] [--list-templates]` | Initializes project metadata and config, or lists templates. |
| `yeast pull` | `yeast pull [image] [--list]` | Lists or downloads trusted base images. |
| `yeast up` | `yeast up` | Starts configured instances and runs post-boot provisioning. |
| `yeast provision` | `yeast provision [instance]` | Reruns provisioning against a reachable running instance. |
| `yeast snapshot` | `yeast snapshot <instance> <name> [--description <text>]` | Creates stopped-VM snapshot for one instance. |
| `yeast restore` | `yeast restore <instance> <name>` | Restores stopped instance disk from a snapshot. |
| `yeast snapshots` | `yeast snapshots <instance>` | Lists snapshots for one instance. |
| `yeast delete-snapshot` | `yeast delete-snapshot <instance> <name>` | Deletes one snapshot. |
| `yeast status` | `yeast status` | Shows project status. |
| `yeast exec` | `yeast exec [instance] -- <command...> [--timeout <duration>]` | Runs a command over SSH. If instance is omitted, exactly one running instance is required. |
| `yeast copy` | `yeast copy <instance> (--to-guest\|--from-guest) <source> <destination> [--timeout <duration>]` | Copies one file over SSH. |
| `yeast logs` | `yeast logs <instance> [--tail <lines>]` | Reads VM runtime log. |
| `yeast inspect` | `yeast inspect <instance>` | Shows detailed tracked state for one instance. |
| `yeast ssh` | `yeast ssh [instance]` | Opens an interactive SSH session. If instance is omitted, exactly one running instance is required. |
| `yeast down` | `yeast down` | Stops running project VMs. |
| `yeast destroy` | `yeast destroy` | Removes tracked project runtime files. |
| `yeast version` | `yeast version` | Prints version string. |
| `yeast docs` | `yeast docs [topic] [--list]` | Renders embedded terminal docs. |

The Cobra-generated `completion` and `help` commands exist, but Yeast-specific v1 docs should focus on the commands above.

## Global Flags

These global flags are part of the v1 surface:

| Flag | Meaning |
|---|---|
| `--json` | Render machine-readable JSON envelopes instead of human output where supported. |
| `--events` | With `--json`, stream lifecycle events as JSON Lines for supported long-running workflows. |
| `--help` / `-h` | Render command help. |

`--events` is intentionally useful only with `--json`. Current event-streaming workflows are `up`, `provision`, `restore`, `down`, and `destroy`.

## Stable Config Schema

The v1 config file is `yeast.yaml` with `version: 1`.

### Top-Level Fields

| Field | Type | Required | Default | Stable meaning |
|---|---|---:|---|---|
| `version` | integer | yes | none | Config schema version. Must be `1`. |
| `instances` | list | yes | none | VM instances in this project. Must contain at least one item. |
| `networks` | list | no | empty | Project-level private lab networks. Current v1 candidate supports at most one. |
| `provision` | object | no | empty | Project-level post-boot provisioning merged before instance provisioning. |

### Instance Fields

| Field | Type | Required | Default | Stable meaning |
|---|---|---:|---|---|
| `name` | string | yes | none | Yeast instance identity and command target. |
| `hostname` | string | no | `name` | Guest hostname rendered by cloud-init. |
| `image` | string | yes | none | Trusted image name. |
| `memory` | integer | no | `512` | Memory in MiB. |
| `cpus` | integer | no | `1` | Virtual CPU count. |
| `disk_size` | string | no | empty | Optional overlay disk size for newly created disks. Existing disks are not resized. |
| `ssh_port` | integer | no | auto from `2222` | Host SSH forwarding port override. |
| `user` | string | no | `yeast` | Guest user created by cloud-init. |
| `sudo` | string | no | `none` | One of `none`, `password`, or `nopasswd`. |
| `env` | map | no | empty | Environment values written to guest profile script. |
| `user_data` | string | no | empty | Raw cloud-init override. Replaces Yeast-generated user-data. |
| `networks` | list | no | empty | Instance attachment to the current project lab network. Current v1 candidate supports at most one attachment. |
| `provision` | object | no | empty | Instance-level provisioning merged after project-level provisioning. |

### Network Fields

| Field | Type | Required | Stable meaning |
|---|---|---:|---|
| `name` | string | yes | Project network identity referenced by instances. |
| `cidr` | string | yes | IPv4 CIDR for the private lab network. |

### Instance Network Fields

| Field | Type | Required | Stable meaning |
|---|---|---:|---|
| `name` | string | yes | Existing project network name. |
| `ipv4` | string | yes | Static guest IPv4 inside the network CIDR. |

### Provisioning Fields

Provisioning objects may appear at top level and per instance.

| Field | Type | Required | Stable meaning |
|---|---|---:|---|
| `packages` | list of strings | no | Package names installed over SSH post-boot. |
| `files` | list of file objects | no | Files copied over SSH post-boot. |
| `shell` | list of strings | no | Shell commands run over SSH post-boot. |

File provisioning object:

| Field | Type | Required | Stable meaning |
|---|---|---:|---|
| `source` | string | yes | Host path. Relative paths resolve from project root. |
| `destination` | string | yes | Guest path. |
| `permissions` | string | no | Optional octal mode, such as `0644`. |

## Stable JSON Envelope

All machine-readable command output should use `schema_version: "yeast.v1"`.

Success envelope:

```json
{
  "ok": true,
  "schema_version": "yeast.v1",
  "command": "status",
  "data": {}
}
```

Error envelope:

```json
{
  "ok": false,
  "schema_version": "yeast.v1",
  "error": {
    "code": "invalid_argument",
    "message": "..."
  }
}
```

Stable envelope fields:

- `ok`
- `schema_version`
- `command`
- `data`
- `error.code`
- `error.message`
- `error.details`

## Stable Error Codes

These error codes are part of the v1 candidate contract:

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

## Stable JSON Data Shapes

The following command data shapes are already documented in `docs/json-contract.md` and should be treated as the initial v1 stable automation surface:

- `status --json`
- `inspect <instance> --json`
- `exec <instance> --json -- <command>`
- `copy <instance> --json`
- `logs <instance> --json`
- `snapshots <instance> --json`
- `init --list-templates --json`

The remaining commands support JSON envelopes but need fuller command-specific field documentation before v1.0:

- `doctor --json`
- `init --json`
- `pull --json`
- `up --json`
- `provision --json`
- `snapshot --json`
- `restore --json`
- `delete-snapshot --json`
- `down --json`
- `destroy --json`
- `docs --json`

`yeast ssh` is interactive and should not be treated as an automation JSON workflow.

## Stable Event Contract

Event streams use JSON Lines with `schema_version: "yeast.v1"` and `type: "event"`.

Supported event-streaming commands:

- `yeast up --json --events`
- `yeast provision [instance] --json --events`
- `yeast restore <instance> <name> --json --events`
- `yeast down --json --events`
- `yeast destroy --json --events`

Stable event envelope fields:

- `schema_version`
- `type`
- `name`
- `command`
- `project_id`
- `instance`
- `message`
- `time`
- `data`

Stable event names:

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

## Stable Examples And Templates

The v1 candidate examples that should keep working:

- `examples/ubuntu-basic`
- `examples/caddy-single-vm`
- `examples/two-vm-lab`
- `examples/labsbackery-attacker-target-basic`

The v1 candidate built-in templates that should keep working:

- `ubuntu-basic`
- `caddy-single-vm`
- `two-vm-lab`

## Audit Findings

The current v0.9.0 surface is close enough to enter v1 stabilization, but it is not ready to tag as v1.0 yet.

### Good Enough To Freeze

- CLI command names are coherent and cover the intended v1 product loop.
- Config schema `version: 1` is small and understandable.
- Project-scoped state and runtime paths are established.
- JSON envelopes and error codes exist.
- Event envelope and event names exist.
- Examples cover lifecycle, provisioning, reset, networking, templates, and LabsBakery package convention.

### Needs Follow-Up Before v1.0

- `docs/json-contract.md` does not yet document every command-specific JSON data shape.
- `docs/config-reference.md` should add a fuller network section with validation rules and examples.
- `docs/command-reference.md` does not exist yet.
- Help rendering should be covered by a simple scripted or unit test before v1.0.
- The installer should verify the final installed version when installing a stable tag.
- `docs/tutorial-test.html` still contains old v0.1 wording and should be refreshed or removed from the public docs path before v1.0.
- GitHub Actions currently passes, but GitHub warns that Node.js 20 actions are deprecated. The workflow should be checked before v1.0.

### Explicitly Out Of Scope For v1.0

- Twarga Cloud
- remote workers
- daemon or web API
- LabsBakery web UI
- Yeast MCP server
- VirtualBox backend
- Windows/macOS host support
- remote template registry
- marketplace
- multi-user controls
- billing

## Next v1 Tasks

This audit unblocks:

- `V1.0-T2`: freeze command reference
- `V1.0-T3`: freeze config reference
- `V1.0-T4`: freeze JSON and event contract
- `V1.0-T5`: harden installer and upgrade path
- `V1.0-T6`: expand release smoke coverage
