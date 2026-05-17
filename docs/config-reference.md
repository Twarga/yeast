# Yeast Config Reference

Yeast reads project configuration from `yeast.yaml`.

## Minimal Config

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
```

`memory`, `cpus`, `user`, and `sudo` have defaults.

## Full v0.1 Example

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
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
| `networks` | no | Reserved for future networking milestones. Not active in v0.1. |
| `provision` | no | Reserved for future provisioning milestones. Not active in v0.1. |

## Instance Fields

| Field | Required | Default | Description |
|---|---:|---|---|
| `name` | yes | none | Instance name. Must be path-safe. |
| `image` | yes | none | Trusted image name, such as `ubuntu-24.04`. |
| `memory` | no | `512` | Memory in MiB. Must be at least `128`. |
| `cpus` | no | `1` | Number of virtual CPUs. Must be at least `1`. |
| `disk_size` | no | empty | Optional overlay disk size, such as `20G`. |
| `user` | no | `yeast` | Linux user created by cloud-init. |
| `sudo` | no | `none` | Sudo policy: `none`, `password`, or `nopasswd`. |
| `env` | no | empty | Environment values rendered into the guest profile script. |
| `user_data` | no | empty | Raw cloud-init user-data override. |
| `networks` | no | empty | Reserved for future networking milestones. |
| `provision` | no | empty | Reserved for future provisioning milestones. |

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
