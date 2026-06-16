# Lab 01: First VM, First SSH

Start one Ubuntu VM, check that Yeast sees it, connect with SSH, then clean it up.

You will learn:

- how a folder becomes a Yeast project
- how Yeast uses a trusted base image
- how `yeast up` turns `yeast.yaml` into a running VM
- how to enter the guest with `yeast ssh`
- the difference between stopping and destroying a project

## What You Will Build

```text
Linux host
└── yeast-lab-01/
    ├── yeast.yaml
    └── web VM
        └── Ubuntu 24.04
```

## Before You Start

Run:

```bash
yeast doctor
```

Continue only after blockers are fixed.

## Step 1: Create The Project

```bash
mkdir yeast-lab-01
cd yeast-lab-01
yeast init --template ubuntu-basic
```

Check what Yeast created:

```bash
find . -maxdepth 3 -type f | sort
sed -n '1,120p' yeast.yaml
```

Expected files:

- `yeast.yaml`
- `.yeast/project.json`
- `README.md`

## Step 2: Check Supported Images

```bash
yeast pull --list
```

This shows the trusted images Yeast knows how to use.

You do not need to download `ubuntu-24.04` manually for this lab. If the image is not cached yet, `yeast up` downloads and verifies it automatically.

Optional cache check:

```bash
yeast pull --cached
```

## Step 3: Start The VM

```bash
yeast up
```

Yeast validates the config, prepares a disk, generates cloud-init data, starts QEMU/KVM, waits for SSH, and records state for the project.

## Step 4: Verify Status

```bash
yeast status
```

Expected result:

- one instance named `web`
- status is running
- an SSH port is shown

## Step 5: SSH Into The VM

```bash
yeast ssh web
```

Inside the guest, run:

```bash
hostname
whoami
exit
```

The default user from the template is `yeast`.

## Step 6: Stop The VM

```bash
yeast down
```

This stops the VM but keeps the project disk.

You can start it again:

```bash
yeast up
```

## Clean Up

```bash
yeast down
yeast destroy
```

!!! warning
    `yeast destroy` removes tracked runtime files and disks for this project.

## What You Learned

You completed the smallest useful Yeast loop:

```text
init -> up -> status -> ssh -> down -> destroy
```

You also saw that Yeast projects are folder-based. The config lives in `yeast.yaml`, while runtime state is tracked separately.

## Next Lab

Continue with [Cloud-Init Basics](02-cloud-init-basics.md).
