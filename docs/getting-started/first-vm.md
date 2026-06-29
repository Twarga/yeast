# First VM

This walkthrough explains the first Yeast VM slowly.

Use it if you want to understand what each command does.

## Create A Folder

```bash
mkdir first-yeast-vm
cd first-yeast-vm
```

Yeast projects are folder-based. Run commands from the project folder.

## Initialize

```bash
yeast init
```

This writes a starter `yeast.yaml` and a project identity file.

The starter config looks like this:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
```

Read that file as:

| Field | Meaning |
|---|---|
| `name: web` | The VM name. You use it in commands such as `yeast ssh web`. |
| `image: ubuntu-24.04` | The base image Yeast uses. If it is not cached, `yeast up` downloads it. |
| `memory: 1024` | RAM in MiB. |
| `cpus: 1` | Number of virtual CPUs. |

If you want to change RAM, CPU, disk size, image, user, sudo, provisioning, or networks, read [Write `yeast.yaml`](write-yeast-yaml.md).

If you want to open a guest web app, API, or dashboard directly from your laptop, add:

```yaml
ports:
  - "8080:80"
```

That forwards host `127.0.0.1:8080` to guest port `80`.

## Start

```bash
yeast up
```

The first run can take longer because Yeast may download the image and the guest must complete cloud-init.

Expected result:

- Yeast validates `yeast.yaml`
- the Ubuntu image is downloaded if missing
- a VM named `web` starts
- SSH becomes ready

## Connect

```bash
yeast ssh web
```

Try:

```bash
hostname
ip addr
exit
```

Expected result:

- `hostname` prints `web`
- `ip addr` shows normal Linux network interfaces
- `exit` returns you to the host terminal

## Inspect From The Host

```bash
yeast status
yeast inspect web
yeast logs web --tail 80
```

These commands are useful when a VM is running but you want to understand what Yeast knows about it.

Use `status` for the quick summary, `inspect` for one VM in detail, and `logs` when boot or SSH readiness is confusing.

## Clean Up

```bash
yeast down
yeast destroy
```

## What You Learned

You learned the basic Yeast loop:

```text
init -> up -> ssh -> status -> down -> destroy
```

You also saw the next useful edit point: `yeast.yaml` can expose guest services to your laptop with `ports`, for example `ports: ["8080:80"]`.
