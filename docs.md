# Yeast Documentation

Yeast is a command-line tool for creating and managing local virtual machines on Linux with KVM and QEMU.

This guide is written for real users, including beginners. It explains:
- what Yeast is
- how Yeast works
- how to install and use it
- how to write `yeast.yaml`
- what every command does
- what networking modes are available
- what Yeast does not support yet

## What Yeast Is

Yeast is a **project-based local VM orchestrator**.

You create a project folder, add a `yeast.yaml` file, and Yeast manages the VMs defined in that file. It is designed for local development workflows where you want:
- real virtual machines, not containers
- shared base images instead of one large copy per project
- cloud-init for guest setup
- a small CLI that is easy to automate

Yeast is closer to **Vagrant's project model** than to a full-machine hypervisor manager.

Important:
- Yeast manages the VMs for the **current project folder**
- one project can define **multiple instances**
- Yeast does **not** provide a global dashboard for every VM on the machine

## How Yeast Works

At a high level:

1. You define one or more instances in `yeast.yaml`
2. You download a trusted base image with `yeast pull`
3. You run `yeast up`
4. Yeast creates a copy-on-write overlay disk for each VM
5. Yeast generates a cloud-init ISO
6. Yeast starts QEMU with KVM acceleration
7. Yeast waits for SSH login readiness
8. Yeast stores runtime state in `yeast.state`

Yeast stores data in two places:

Project directory:
- `yeast.yaml`: your VM definitions
- `yeast.state`: Yeast's local runtime state for this project
- `yeast.state.lock`: temporary lock file while a command is mutating state

Home directory:
- `~/.yeast/cache/`: shared base images
- `~/.yeast/instances/<name>/`: per-instance runtime files

Per-instance runtime files include:
- `disk.qcow2`
- `seed.iso`
- `user-data`
- `meta-data`
- `vm.log`

## Supported Platforms and Requirements

Yeast currently targets:
- Linux only

You need:
- KVM support enabled on the host
- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage`
- `ssh`
- an SSH public key in one of:
  - `~/.ssh/id_ed25519.pub`
  - `~/.ssh/id_rsa.pub`
- your user in the `kvm` group in most non-root setups

Example package installs:

```bash
# Ubuntu / Debian
sudo apt install qemu-system-x86 qemu-utils genisoimage

# Fedora / RHEL
sudo dnf install qemu-system-x86 qemu-img genisoimage

# Arch Linux
sudo pacman -S qemu-base cdrtools
```

Add your user to the `kvm` group if needed:

```bash
sudo usermod -aG kvm $USER
```

Then log out and log back in.

## Installation

Current repository documentation is source-first.

### One-command installer

If you already have this repository available locally, run:

```bash
bash install.sh
```

What the installer does:
- detects the package manager
- installs Yeast host dependencies
- installs Go if needed for the source build path
- clones/builds Yeast
- installs the binary into `/usr/local/bin/yeast`
- creates `~/.yeast/cache/`
- generates an SSH key if the user does not already have one
- attempts to add the user to the `kvm` group when available

Installer environment overrides:

```bash
YEAST_REPO_URL=https://github.com/Twarga/yeast.git YEAST_REF=dev bash install.sh
```

Note:
- if the repo is private, the installer can only clone it if Git access is already authenticated
- if the installer adds you to the `kvm` group, log out and back in before your first `yeast up`

Build from source:

```bash
git clone <repository-url>
cd <repo-directory>
go build -o yeast ./cmd/yeast
sudo mv yeast /usr/local/bin/
```

Check the CLI:

```bash
yeast --help
```

## Quick Start

### 1. Check your host

Run:

```bash
yeast doctor
```

This checks:
- QEMU binaries
- KVM device access
- `kvm` group membership
- SSH key presence
- image cache directory

If there are blockers, fix them first.

### 2. Create a project

```bash
mkdir my-project
cd my-project
yeast init
```

This creates:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 1
    user: yeast
    sudo: none
```

You can also generate a starter config with your own values:

```bash
yeast init \
  --name api \
  --image ubuntu-24.04 \
  --memory 2048 \
  --cpus 2 \
  --user operator \
  --sudo password
```

### 3. Download a trusted image

List supported pullable images first:

```bash
yeast pull --list
```

Then pull one:

```bash
yeast pull ubuntu-22.04
```

Currently supported images:
- `ubuntu-22.04`
- `ubuntu-24.04`

### 4. Start the project VMs

```bash
yeast up
```

### 5. Check status

```bash
yeast status
```

### 6. Connect over SSH

```bash
yeast ssh web
```

### 7. Stop the VMs

```bash
yeast down
```

## Available Images

Use this command to see every built-in trusted image Yeast can download:

```bash
yeast pull --list
```

Current built-in images:
- `ubuntu-22.04`
- `ubuntu-24.04`

What the list shows:
- image name
- source URL
- pinned SHA256 checksum

If you are new to Yeast, start here before running `yeast pull <image>`.

## Yeast Project Model

Yeast is **project-local**.

That means:
- `yeast up` starts the instances defined in the `yeast.yaml` in your current folder
- `yeast status` reads `yeast.state` in your current folder
- `yeast down`, `halt`, `restart`, and `destroy` operate on that project's tracked state

One project can define multiple VMs:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 1

  - name: db
    image: ubuntu-22.04
    memory: 2048
    cpus: 2
```

But Yeast does not have a built-in global command like "show every VM on this machine."

## The `yeast.yaml` File

### Schema

Current config version:

```yaml
version: 1
instances:
  - ...
```

You must define at least one instance.

### Minimal Example

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
```

Defaults applied automatically:
- `memory: 512`
- `cpus: 1`
- `user: yeast`
- `sudo: none`

### Full Example

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 1
    user: yeast
    sudo: none
    env:
      APP_ENV: development
      LOG_LEVEL: debug
```

### Field Reference

#### `version`

Required.

Only supported value today:

```yaml
version: 1
```

#### `instances`

Required list of VM definitions.

#### `name`

Required.

This is:
- the instance name used in commands like `yeast ssh <name>`
- the key in `yeast.state`
- the directory name under `~/.yeast/instances/<name>/`

Allowed characters:
- letters
- numbers
- `.`
- `_`
- `-`

Examples:

```yaml
name: web
name: db-server
name: worker_1
```

#### `image`

Required.

This must match the base image filename in the Yeast cache, without the `.img` suffix.

Example:

```yaml
image: ubuntu-22.04
```

Yeast expects the base image at:

```text
~/.yeast/cache/ubuntu-22.04.img
```

#### `memory`

Optional.

Memory in MB.

Default:

```yaml
memory: 512
```

Example:

```yaml
memory: 2048
```

#### `cpus`

Optional.

Number of virtual CPU cores.

Default:

```yaml
cpus: 1
```

Example:

```yaml
cpus: 2
```

#### `disk_size`

Optional.

Requested virtual disk size for the instance overlay disk.

Examples:

```yaml
disk_size: 20G
disk_size: 25600M
```

Behavior:
- if omitted, Yeast uses the base image's default virtual size
- if set on first boot, Yeast creates the overlay disk with that size
- if increased later, Yeast grows the existing overlay disk on the next `yeast up` or `yeast restart`
- Yeast does not shrink disks automatically

#### `user`

Optional.

Bootstrap username used when Yeast generates cloud-init automatically.

Default:

```yaml
user: yeast
```

Example:

```yaml
user: operator
```

Important:
- this affects generated cloud-init
- it does **not** automatically change the default username used by `yeast ssh`
- if you set `user: operator`, connect with:

```bash
yeast ssh web --user operator
```

#### `sudo`

Optional.

Supported values:
- `none`
- `password`
- `nopasswd`

Default:

```yaml
sudo: none
```

Behavior:
- `none`: no sudo rule is added
- `password`: generates `sudo: ALL=(ALL) ALL`
- `nopasswd`: generates `sudo: ALL=(ALL) NOPASSWD:ALL`

Examples:

```yaml
sudo: none
sudo: password
sudo: nopasswd
```

#### `env`

Optional.

Key/value environment variables to export via cloud-init when Yeast generates the bootstrap config.

Example:

```yaml
env:
  APP_ENV: development
  API_BASE_URL: http://example.test
```

Yeast writes these into:

```text
/etc/profile.d/yeast-env.sh
```

Important:
- keys must look like shell variable names
- values cannot contain newlines
- this only applies when Yeast generates cloud-init automatically

#### `user_data`

Optional.

Use this when you want to provide your own cloud-init configuration.

Example:

```yaml
user_data: |
  #cloud-config
  packages:
    - nginx
    - git
  runcmd:
    - systemctl enable --now nginx
```

Important behavior:
- if `user_data` is present, Yeast uses it as the cloud-init user-data
- Yeast adds `#cloud-config` automatically if you omit it
- Yeast does **not** merge your custom `user_data` with generated values

That means when you use `user_data`, Yeast does **not** automatically inject:
- your SSH key
- the configured `user`
- the configured `sudo`
- the configured `env`

If you use custom `user_data` and still want SSH access, you must define your own user and SSH authorized keys inside that cloud-init content.

Example custom `user_data` with SSH access:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    user_data: |
      #cloud-config
      users:
        - name: yeast
          shell: /bin/bash
          ssh-authorized-keys:
            - ssh-ed25519 AAAA...replace-with-your-real-key...
      packages:
        - nginx
      runcmd:
        - systemctl enable --now nginx
```

### YAML Validation Rules

Yeast validates the config before starting VMs.

Current rules include:
- `version` must be `1`
- at least one instance is required
- each instance name must be unique
- `image` cannot be empty
- `memory` must be `0` or at least `128`
- `cpus` must be `0` or at least `1`
- `disk_size`, if provided, must use a supported size like `20G`, `10240M`, or raw bytes
- `user`, if provided, must be a lowercase Linux-style username
- `sudo`, if provided, must be `none`, `password`, or `nopasswd`
- env keys must be valid shell variable names
- env values cannot contain newlines

## Command Reference

### Global Flag: `--json`

Most major commands support machine-readable output with:

```bash
yeast <command> --json
```

Commands with JSON support:
- `init`
- `doctor`
- `pull`
- `up`
- `status`
- `down`
- `halt`
- `restart`
- `destroy`

`yeast ssh` is interactive and does not support JSON output.

### `yeast doctor`

Check whether the host is ready to run Yeast.

Usage:

```bash
yeast doctor
yeast doctor --json
```

Use this before your first `yeast up` on a machine.

### `yeast init`

Create a starter `yeast.yaml` in the current directory.

Usage:

```bash
yeast init
yeast init --json
yeast init --name api --image ubuntu-24.04 --memory 2048 --cpus 2 --disk-size 25G
```

Behavior:
- creates a basic one-instance config
- lets you choose starter values with flags
- fails if `yeast.yaml` already exists

Useful flags:
- `--name`
- `--image`
- `--memory`
- `--cpus`
- `--disk-size`
- `--user`
- `--sudo`

Example:

```bash
yeast init \
  --name api \
  --image ubuntu-24.04 \
  --memory 2048 \
  --cpus 2 \
  --disk-size 25G \
  --user operator \
  --sudo password
```

### `yeast pull`

Download a trusted base image and verify its SHA256 checksum.

Usage:

```bash
yeast pull --list
yeast pull ubuntu-22.04
yeast pull ubuntu-24.04
yeast pull ubuntu-22.04 --force
yeast pull ubuntu-22.04 --retries 5 --timeout 45m
```

Flags:
- `--list`
- `--force`
- `--retries`
- `--timeout`

Behavior:
- `--list` shows every built-in trusted image Yeast can pull
- downloads to `~/.yeast/cache/<image>.img`
- verifies checksum while downloading
- retries transient failures
- replaces an existing broken image only when `--force` is used

Use `yeast pull --list` when you are not sure which image names are valid.

### `yeast up`

Start all instances defined in the current project's `yeast.yaml`.

Usage:

```bash
yeast up
yeast up --json
yeast up --network-mode user
yeast up --network-mode private
yeast up --network-mode bridge --bridge br0
```

Behavior:
- reads `yeast.yaml`
- creates overlay disks if needed
- generates cloud-init artifacts
- allocates SSH host ports automatically
- starts QEMU in the background
- waits until SSH login is actually ready
- stores runtime state in `yeast.state`

Current scope:
- starts **all configured instances**
- does **not** support `yeast up <name>`

### `yeast status`

Show tracked instances for the current project.

Usage:

```bash
yeast status
yeast status --json
```

Typical output:

```text
NAME    STATUS    PID     IP           SSH PORT
web     running   12345   127.0.0.1    45678
```

Behavior:
- reads `yeast.state`
- reconciles stale process state first
- prints tracked instances, including stopped ones

### `yeast ssh`

Open an SSH session to a running instance.

Usage:

```bash
yeast ssh web
yeast ssh web --user operator
yeast ssh web --insecure
```

Flags:
- `--user`
- `--insecure`

Behavior:
- connects to `127.0.0.1:<ssh_port>`
- uses the system `ssh` binary
- replaces the current process with `ssh`

Important:
- default SSH username is `yeast`
- if your instance was created with a different bootstrap user, pass `--user`

Use `--insecure` only when you explicitly want to disable host key verification.

### `yeast halt`

Stop one or more tracked instances.

Usage:

```bash
yeast halt web
yeast halt web db
yeast halt
yeast halt --json
```

Behavior:
- with names: stops only those tracked instances
- without names: stops all tracked instances in the current project state

### `yeast down`

Stop all tracked instances in the current project state.

Usage:

```bash
yeast down
yeast down --json
```

Behavior:
- stops tracked instances from `yeast.state`
- does not delete disks or instance directories

### `yeast restart`

Restart one or more configured instances.

Usage:

```bash
yeast restart web
yeast restart web db
yeast restart
yeast restart --network-mode private
yeast restart --network-mode bridge --bridge br0
yeast restart --json
```

Behavior:
- with names: restarts only those configured instances
- without names: restarts all instances in `yeast.yaml`
- reuses the existing overlay disk

Important:
- `restart` is not the same as "rebuild from scratch"
- network mode is selected at command time

### `yeast destroy`

Stop instances and remove their local instance data.

Usage:

```bash
yeast destroy web
yeast destroy web db
yeast destroy
yeast destroy --json
```

Behavior:
- stops the instance if it is running
- removes `~/.yeast/instances/<name>/`
- removes the instance entry from `yeast.state`

Use this when you want a clean reprovision on the next `yeast up`.

### `yeast completion`

The CLI also exposes shell completion support through Cobra:

```bash
yeast completion bash
yeast completion zsh
```

Use `yeast completion --help` for shell-specific instructions.

## Networking

Yeast currently supports three network modes.

### `user` mode

Default mode.

Behavior:
- QEMU user-mode NAT
- no direct LAN exposure
- SSH is available through a forwarded host port

Use it when:
- you want the simplest setup
- you are getting started
- you do not need LAN presence for the guest

Example:

```bash
yeast up --network-mode user
```

### `private` mode

Restricted user-mode networking.

Behavior:
- QEMU user-mode NAT
- `restrict=on`
- SSH still works through a forwarded host port

Use it when:
- you want tighter isolation
- your VM does not need normal outbound network access

Example:

```bash
yeast up --network-mode private
```

### `bridge` mode

Attach the guest to a host bridge and keep a separate restricted management NIC for SSH.

Behavior:
- guest can join the LAN through the selected bridge
- Yeast still keeps SSH management via host port forwarding

Use it when:
- you need the VM to behave more like a LAN-attached machine
- you already have a host bridge configured

Example:

```bash
yeast up --network-mode bridge --bridge br0
```

Important:
- bridge mode requires `--bridge <bridge-name>`
- Yeast does not create the host bridge for you

### Networking Rules and Limits

Current behavior:
- network flags are supported on `up` and `restart`
- the selected network mode applies to every instance started by that command
- network mode is **not** stored in `yeast.yaml`
- network mode is **not** remembered in `yeast.state`

This means:
- if you run `yeast up --network-mode bridge --bridge br0`
- and later run `yeast restart web` without flags
- the restart falls back to the default `user` mode

### Port Forwarding

Yeast currently supports **automatic SSH port forwarding only**.

What Yeast does:
- chooses a free host TCP port
- forwards `host:<random_port>` to `guest:22`
- stores that port in `yeast.state`
- uses it for `yeast ssh`

What Yeast does not currently do:
- custom application port forwards like `8080:80`
- per-instance forwarded port configuration in YAML
- multiple arbitrary forwarding rules

## Common Use Cases

### Single VM for development

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 1
```

Use this when:
- you want one Linux VM for testing code, packages, or scripts

### Multi-VM local stack

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 1

  - name: db
    image: ubuntu-22.04
    memory: 2048
    cpus: 2

  - name: worker
    image: ubuntu-22.04
    memory: 1024
    cpus: 1
```

Use this when:
- you want several local VMs inside one project
- you want one Yeast workflow for a small environment

### Default bootstrap with environment variables

```yaml
version: 1
instances:
  - name: app
    image: ubuntu-22.04
    user: yeast
    sudo: password
    env:
      APP_ENV: development
      APP_PORT: "8080"
```

Use this when:
- you want a default Yeast user
- you want environment variables available in guest shell sessions

### Full custom cloud-init

```yaml
version: 1
instances:
  - name: nginx
    image: ubuntu-22.04
    user_data: |
      #cloud-config
      users:
        - name: yeast
          shell: /bin/bash
          ssh-authorized-keys:
            - ssh-ed25519 AAAA...replace-with-your-real-key...
      packages:
        - nginx
      runcmd:
        - systemctl enable --now nginx
```

Use this when:
- you need direct control of cloud-init behavior
- you understand that Yeast will not merge in user, sudo, env, or SSH settings for you

## Files and Directories

### In the project directory

`yeast.yaml`
- your desired VM definitions

`yeast.state`
- Yeast's runtime record of tracked instances

`yeast.state.lock`
- temporary lock file while mutating commands are running

### In the home directory

`~/.yeast/cache/`
- trusted base images

`~/.yeast/instances/<name>/`
- per-instance runtime data

Typical contents:
- `disk.qcow2`
- `seed.iso`
- `user-data`
- `meta-data`
- `vm.log`
- rotated `vm.<timestamp>.<unixnano>.log` archives

## Logging

Yeast captures QEMU stdout and stderr into a per-instance log file.

Active log:

```text
~/.yeast/instances/<name>/vm.log
```

Rotated logs:

```text
~/.yeast/instances/<name>/vm.<timestamp>.<unixnano>.log
```

Archive retention:
- default is `5`
- configured with `YEAST_VM_LOG_RETENTION`

Example:

```bash
export YEAST_VM_LOG_RETENTION=10
```

## JSON Output for Automation

Most major commands support JSON output.

Example:

```bash
yeast status --json
```

Top-level structure:

```json
{
  "schema": "yeast.command.v1",
  "command": "status",
  "ok": true,
  "data": {},
  "error": null
}
```

Use JSON output when:
- writing scripts
- building CI checks
- integrating Yeast into local automation

## Troubleshooting

### First command to run

If something does not work, start here:

```bash
yeast doctor
```

### Check state

```bash
cat yeast.state
```

Look for:
- stale `running` entries with dead PIDs
- missing or unexpected SSH ports

### Check VM logs

```bash
tail -n 200 ~/.yeast/instances/web/vm.log
```

### Verify images exist

```bash
ls -lah ~/.yeast/cache/
```

### Force a clean reprovision

If an instance has persistent overlay state you no longer want:

```bash
yeast halt web
yeast destroy web
yeast up
```

### Common Gotchas

#### "I changed `user` in YAML, but `yeast ssh` still fails"

By default, `yeast ssh` uses `--user yeast`.

If you changed the bootstrap user:

```bash
yeast ssh web --user operator
```

#### "I changed `user_data`, but the VM did not behave like a fresh machine"

The overlay disk is persistent. `restart` is not a rebuild.

To get a clean reprovision:

```bash
yeast destroy web
yeast up
```

#### "I used custom `user_data` and now SSH does not work"

When you provide `user_data`, Yeast does not inject your SSH key automatically.

Add your own user and `ssh-authorized-keys` in the cloud-init content.

#### "I restarted and lost bridge mode"

Network mode is selected on `up` and `restart` flags and is not persisted in the config or state.

Restart with the flags again:

```bash
yeast restart web --network-mode bridge --bridge br0
```

## Current Limits

Yeast is intentionally small today.

It does **not** currently provide:
- a GUI or web control panel
- a VM marketplace or template store
- arbitrary custom port forwarding
- per-instance network settings in `yeast.yaml`
- snapshots
- suspend/resume
- a global "all VMs on this host" inventory command
- single-instance `yeast up <name>`
- guest IP discovery in bridge mode
- automatic merge between custom `user_data` and generated bootstrap config
- reusable cloud-init setup templates or provisioning presets

## Important Scope Notes

### Project-based, not host-global

Yeast behaves like a project tool, not like a whole-machine VM manager.

### Multiple VMs per project are supported

One `yeast.yaml` can define several instances.

### Instance directories are global by name

Yeast stores instance files under:

```text
~/.yeast/instances/<name>/
```

Because the directory is keyed by instance name, using the same instance name in multiple separate projects can cause collisions. For now, keep instance names unique across projects when possible.

## Recommended Beginner Workflow

If you are new to Yeast, use this flow:

1. Run `yeast doctor`
2. Run `yeast init`
3. Keep the default config first
4. Run `yeast pull ubuntu-22.04`
5. Run `yeast up`
6. Run `yeast status`
7. Run `yeast ssh web`
8. When done, run `yeast down`
9. Only after that, start customizing `memory`, `cpus`, `user`, `sudo`, `env`, or `user_data`

## Summary

Yeast is a Linux-native, project-based VM tool with:
- declarative YAML config
- trusted image pulls
- cloud-init provisioning
- local state tracking
- SSH access through automatic host port forwarding
- simple lifecycle management

It is a good fit when you want real local VMs with a small CLI and predictable behavior.
