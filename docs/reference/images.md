# Images

Yeast uses a trusted image manifest and a shared local image cache.

List supported images:

```bash
yeast pull --list
```

List locally cached images:

```bash
yeast pull --cached
```

## Auto-Download Images

These images are downloaded automatically by `yeast up` when missing. You can also pre-cache one with `yeast pull <image>`.

| Image | Category | Cloud-Init | Approx Size | Description |
|---|---|---:|---:|---|
| `debian-12` | General Purpose | yes | ~400MB | Debian 12 Bookworm, stable/minimal |
| `debian-13` | General Purpose | yes | ~400MB | Debian 13 Trixie, newer packages |
| `ubuntu-22.04` | General Purpose | yes | ~500MB | Ubuntu 22.04 LTS, legacy LTS |
| `ubuntu-24.04` | General Purpose | yes | ~600MB | Ubuntu 24.04 LTS, default choice |
| `fedora-41` | DevOps & Cloud | yes | ~500MB | Fedora 41, balanced stable/bleeding-edge |
| `fedora-42` | DevOps & Cloud | yes | ~500MB | Fedora 42, newer tooling/kernels |
| `alma-9` | Enterprise | yes | ~1GB | AlmaLinux 9, RHEL-compatible |
| `centos-stream-9` | Enterprise | yes | ~800MB | CentOS Stream 9, upstream RHEL |
| `rocky-9` | Enterprise | yes | ~1GB | Rocky Linux 9, RHEL-compatible |

## Manual Images

Manual images are listed by Yeast, but Yeast does not download them automatically. Running `yeast pull <image>` prints setup instructions.

| Image | Category | Cloud-Init | Approx Size | Notes |
|---|---|---:|---:|---|
| `amazon-linux-2023` | Enterprise | yes | ~1.4GB | Manual QCOW2 download |
| `opensuse-leap-15.6` | Enterprise | no | ~1GB | Manual setup or conversion |
| `kali-2026.1` | Security | no | ~3.6GB | Manual QEMU image, default `kali/kali` credentials |
| `parrot-security-7.1` | Security | no | ~11.7GB | Manual QCOW2 setup |
| `alpine-3.21` | Minimal | no | ~50MB | Manual ISO/QCOW2 setup |
| `arch-linux` | Niche | no | ~800MB | Manual arch-boxes or custom QCOW2 |
| `nixos-24.11` | Niche | no | ~1GB | Manual NixOS generator/download setup |

## Cache Location

Images are cached under:

```text
~/.yeast/cache/images/
```

Each cached image is stored as:

```text
~/.yeast/cache/images/<image>/image.qcow2
```

## Clean Cached Images

Remove one cached image:

```bash
yeast images clean ubuntu-24.04
```

Preview cleanup:

```bash
yeast images clean ubuntu-24.04 --dry-run
```

Remove all cached images:

```bash
yeast images clean --all
```
