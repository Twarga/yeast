# Yeast Config Reference

Yeast reads project configuration from `yeast.yaml`.

## Minimal Config

```yaml
version: 1
instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
```

`memory`, `cpus`, `user`, and `sudo` have defaults.

## Full Example

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    ssh_port: 2205
    user: yeast
    sudo: none
    env:
      APP_ENV: development
      LOG_LEVEL: debug
```

## Top-Level Fields

| Field | Required | Description |
|---|---:|---|
| `version` | yes | Config schema version. Must be `1`. |
| `instances` | yes | List of VMs in the project. Must contain at least one instance. |
| `networks` | no | One project-level private lab network for current v0.5+ networking. |
| `provision` | no | Provisioning config shared across instances. Runs before instance-level provisioning during `yeast up` and `yeast provision`. |

## Instance Fields

| Field | Required | Default | Description |
|---|---:|---|---|
| `name` | yes | none | Instance name. Must be path-safe. |
| `hostname` | no | instance `name` | Guest hostname rendered into cloud-init. |
| `image` | yes | none | Trusted image name, such as `ubuntu-24.04`. |
| `memory` | no | `512` | Memory in MiB. Must be at least `128`. |
| `cpus` | no | `1` | Number of virtual CPUs. Must be at least `1`. |
| `disk_size` | no | empty | Optional overlay disk size, such as `20G`. |
| `ssh_port` | no | auto from `2222` | Optional host SSH port forwarding override. |
| `user` | no | `yeast` | Linux user created by cloud-init. |
| `sudo` | no | `none` | Sudo policy: `none`, `password`, or `nopasswd`. |
| `env` | no | empty | Environment values rendered into the guest profile script. |
| `user_data` | no | empty | Raw cloud-init user-data override. |
| `networks` | no | empty | Per-instance attachment to the current project lab network. |
| `provision` | no | empty | Instance-specific provisioning config. Runs after top-level provisioning during `yeast up` and `yeast provision`. |

## Supported Images

Current built-in trusted images:

- `ubuntu-22.04`
- `ubuntu-24.04`

List them from the CLI:

```bash
yeast pull --list
```

## Instance Names

Instance names are used in runtime paths and state. Keep them simple.

Recommended pattern:

- lowercase letters
- numbers
- dashes
- underscores

Examples:

- `web`
- `db`
- `ubuntu_dev`
- `target-01`

Avoid spaces, slashes, shell syntax, and names that need quoting.

## Hostname

`hostname` controls the guest hostname Yeast writes into cloud-init `user-data` and `meta-data`.

If you omit it, Yeast uses the instance `name`.

Example:

```yaml
name: web
hostname: web-lab
```

This changes the hostname inside the VM without changing the Yeast instance identity, runtime paths, or command target name.

## Disk Size

`disk_size` controls the virtual size passed to `qemu-img` when Yeast creates a new instance overlay disk.

Supported formats use whole numbers with optional `K`, `M`, `G`, `T`, or `P` suffixes. A trailing `B` and spaces are accepted and normalized, so `20GB` and `20 gb` become `20G`.

If the instance disk already exists, Yeast keeps it and does not resize it during `up`.

Examples:

```yaml
disk_size: 20G
disk_size: 25600M
disk_size: 10737418240
```

## SSH Port

`ssh_port` lets you request a specific host port for SSH forwarding instead of using Yeast's automatic allocation starting at `2222`.

If you omit it, Yeast keeps the current behavior and picks the next available port.

Example:

```yaml
name: web
ssh_port: 2205
```

Rules:

- must be between `1` and `65535`
- must not collide with another requested or already-running Yeast instance port in the same project run
- if tracked state already uses a different SSH port for the same instance, Yeast fails instead of silently switching it

## Provisioning Schema

Yeast now accepts provisioning config in the schema. During `v0.3.0`, this config becomes the contract for post-boot provisioning.

Provisioning can appear:

- once at the top level for project-wide defaults
- per instance for instance-specific steps

Current schema:

```yaml
provision:
  packages:
    - caddy
    - curl
  files:
    - source: ./site
      destination: /srv/site
      permissions: "0644"
  shell:
    - systemctl enable --now caddy
```

### `packages`

- list of package names
- entries cannot be empty
- entries cannot contain newlines

### `files`

Each file item requires:

- `source`
- `destination`

Optional:

- `permissions`

Rules:

- `source` cannot be empty
- `destination` cannot be empty
- `source` and `destination` cannot contain newlines
- `permissions`, when set, must be an octal mode like `644` or `0644`

### `shell`

- list of shell commands
- entries cannot be empty after trimming whitespace

Provisioning behavior in `v0.3`:

- top-level steps run before instance-level steps
- `packages`, `files`, and `shell` run post-boot over SSH
- cloud-init remains responsible for user, SSH key, hostname, sudo, and environment bootstrap

Merge order:

```text
project packages -> instance packages
project files    -> instance files
project shell    -> instance shell
```

`yeast up` runs the merged post-boot provisioning plan automatically after SSH readiness.

`yeast provision` reruns the same merged post-boot provisioning plan against an existing reachable VM. It does not recreate disks, regenerate cloud-init, or reboot the VM unless a user-authored shell command does that.

Idempotency expectations:

- package installation relies on the guest package manager's normal idempotency
- file provisioning overwrites destination files
- shell commands always run, so write them to be safe on reruns

File source paths are resolved relative to the project root when they are not absolute.

## Templates

Templates are not a separate config schema. They are starter project directories copied by `yeast init`.

List built-in templates:

```bash
yeast init --list-templates
```

Initialize from a built-in template:

```bash
yeast init --template caddy-single-vm
```

Initialize from a local template directory:

```bash
yeast init --template ../my-template
```

Current local template shape:

```text
template.yaml
yeast.yaml
optional-project-files/
```

`template.yaml` describes the starter:

```yaml
name: my-template
title: My Template
description: Reusable Yeast starter.
category: app
version: "1"
files:
  - yeast.yaml
  - README.md
```

The listed files are copied into the new project. After initialization, generated files are normal editable project files.

## Sudo Modes

`sudo: none`

The bootstrap user is created without sudo access.

`sudo: password`

The bootstrap user gets sudo access that requires a password.

`sudo: nopasswd`

The bootstrap user gets passwordless sudo access.

Use `nopasswd` for disposable local labs only.

## Raw `user_data`

`user_data` replaces Yeast-generated cloud-init.

That means Yeast will not automatically merge:

- the configured user
- the SSH authorized key
- the sudo policy
- environment values

Only use raw `user_data` if you are comfortable writing complete cloud-init config yourself.

## Validation Rules

Yeast rejects:

- unsupported config versions
- empty instance lists
- duplicate instance names
- unsafe instance names
- missing images
- memory below `128`
- CPU count below `1`
- invalid disk sizes
- invalid Linux usernames
- invalid sudo values
- env keys that are unsafe for shell export
- env values containing newlines
