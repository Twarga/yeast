# Lab 01: First VM, First SSH

In this lab, you will start one Ubuntu VM and connect to it with SSH.

You will learn:

- `yeast init`
- `yeast pull`
- `yeast up`
- `yeast status`
- `yeast ssh`
- `yeast down`
- `yeast destroy`

## What You Will Build

```text
host
└── Yeast project
    └── web VM
        └── Ubuntu 24.04
```

## Before You Start

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

## Step 2: Pull The Image

```bash
yeast pull ubuntu-24.04
```

## Step 3: Start The VM

```bash
yeast up
```

Yeast creates the VM disk, generates cloud-init data, starts QEMU/KVM, and waits for SSH.

## Step 4: Check Status

```bash
yeast status
```

Expected result: one running instance named `web`.

## Step 5: SSH Into The VM

```bash
yeast ssh web
```

Inside the guest:

```bash
hostname
whoami
exit
```

## Step 6: Clean Up

```bash
yeast down
yeast destroy
```

## What You Learned

You learned the smallest complete Yeast workflow:

```text
init -> pull -> up -> status -> ssh -> down -> destroy
```

Next: [Cloud-Init Basics](02-cloud-init-basics.md).
