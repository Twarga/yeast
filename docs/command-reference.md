# Yeast Command Reference

Status: Draft for v1.0.0 stabilization

This reference documents the Yeast CLI commands that are part of the v1 candidate surface.

Human output is designed for readability and may change visually. Automation should use `--json`. Long-running workflow progress should use `--json --events` where supported.

## Global Flags

| Flag | Description |
|---|---|
| `--json` | Output machine-readable JSON when the command supports it. |
| `--events` | Stream lifecycle events as JSON Lines. Must be used with `--json`. |
| `-h`, `--help` | Show help for the current command. |

Supported event-streaming commands:

- `up`
- `provision`
- `restore`
- `down`
- `destroy`

## `yeast doctor`

Purpose:

Check whether the host has the tools and permissions Yeast needs.

Syntax:

```bash
yeast doctor
```

Flags:

- global flags only

Human behavior:

- shows checks for QEMU, `qemu-img`, ISO builder, SSH, `/dev/kvm`, SSH public key, and cache directory
- separates blockers from warnings

JSON behavior:

```bash
yeast doctor --json
```

Returns `checks`, `blockers`, and `warnings`.

Example:

```bash
yeast doctor
```

Known limits:

- host checks do not prove a full VM boot will succeed
- distro-specific package advice belongs to installer docs, not this command

## `yeast init`

Purpose:

Create a Yeast project in the current directory, or initialize from a template.

Syntax:

```bash
yeast init
yeast init --template <name-or-path>
yeast init --list-templates
```

Flags:

| Flag | Description |
|---|---|
| `--template <name-or-path>` | Initialize from a built-in template or local template directory. |
| `--list-templates` | List available built-in templates. |

Human behavior:

- creates `yeast.yaml`
- creates `.yeast/project.json`
- refuses to overwrite an existing project config
- lists template metadata when `--list-templates` is set

JSON behavior:

```bash
yeast init --json
yeast init --template caddy-single-vm --json
yeast init --list-templates --json
```

Returns project paths and project id for project creation, or template summaries for template listing.

Example:

```bash
mkdir demo
cd demo
yeast init --template ubuntu-basic
```

Known limits:

- remote template registries are not supported
- template variables are not supported

## `yeast pull`

Purpose:

List or download trusted base images into the shared Yeast image cache.

Syntax:

```bash
yeast pull --list
yeast pull <image>
```

Flags:

| Flag | Description |
|---|---|
| `--list` | List supported trusted images. |

Human behavior:

- lists known images or downloads and verifies the requested image

JSON behavior:

```bash
yeast pull --list --json
yeast pull ubuntu-24.04 --json
```

Returns image list or image cache details.

Example:

```bash
yeast pull ubuntu-24.04
```

Known limits:

- only built-in trusted images are supported
- custom image registration is not a v1 feature

## `yeast up`

Purpose:

Start the instances described by the current `yeast.yaml`.

Syntax:

```bash
yeast up
```

Flags:

- global flags only

Human behavior:

- loads project metadata and config
- checks cached images
- creates disks and cloud-init seed files as needed
- starts QEMU/KVM processes
- waits for SSH readiness
- runs post-boot provisioning when configured
- saves runtime state

JSON behavior:

```bash
yeast up --json
yeast up --json --events
```

Returns project id and per-instance startup results. With `--events`, emits JSON Lines progress before the final JSON result.

Example:

```bash
yeast up
yeast status
```

Known limits:

- existing disks are not resized when `disk_size` changes
- provisioning shell commands are user-authored and may not be idempotent
- only the current private network model is supported

## `yeast provision`

Purpose:

Rerun the merged provisioning plan against a running reachable instance.

Syntax:

```bash
yeast provision [instance]
```

Flags:

- global flags only

Human behavior:

- selects the target instance
- copies configured files
- installs configured packages
- runs configured shell commands
- updates provisioning status and log path

JSON behavior:

```bash
yeast provision web --json
yeast provision web --json --events
```

Returns project id and provisioning result for the target instance.

Example:

```bash
yeast provision web
```

Known limits:

- requires a running reachable VM
- does not recreate disks, regenerate cloud-init, or reboot unless user shell steps do that

## `yeast snapshot`

Purpose:

Create a named disk snapshot for one stopped instance.

Syntax:

```bash
yeast snapshot <instance> <name> [--description <text>]
```

Flags:

| Flag | Description |
|---|---|
| `--description <text>` | Optional snapshot description. |

Human behavior:

- verifies the instance exists and is stopped
- creates a snapshot file
- records snapshot metadata in project state

JSON behavior:

```bash
yeast snapshot web clean --json
```

Returns project id, instance, and snapshot metadata.

Example:

```bash
yeast down
yeast snapshot web clean --description "Clean baseline"
```

Known limits:

- live snapshots are not supported
- project-wide atomic snapshots are not supported

## `yeast restore`

Purpose:

Restore a stopped instance disk from a named snapshot.

Syntax:

```bash
yeast restore <instance> <name>
```

Flags:

- global flags only

Human behavior:

- verifies the instance is stopped
- replaces the instance disk from the snapshot
- keeps snapshot metadata

JSON behavior:

```bash
yeast restore web clean --json
yeast restore web clean --json --events
```

Returns project id, instance, and restored snapshot metadata.

Example:

```bash
yeast down
yeast restore web clean
yeast up
```

Known limits:

- live restore is not supported
- project-wide atomic restore is not supported

## `yeast snapshots`

Purpose:

List snapshots for one instance.

Syntax:

```bash
yeast snapshots <instance>
```

Flags:

- global flags only

Human behavior:

- lists snapshot names, creation times, and descriptions

JSON behavior:

```bash
yeast snapshots web --json
```

Returns project id, instance, and snapshot metadata list.

Example:

```bash
yeast snapshots web
```

Known limits:

- lists snapshots for one instance at a time

## `yeast delete-snapshot`

Purpose:

Delete one named snapshot for one instance.

Syntax:

```bash
yeast delete-snapshot <instance> <name>
```

Flags:

- global flags only

Human behavior:

- removes snapshot file and metadata

JSON behavior:

```bash
yeast delete-snapshot web clean --json
```

Returns project id, instance, and deleted snapshot name.

Example:

```bash
yeast delete-snapshot web clean
```

Known limits:

- no bulk snapshot deletion command exists yet

## `yeast status`

Purpose:

Show tracked VM status for the current project.

Syntax:

```bash
yeast status
```

Flags:

- global flags only

Human behavior:

- shows instance name, status, SSH address, lab IP, and provisioning status when available
- reconciles tracked process state before rendering

JSON behavior:

```bash
yeast status --json
```

Returns project id and instance records.

Example:

```bash
yeast status
```

Known limits:

- status is project-scoped
- external changes inside the guest are not inspected

## `yeast exec`

Purpose:

Run a command inside a running instance over SSH.

Syntax:

```bash
yeast exec [instance] -- <command...> [--timeout <duration>]
```

Flags:

| Flag | Description |
|---|---|
| `--timeout <duration>` | Maximum time to wait for remote command completion. |

Human behavior:

- prints guest stdout and stderr
- exits with command failure when guest command fails

JSON behavior:

```bash
yeast exec web --json -- whoami
```

Returns command, exit code, stdout, stderr, timestamps, duration, and timeout status.

Example:

```bash
yeast exec web -- uname -a
```

Known limits:

- SSH-backed only
- interactive TTY command support is not the target of `exec`; use `yeast ssh`

## `yeast copy`

Purpose:

Copy one file between host and guest over SSH.

Syntax:

```bash
yeast copy <instance> --to-guest <source> <destination> [--timeout <duration>]
yeast copy <instance> --from-guest <source> <destination> [--timeout <duration>]
```

Flags:

| Flag | Description |
|---|---|
| `--to-guest` | Copy a local file to the guest. |
| `--from-guest` | Copy a guest file to the local machine. |
| `--timeout <duration>` | Maximum time to wait for transfer completion. |

Human behavior:

- copies in exactly one direction
- fails unless exactly one direction flag is set

JSON behavior:

```bash
yeast copy web --to-guest ./app.conf /home/yeast/app.conf --json
```

Returns project id, instance, direction, source, destination, timestamps, and duration.

Example:

```bash
yeast copy web --from-guest /var/log/cloud-init.log ./cloud-init.log
```

Known limits:

- recursive directory sync is not a v1 command
- large transfer progress is not reported yet

## `yeast logs`

Purpose:

Read Yeast's runtime VM log for one instance.

Syntax:

```bash
yeast logs <instance> [--tail <lines>]
```

Flags:

| Flag | Description |
|---|---|
| `--tail <lines>` | Return only the last N lines. |

Human behavior:

- prints VM runtime log content

JSON behavior:

```bash
yeast logs web --tail 100 --json
```

Returns project id, instance, log path, and content.

Example:

```bash
yeast logs web --tail 80
```

Known limits:

- does not stream logs live
- reads Yeast runtime log, not arbitrary guest service logs

## `yeast inspect`

Purpose:

Show detailed tracked state for one instance.

Syntax:

```bash
yeast inspect <instance>
```

Flags:

- global flags only

Human behavior:

- shows state fields for a single instance
- includes snapshot summary

JSON behavior:

```bash
yeast inspect web --json
```

Returns project id, instance status data, snapshot names, and snapshot count.

Example:

```bash
yeast inspect web
```

Known limits:

- reports tracked state, not a full guest inventory

## `yeast ssh`

Purpose:

Open an interactive SSH session to a running instance.

Syntax:

```bash
yeast ssh [instance]
```

Flags:

- global flags only

Human behavior:

- launches the host `ssh` command against the selected instance
- if no instance is provided, exactly one running instance is required

JSON behavior:

`yeast ssh` is interactive and should not be used as an automation JSON workflow.

Example:

```bash
yeast ssh web
```

Known limits:

- requires SSH readiness
- does not expose a browser terminal by itself

## `yeast down`

Purpose:

Stop running VMs in the current project without deleting disks.

Syntax:

```bash
yeast down
```

Flags:

- global flags only

Human behavior:

- stops tracked running instances
- saves stopped state

JSON behavior:

```bash
yeast down --json
yeast down --json --events
```

Returns project id and per-instance stop results.

Example:

```bash
yeast down
```

Known limits:

- project-scoped only
- does not delete disks or snapshots

## `yeast destroy`

Purpose:

Remove tracked VM runtime files for the current project.

Syntax:

```bash
yeast destroy
```

Flags:

- global flags only

Human behavior:

- removes tracked instance runtime directories
- clears project state entries
- keeps the shared image cache

JSON behavior:

```bash
yeast destroy --json
yeast destroy --json --events
```

Returns project id and per-instance destroy results.

Example:

```bash
yeast down
yeast destroy
```

Known limits:

- project metadata and `yeast.yaml` remain in the project directory
- shared cached images remain under `~/.yeast/cache/images`

## `yeast version`

Purpose:

Print the Yeast version.

Syntax:

```bash
yeast version
```

Flags:

- global flags only

Human behavior:

- prints a version string

JSON behavior:

```bash
yeast version --json
```

Returns the version string in the standard JSON success envelope.

Example:

```bash
yeast version
```

Known limits:

- development builds may report `0.0.0-dev`

## `yeast docs`

Purpose:

Render embedded Yeast docs in the terminal.

Syntax:

```bash
yeast docs [topic]
yeast docs --list
```

Flags:

| Flag | Description |
|---|---|
| `--list` | List available docs topics. |

Human behavior:

- renders markdown docs in the terminal
- lists embedded topics with `--list`

JSON behavior:

`yeast docs` does not support `--json`.

Example:

```bash
yeast docs quickstart
yeast docs --list
```

Known limits:

- embedded docs are a curated subset of repository docs

