# yeast.yaml Reference

`yeast.yaml` describes the desired state of a Yeast project.

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

## Defaults

| Field | Default |
|---|---|
| `management_host` | `127.0.0.1` |
| `memory` | `512` |
| `cpus` | `1` |
| `hostname` | same as `name` |
| `user` | `yeast` |
| `sudo` | `none` |

## Top-Level Fields

| Field | Required | Purpose |
|---|---:|---|
| `version` | yes | Config schema version. Use `1` for v1.1. |
| `management_host` | no | Host IP used for management access such as SSH port forwarding. Defaults to `127.0.0.1`. |
| `networks` | no | Project private networks. v1.1 supports at most one project network. |
| `provision` | no | Provisioning shared by instances. |
| `instances` | yes | One or more VM definitions. |

## Network Fields

| Field | Required | Purpose |
|---|---:|---|
| `networks[].name` | yes | Unique network name inside the project. |
| `networks[].cidr` | yes | IPv4 CIDR for the private network. |

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
| `instances[].networks[].ipv4` | no | Static IPv4 inside the network CIDR. |
| `instances[].provision` | no | Instance-specific provisioning. |

## Unsupported In v1.1

Do not use these fields in public v1.1 docs:

- `ports`
- `host_port`
- `guest_port`
