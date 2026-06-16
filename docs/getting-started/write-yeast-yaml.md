# Write `yeast.yaml`

`yeast.yaml` is the main file you edit in a Yeast project.

It describes the VMs you want. Yeast reads it when you run commands such as `yeast up`, `yeast status`, `yeast provision`, and `yeast destroy`.

## Where This File Lives

Run Yeast commands from your project folder:

```bash
mkdir my-lab
cd my-lab
yeast init
```

After `yeast init`, edit:

```text
my-lab/yeast.yaml
```

Use any editor:

```bash
nano yeast.yaml
```

or:

```bash
code yeast.yaml
```

## The Smallest Useful File

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
```

This creates one VM named `web` from the `ubuntu-24.04` image.

If you leave out RAM, CPU, user, and hostname, Yeast applies defaults:

| Field | Default |
|---|---|
| `memory` | `512` MiB |
| `cpus` | `1` |
| `hostname` | same as `name` |
| `user` | `yeast` |
| `sudo` | `none` |
| `management_host` | `127.0.0.1` |

## Common Edits

Most beginner edits happen inside one item under `instances`.

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 2048
    cpus: 2
    disk_size: 30G
    ssh_port: 2222
    user: yeast
    sudo: nopasswd
```

| Want | Edit | Notes |
|---|---|---|
| More RAM | `memory: 2048` | Value is MiB, so `2048` means 2 GiB. |
| More CPU | `cpus: 2` | Minimum is `1`. |
| Bigger disk | `disk_size: 30G` | Applies when the instance disk is created. Existing disks are not automatically resized. |
| Different image | `image: debian-12` | Use an image from the supported image list. |
| Fixed SSH port | `ssh_port: 2222` | Useful when you want predictable host ports. |
| Passwordless sudo | `sudo: nopasswd` | Useful for labs and provisioning commands. |
| Different login user | `user: operator` | Must be a Linux-style username. |
| Different hostname | `hostname: web-lab` | Defaults to `name` if omitted. |

After editing, run:

```bash
yeast up
```

If the VM already exists, some changes affect the next boot, while disk-size changes may require recreating the VM disk.

## Pick An Image

List supported images:

```bash
yeast pull --list
```

Then set the image:

```yaml
instances:
  - name: web
    image: ubuntu-24.04
```

Supported auto-download images are fetched by `yeast up` when missing. Manual/setup-only images print instructions instead.

See the full [image reference](../reference/images.md).

## Add Provisioning

Provisioning runs after the VM becomes reachable over SSH.

Use it to install packages, copy files, and run shell commands.

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    sudo: nopasswd
    provision:
      packages:
        - curl
        - caddy
      files:
        - source: ./site/index.html
          destination: /home/yeast/index.html
          permissions: "0644"
      shell:
        - sudo install -D -m 0644 /home/yeast/index.html /var/www/html/index.html
        - sudo systemctl restart caddy
```

Provisioning order is:

1. install packages
2. copy files
3. run shell commands

Use `sudo: nopasswd` when your provisioning commands need `sudo -n` behavior.

## Add A Second VM

Add another item under `instances`.

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    ssh_port: 2222

  - name: db
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    ssh_port: 2223
```

Start both:

```bash
yeast up
yeast status
```

Connect to one:

```bash
yeast ssh web
```

or:

```bash
yeast ssh db
```

## Add A Private Network

Use a project network when VMs should talk to each other on a private lab IP.

In v1.1, Yeast supports one project private network and one private attachment per instance.

```yaml
version: 1
networks:
  - name: lab
    cidr: 10.10.10.0/24

instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2222
    networks:
      - name: lab
        ipv4: 10.10.10.10

  - name: db
    image: ubuntu-24.04
    ssh_port: 2223
    networks:
      - name: lab
        ipv4: 10.10.10.20
```

After `yeast up`, test from `web`:

```bash
yeast ssh web
ping -c 2 10.10.10.20
exit
```

## Validate The File Safely

The simplest validation is:

```bash
yeast up
```

Yeast validates `yeast.yaml` before starting the VM. If the file is invalid, Yeast stops before creating or changing the VM.

Useful checks:

```bash
yeast status
yeast inspect web
yeast logs web --tail 80
```

For automation:

```bash
yeast status --json
```

## Common Mistakes

| Mistake | Fix |
|---|---|
| Editing `yeast.yaml` from the wrong folder | Run `pwd` and make sure you are inside the project folder. |
| Writing `cpu` instead of `cpus` | Use `cpus`. |
| Writing `ram` instead of `memory` | Use `memory`, measured in MiB. |
| Forgetting `version: 1` | Add it at the top. |
| Using a duplicate `name` | Every instance name must be unique. |
| Reusing the same `ssh_port` | Give each VM a different host-side SSH port. |
| Setting `disk_size` after a disk already exists | Destroy/recreate the VM if you need the disk created with the new size. |
| Using an IPv4 outside the network CIDR | Pick an address inside the configured `cidr`. |

## When To Use The Reference

This page teaches the normal editing path.

Use the [full `yeast.yaml` reference](../reference/yeast-yaml.md) when you need every field, validation rule, and complete shape in one place.
