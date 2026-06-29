# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v1.1.5] - 2026-06-29

### Added
- **Docker-style service port forwarding**: `yeast.yaml` now supports `instances[].ports`, including simple mappings like `["8080:80"]` and object syntax for named forwards
- **Copy-ready service URLs**: `yeast up` and `yeast status` now show forwarded host URLs so labs and demos are easier to use from the laptop

### Fixed
- **`yeast destroy` local cleanup**: full destroy now actually removes `.yeast/` and `yeast.yaml` after confirmation instead of leaving stale project files behind
- **Host port collision checks**: Yeast now rejects conflicting SSH/service host port bindings before boot

### Changed
- **Academy access flow**: key labs and shared access docs now teach Yeast-native forwarded ports before manual SSH tunnels
- **Current stable release**: release-facing docs and install examples now target `v1.1.5`

[v1.1.5]: https://github.com/Twarga/yeast/releases/tag/v1.1.5

## [v1.1.2] - 2026-06-17

### Added
- **Daily update notice on `yeast up`**: human `yeast up` runs a quiet update check at most once per day and shows a small banner when a newer Yeast release is available
- **Update-check cache**: Yeast stores the latest release check in `~/.yeast/cache/update-check.json` to avoid repeated GitHub requests

### Changed
- **Automation-safe output**: update notices are hidden for `--json`, `--events`, and `--quiet` so scripts keep stable output
- **Current stable release**: install examples and release smoke docs now target `v1.1.2`

[v1.1.2]: https://github.com/Twarga/yeast/releases/tag/v1.1.2

## [v1.1.1] - 2026-06-17

### Fixed
- **Release tarball packaging**: release artifacts now contain the `yeast` binary expected by the installer and updater
- **`yeast update` reliability**: the updater and release artifacts now agree on the binary name and checksum path
- **Stale VM cleanup**: `yeast down` and `yeast destroy` recover hidden running VMs even when state is stale
- **SSH command noise**: SSH-backed commands no longer print the repeated known-host warning on every guest control call
- **Release docs and installer defaults**: the current stable install and smoke-check docs now point at `v1.1.1`

### Changed
- **Current stable release**: Yeast docs and install examples now target `v1.1.1`
- **Release smoke validation**: fresh-install and update validation docs were refreshed for the patch release

[v1.1.1]: https://github.com/Twarga/yeast/releases/tag/v1.1.1

## [v1.1.0] - 2026-06-14

### Added
- **Auto-download images on `yeast up`**: Yeast now automatically downloads missing VM images instead of requiring manual `yeast pull`
- **22 pre-configured images** across 6 categories:
  - **General**: Ubuntu 24.04, Ubuntu 22.04, Debian 12, Debian 13
  - **DevOps**: Fedora 42, Fedora 41
  - **Enterprise**: Rocky Linux 9, AlmaLinux 9, CentOS Stream 9
  - **Minimal**: Alpine 3.21 (manual download)
  - **Security**: Kali 2026.1, Parrot Security 7.1 (manual download)
  - **Niche**: Arch Linux, NixOS 24.11, openSUSE Leap 15.6 (manual download)
- **Image categories & metadata**: Each image has description, cloud-init support, size, and manual installation instructions
- **`yeast images` command**: List, categorize, and manage cached images
- **`yeast images clean`**: Remove cached images with `--all` and `--dry-run` options
- **`yeast update` command**: Self-update to latest release with SHA256 verification and atomic binary replacement
- **`yeast completion`**: Shell completion for bash, zsh, fish
- **Sequential VM boot**: Multiple VMs now boot sequentially with resource profiling
- **Fingerprint-based reprovisioning**: Detects configuration drift and reprovisions only when needed
- **`--no-provision` flag**: Skip provisioning step on `yeast up`
- **Management host address configurable** via `yeast.yaml`
- **QEMU process profiling**: CPU/memory metrics during VM lifecycle

### Changed
- **`yeast up` behavior**: Now auto-downloads missing images instead of erroring
- **SSH transport refactor**: Improved connection reliability and performance
- **QEMU runtime improvements**: Better disk handling, process management, QMP integration
- **Install script**: Updated to Go 1.22.5, fixed version constants, added Go tarball checksum verification
- **Project structure**: Cleaner internal packages, new output/progress/spinner packages

### Fixed
- **Stale v0.5 error strings** in config validation (now v1.0 consistent)
- **WSL2 detection**: Better nested virtualization warnings
- **KVM permission handling**: Auto-loads modules, fixes device permissions
- **genisoimage compatibility**: Falls back to xorriso wrapper

### Security
- **SHA256 verification** for all downloaded images and Go toolchain
- **Atomic binary replacement** in `yeast update` prevents partial writes

---

## [v1.0.1] - 2026-05-20

### Added
- Initial VM orchestration with QEMU/KVM
- Cloud-init user-data and meta-data generation
- Project-based VM management with `yeast.yaml`
- SSH port forwarding and key injection
- Snapshot create/restore/list/delete
- Multi-VM private networking
- Template system for reusable configurations

### Changed
- Refactored internal architecture for v1.0 stability

---

## [v1.0.0] - 2026-04-15

### Added
- First stable release
- Core VM lifecycle: init, up, down, destroy, ssh, status, logs, inspect
- Cloud-init integration
- Basic provisioning (packages, files, shell)
- State management with locking

[v1.1.0]: https://github.com/Twarga/yeast/releases/tag/v1.1.0
[v1.0.1]: https://github.com/Twarga/yeast/releases/tag/v1.0.1
[v1.0.0]: https://github.com/Twarga/yeast/releases/tag/v1.0.0
