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

## Start

```bash
yeast up
```

The first run can take longer because Yeast may download the image and the guest must complete cloud-init.

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

## Inspect From The Host

```bash
yeast status
yeast inspect web
yeast logs web --tail 80
```

These commands are useful when a VM is running but you want to understand what Yeast knows about it.

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
