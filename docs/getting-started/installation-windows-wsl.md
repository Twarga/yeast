# Install on Windows With WSL

This page is a beta path for experimenting with Yeast on Windows through WSL 2.

!!! warning
    This path is experimental and may be unpredictable.
    Yeast is still Linux-first, and the supported path is native Linux.
    If `yeast doctor` reports missing `/dev/kvm` or another virtualization blocker, stop here and use Linux instead.

## What This Page Covers

This page shows how to:

- install WSL 2
- install a Linux distro inside WSL
- install Yeast inside that Linux distro
- check whether your machine is usable for Yeast

## Before You Start

You need:

- a Windows machine with WSL 2 support
- hardware virtualization enabled in BIOS or UEFI
- PowerShell or Windows Terminal with administrator access for WSL setup
- a Linux distro inside WSL, such as Ubuntu

## 1. Install WSL

Open an administrator PowerShell window and run:

```powershell
wsl --install
wsl --update
wsl --set-default-version 2
```

If you want to pick a distro explicitly:

```powershell
wsl --install -d Ubuntu
```

Check what is installed:

```powershell
wsl -l -v
```

If WSL asks you to reboot, do that before continuing.

## 2. Open The Linux Distro

Start your distro from the Start menu or with:

```powershell
wsl
```

Confirm you are inside Linux:

```bash
uname -a
cat /etc/os-release
```

## 3. Prepare The Linux Environment

Inside the WSL distro, install the packages Yeast needs:

```bash
sudo apt update
sudo apt install -y qemu-kvm qemu-utils genisoimage openssh-client curl
```

If your distro does not have `apt`, use the package manager for that distro.

## 4. Install Yeast

Use the same install script as Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

Or install a specific release:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | YEAST_VERSION=v1.1.2 bash
```

## 5. Verify The Beta Path

Run:

```bash
yeast version
yeast doctor
```

Expected outcome:

- `yeast version` prints the installed version
- `yeast doctor` may still report blockers if WSL on your machine cannot provide the virtualization Yeast needs

If `yeast doctor` complains about `/dev/kvm`, QEMU/KVM, or host virtualization access, this WSL setup is not ready for Yeast yet.

## 6. What To Expect

WSL beta can be inconsistent because Yeast still expects Linux virtualization behavior.

You may see:

- slower startup
- missing hardware virtualization
- image or networking differences from native Linux
- commands that work on Linux but fail on this beta path

For anything important, use native Linux instead.

## Next Step

If the beta path works on your machine, continue with the [Quickstart](quickstart.md).
