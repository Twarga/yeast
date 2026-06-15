# Lab 02: Cloud-Init Basics

In this lab, you will see how Yeast prepares a VM for first boot.

You will learn:

- how `hostname` becomes the guest hostname
- how the default `yeast` user is created
- how SSH key access is prepared
- why cloud-init runs before Yeast provisioning

## What You Will Build

One Ubuntu VM with a predictable hostname and SSH user.

## Step 1: Create The Project

```bash
mkdir yeast-lab-02
cd yeast-lab-02
yeast init
```

Edit `yeast.yaml`:

```yaml
version: 1
instances:
  - name: web
    hostname: cloudinit-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    user: yeast
    sudo: none
```

## Step 2: Start The VM

```bash
yeast up
```

## Step 3: Verify The Guest

```bash
yeast exec web -- hostname
yeast exec web -- whoami
```

Expected:

```text
cloudinit-lab
yeast
```

## What Happened

Yeast generated cloud-init data before starting QEMU. Cloud-init prepared the guest user, hostname, and SSH access during first boot.

## Clean Up

```bash
yeast down
yeast destroy
```

Next: [Provisioning After Boot](03-provisioning-after-boot.md).
