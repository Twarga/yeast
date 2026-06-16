# Installation

Choose the path that matches your machine.

!!! note
    Yeast is Linux-first. Native Linux is the supported path.
    Windows users can try Yeast through WSL 2, but that path is beta and may be unpredictable.
    macOS is not covered yet.

## Pick Your Path

| Path | Status | Best For | Read |
|---|---|---|---|
| [Linux](installation-linux.md) | Supported | Native Linux hosts with QEMU/KVM | Install on Linux |
| [Windows with WSL 2](installation-windows-wsl.md) | Beta | Windows users who want to experiment | Install on Windows with WSL |

If you are not sure, use the Linux path.

## Before You Install

You should be able to run:

```bash
yeast doctor
```

If Yeast is not installed yet, read the page for your operating system first.

## After Installation

Verify the install:

```bash
yeast version
yeast doctor
```

Then continue with the [Quickstart](quickstart.md).
