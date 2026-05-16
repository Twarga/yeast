# Yeast Troubleshooting

Start with:

```bash
yeast doctor
```

Fix all blockers before debugging `yeast up`.

## `qemu-system-x86_64` Missing

Install QEMU system packages.

Ubuntu / Debian:

```bash
sudo apt install qemu-system-x86
```

Arch Linux:

```bash
sudo pacman -S qemu-base
```

## `qemu-img` Missing

Install QEMU image tools.

Ubuntu / Debian:

```bash
sudo apt install qemu-utils
```

Arch Linux:

```bash
sudo pacman -S qemu-base
```

## ISO Builder Missing

Yeast needs `genisoimage` or `mkisofs` to create the cloud-init seed ISO.

Ubuntu / Debian:

```bash
sudo apt install genisoimage
```

Arch Linux:

```bash
sudo pacman -S cdrtools
```

## `/dev/kvm` Missing Or Inaccessible

Check:

```bash
ls -l /dev/kvm
```

If the device does not exist, enable virtualization in BIOS/UEFI and make sure KVM modules are available.

If the device exists but access is denied, add your user to the correct group. Common case:

```bash
sudo usermod -aG kvm $USER
```

Then log out and back in.

## SSH Public Key Missing

Generate a supported key:

```bash
ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
```

Yeast checks:

- `~/.ssh/id_ed25519.pub`
- `~/.ssh/id_rsa.pub`

## Image Not Found In Cache

If `yeast up` says an image is missing, pull it first:

```bash
yeast pull ubuntu-24.04
```

List supported images:

```bash
yeast pull --list
```

## Checksum Mismatch

A checksum mismatch means the downloaded image does not match Yeast's trusted manifest.

Do not bypass this manually.

Try:

```bash
rm -rf ~/.yeast/cache/images/ubuntu-24.04
yeast pull ubuntu-24.04
```

If it still fails, the upstream image or Yeast manifest may need an update.

## `yeast init` Says Project Already Initialized

`yeast init` refuses to overwrite an existing project.

It checks for:

- `yeast.yaml`
- `.yeast/project.json`

Use a new empty folder, or remove the files only if you are intentionally resetting the project metadata.

## `yeast status` Says Stopped After A VM Was Killed

This is expected.

Yeast reconciles state against the real process. If QEMU was killed outside Yeast, status is corrected to `stopped`.

## State Lock Timeout

Yeast uses a project state lock to prevent two commands from modifying the same project at once.

If a command is still running, wait for it to finish.

If a previous process crashed, Yeast can recover stale locks after the stale timeout.

## SSH Does Not Connect

Check status:

```bash
yeast status
```

If the instance is not running, start it:

```bash
yeast up
```

If it is running but SSH fails:

- wait a little longer for cloud-init
- check `vm.log` in the project runtime directory
- verify the SSH key used by Yeast exists
- verify the forwarded SSH port in `yeast status`

## Where To Find Logs

Runtime files live under:

```text
~/.yeast/projects/<project-id>/instances/<instance-name>/
```

Useful files:

- `vm.log`
- `user-data`
- `meta-data`
- `seed.iso`
- `disk.qcow2`
