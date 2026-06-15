# Lifecycle

The core Yeast lifecycle is:

```text
init -> pull -> up -> use -> down -> destroy
```

## Initialize

```bash
yeast init
```

Creates the project config and identity.

## Start

```bash
yeast up
```

Validates config, prepares disks, starts QEMU/KVM, waits for SSH, and runs provisioning.

## Stop

```bash
yeast down
```

Stops running VMs while preserving disks.

## Destroy

```bash
yeast destroy
```

Removes tracked runtime files and disks for the project.

!!! warning
    Use `yeast down` when you want to stop. Use `yeast destroy` when you want to remove the lab.
