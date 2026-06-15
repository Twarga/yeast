# Commands

Use `yeast <command> --help` for exact CLI help.

## Global Flags

These flags are available on Yeast commands:

| Flag | Purpose | Docs Home |
|---|---|---|
| `--json` | Print machine-readable JSON output | [JSON Output](json-output.md) |
| `--events` | Stream lifecycle events as JSON Lines | [Events](events.md) |
| `--quiet`, `-q` | Suppress progress output where supported | this page |
| `--help`, `-h` | Show command help | this page |

`--events` is designed for automation and is normally used with JSON-capable workflows.

## Project And Host

| Command | Purpose |
|---|---|
| `yeast doctor` | Check whether the host is ready for Yeast |
| `yeast init` | Initialize a Yeast project |
| `yeast version` | Print Yeast version |
| `yeast docs` | Render terminal docs |
| `yeast completion` | Generate shell completion scripts |

Useful flags:

| Command | Flags |
|---|---|
| `yeast init` | `--list-templates`, `--template <name-or-path>` |
| `yeast docs` | `--list` |
| `yeast completion` | `bash`, `zsh`, `fish`, `powershell` |

Terminal docs are intentionally short offline help. Current topics are `quickstart`, `installation`, `config`, `troubleshooting`, and `release-smoke`.

`yeast docs` does not support `--json`.

## Images

| Command | Purpose |
|---|---|
| `yeast pull [image]` | List, search, or download trusted base images |
| `yeast images clean [image]` | Remove cached VM images |

Useful flags:

| Command | Flags |
|---|---|
| `yeast pull` | `--list`, `--cached` |
| `yeast images clean` | `--all`, `--dry-run` |

See [Images](images.md) for the supported image list.

## Lifecycle

| Command | Purpose |
|---|---|
| `yeast up` | Start the VMs described by the current project |
| `yeast down` | Stop running VMs in the current project |
| `yeast destroy` | Remove tracked VM runtime files for the current project |

Useful flags:

| Command | Flags |
|---|---|
| `yeast up` | `--no-provision`, `--reprovision`, `--sequential`, `--profile` |

## Guest Control

| Command | Purpose |
|---|---|
| `yeast status` | Show tracked VM status for the current project |
| `yeast inspect <instance>` | Show detailed state for one instance |
| `yeast logs <instance>` | Read the VM runtime log for one instance |
| `yeast ssh [instance]` | Open an SSH session to a running instance |
| `yeast exec [instance] -- <command...>` | Run a command inside a running instance over SSH |
| `yeast copy <instance> [--to-guest\|--from-guest] <source> <destination>` | Copy a file between host and guest |

Useful flags:

| Command | Flags |
|---|---|
| `yeast logs` | `--tail <lines>` |
| `yeast ssh` | `--verbose` |
| `yeast exec` | `--timeout <duration>` |
| `yeast copy` | `--to-guest`, `--from-guest`, `--timeout <duration>` |

## Provisioning And Snapshots

| Command | Purpose |
|---|---|
| `yeast provision [instance]` | Rerun provisioning for a reachable running instance |
| `yeast snapshot <instance> <name>` | Create a stopped-VM snapshot for one instance |
| `yeast snapshots <instance>` | List snapshots for one instance |
| `yeast restore <instance> <name>` | Restore a stopped instance disk from a named snapshot |
| `yeast delete-snapshot <instance> <name>` | Delete a named snapshot for one instance |

Useful flags:

| Command | Flags |
|---|---|
| `yeast snapshot` | `--description <text>` |

## Utilities

| Command | Purpose |
|---|---|
| `yeast update` | Update Yeast |

Useful flags:

| Command | Flags |
|---|---|
| `yeast update` | `--check`, `--force`, `--version <tag>` |
