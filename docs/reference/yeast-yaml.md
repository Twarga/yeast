# yeast.yaml Reference

`yeast.yaml` describes the desired state of a Yeast project.

Yeast reads this file when you run commands such as `yeast up`, `yeast provision`, and `yeast status`.

If this is your first time editing the file, start with [Write `yeast.yaml`](../getting-started/write-yeast-yaml.md). This page is the complete reference after you know the basic shape.

## Minimal Example

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
```

## Complete Shape

```yaml
version: 1
management_host: 127.0.0.1

networks:
  - name: lab
    cidr: 10.10.10.0/24

provision:
  packages:
    - curl
  files:
    - source: ./file.txt
      destination: /home/yeast/file.txt
      permissions: "0644"
  shell:
    - echo "project provision"

instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    ssh_port: 2222
    user: yeast
    sudo: nopasswd
    env:
      APP_ENV: dev
    user_data: |
      #cloud-config
      package_update: true
    networks:
      - name: lab
        ipv4: 10.10.10.10
    provision:
      packages:
        - caddy
```

This example shows the supported v1.1 shape. You do not need every field in normal projects.

## Everyday Edit Cheat Sheet

Most projects only need one or two of these edits:

| Goal | Field | Example |
|---|---|---|
| Change RAM | `instances[].memory` | `memory: 2048` |
| Change CPU count | `instances[].cpus` | `cpus: 2` |
| Change disk size for a new disk | `instances[].disk_size` | `disk_size: 30G` |
| Change image | `instances[].image` | `image: debian-12` |
| Pin host SSH port | `instances[].ssh_port` | `ssh_port: 2222` |
| Change login user | `instances[].user` | `user: operator` |
| Allow passwordless sudo | `instances[].sudo` | `sudo: nopasswd` |
| Add packages/files/shell | `instances[].provision` | see [Provision Fields](#provision-fields) |
| Add a private lab IP | `instances[].networks[].ipv4` | `ipv4: 10.10.10.10` |

## Defaults

| Field | Default |
|---|---|
| `management_host` | `127.0.0.1` |
| `memory` | `512` |
| `cpus` | `1` |
| `hostname` | same as `name` |
| `user` | `yeast` |
| `sudo` | `none` |

## Validation Rules

Yeast validates config before starting VMs.

| Area | Rule |
|---|---|
| `version` | must be `1` |
| `management_host` | must be empty, `127.0.0.1`, `0.0.0.0`, or another valid IPv4 address |
| project networks | at most one project network |
| network CIDR | must be IPv4 CIDR |
| instances | at least one instance is required |
| instance names | unique; letters, numbers, `.`, `_`, and `-`; cannot contain `..` |
| hostnames | same naming style as instance names |
| memory | minimum `128` MiB when set |
| CPUs | minimum `1` when set |
| `ssh_port` | `1` through `65535` when set |
| `disk_size` | number with optional `K`, `M`, `G`, `T`, or `P` suffix |
| user | Linux-style user name, max 32 characters |
| sudo | `none`, `password`, or `nopasswd` |
| env keys | shell-style names such as `APP_ENV`; no newlines in values |
| network attachments | at most one private network per instance |
| static IPv4 | required for private attachments, inside the CIDR, not network/broadcast, and not duplicated |
| file permissions | three or four octal digits, for example `"0644"` |

## Top-Level Fields

| Field | Required | Purpose |
|---|---:|---|
| `version` | yes | Config schema version. Use `1` for v1.1. |
| `management_host` | no | Host IP used for management access such as SSH port forwarding. Defaults to `127.0.0.1`. |
| `networks` | no | Project private networks. v1.1 supports at most one project network. |
| `provision` | no | Provisioning shared by instances. |
| `instances` | yes | One or more VM definitions. |

## Management Host

`management_host` controls the host IP Yeast uses for management access such as SSH forwarding.

Most users should keep the default:

```yaml
management_host: 127.0.0.1
```

Use `0.0.0.0` only when you intentionally want management ports bound on all host interfaces.

!!! warning
    Binding management ports broadly can expose guest SSH to your network. Prefer `127.0.0.1` unless you understand the risk.

## Network Fields

| Field | Required | Purpose |
|---|---:|---|
| `networks[].name` | yes | Unique network name inside the project. |
| `networks[].cidr` | yes | IPv4 CIDR for the private network. |

Example:

```yaml
networks:
  - name: lab
    cidr: 10.10.10.0/24
```

In v1.1, every instance attached to this network must define a static `ipv4`.

## Provision Fields

Provisioning can be defined at the top level or per instance.

| Field | Required | Purpose |
|---|---:|---|
| `provision.packages` | no | Packages to install in the guest. |
| `provision.files` | no | Files to copy from host to guest. |
| `provision.files[].source` | yes | Local source path. |
| `provision.files[].destination` | yes | Guest destination path. |
| `provision.files[].permissions` | no | File mode such as `"0644"`. Quote modes so YAML keeps them as strings. |
| `provision.shell` | no | Shell commands to run after packages and files. |

Provisioning order is:

1. install packages
2. copy files
3. run shell commands

Top-level provisioning applies to instances. Instance-level provisioning adds instance-specific work.

## Instance Fields

| Field | Required | Purpose |
|---|---:|---|
| `instances[].name` | yes | Unique instance name. Used by commands such as `yeast ssh web`. |
| `instances[].hostname` | no | Guest hostname. Defaults to `name`. |
| `instances[].image` | yes | Supported image name. See [Images](images.md). |
| `instances[].memory` | no | Memory in MiB. Minimum is `128` when set. |
| `instances[].cpus` | no | vCPU count. Minimum is `1` when set. |
| `instances[].disk_size` | no | Overlay disk size. Applies when the instance disk is created. |
| `instances[].ssh_port` | no | Host-side management SSH port. |
| `instances[].user` | no | Linux user created/configured for access. |
| `instances[].sudo` | no | Sudo behavior: `none`, `password`, or `nopasswd`. |
| `instances[].env` | no | Environment-style values used by Yeast guest setup and provisioning context. |
| `instances[].user_data` | no | Custom cloud-init user data. Use carefully because it can override generated guest setup expectations. |
| `instances[].networks` | no | Private network attachments. v1.1 supports at most one attachment. |
| `instances[].networks[].name` | yes | Name of the project network to attach. |
| `instances[].networks[].ipv4` | yes | Static IPv4 inside the network CIDR. |
| `instances[].provision` | no | Instance-specific provisioning. |

## Disk Size

`disk_size` is used when Yeast creates the instance disk.

Examples:

```yaml
disk_size: 20G
disk_size: 512M
```

Changing `disk_size` later does not automatically resize an existing project disk. Recreate the instance disk if you need a different size.

## Sudo Policy

| Value | Meaning |
|---|---|
| `none` | no special sudo setup |
| `password` | sudo requires password behavior from the image/user setup |
| `nopasswd` | passwordless sudo for the configured user |

Use `nopasswd` for labs where provisioning commands need `sudo`.

## Environment Values

`env` is a string map:

```yaml
env:
  APP_ENV: dev
  ROLE: web
```

Keys must look like shell variable names. Values cannot contain newlines.

## Custom User Data

`user_data` lets you provide custom cloud-init user data:

```yaml
user_data: |
  #cloud-config
  package_update: true
```

Use it carefully. If custom user data conflicts with Yeast's expected user or SSH setup, the VM may boot but Yeast may not be able to connect.

## Unsupported In v1.1

Do not use these fields in public v1.1 docs:

- `ports`
- `host_port`
- `guest_port`
