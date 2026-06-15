# Lab 02: Cloud-Init Basics

Customize first boot settings and verify them from inside the VM.

You will learn:

- how `hostname` becomes the guest hostname
- how `user` controls the Linux login user
- how `sudo` changes privilege behavior
- why cloud-init runs before Yeast provisioning
- where cloud-init stops and provisioning begins

## What You Will Build

```text
yeast-lab-02/
└── web VM
    ├── hostname: cloudinit-lab
    ├── user: yeast
    └── sudo: nopasswd
```

## Before You Start

Run:

```bash
yeast doctor
```

You should already understand the basic loop from Lab 01.

## Step 1: Create The Project

```bash
mkdir yeast-lab-02
cd yeast-lab-02
yeast init
```

Replace `yeast.yaml` with:

```yaml
version: 1
instances:
  - name: web
    hostname: cloudinit-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    user: yeast
    sudo: nopasswd
```

## Step 2: Start The VM

```bash
yeast up
```

Cloud-init runs during the guest's first boot. It prepares the hostname, user, SSH access, and sudo policy before Yeast can connect for guest operations.

## Step 3: Verify The Hostname

```bash
yeast exec web -- hostname
```

Expected output:

```text
cloudinit-lab
```

## Step 4: Verify The User

```bash
yeast exec web -- whoami
```

Expected output:

```text
yeast
```

## Step 5: Verify Sudo Policy

```bash
yeast exec web -- sudo -n true
```

The command should exit successfully because `sudo: nopasswd` allows passwordless sudo for this lab.

## Step 6: Inspect The VM

```bash
yeast inspect web
```

Use inspect when you want detailed state for one instance.

## Clean Up

```bash
yeast down
yeast destroy
```

## What You Learned

Cloud-init is the first-boot setup layer. Yeast uses it to make the VM reachable and usable.

Provisioning is different: provisioning happens after SSH is ready. You will use that next.

## Next Lab

Continue with [Provisioning After Boot](03-provisioning-after-boot.md).
