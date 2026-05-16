# Changelog

All notable changes to this project are documented in this file.

This format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Installer hardening is planned before the v0.1.0 release artifact is built.
- Final Linux/KVM host checklist is still pending before tagging.

## [0.1.0] - Draft

### Summary

Yeast v0.1.0 is the first clean lifecycle release of the v2 rebuild. It provides a Linux-first QEMU/KVM workflow for creating a Yeast project, pulling trusted Ubuntu cloud images, starting a VM, waiting for SSH readiness, checking status, connecting over SSH, stopping the VM, and destroying project runtime files.

This release is intentionally narrow. It proves the local VM engine before provisioning, snapshots, private networking, LabsBackery integration, MCP, or cloud workers are added.

### Added

- Clean v2 CLI entrypoint with thin Cobra commands.
- Project initialization through `yeast init`.
- Project identity metadata in `.yeast/project.json`.
- Project-safe runtime paths under `~/.yeast/projects/<project-id>/`.
- Config model, loader, validation, defaults, and normalization for `yeast.yaml`.
- State model, atomic state save, state lock, and process reconciliation.
- Trusted image manifest for:
  - `ubuntu-22.04`
  - `ubuntu-24.04`
- Shared image cache under `~/.yeast/cache/images`.
- Image download with SHA-256 verification and partial-file cleanup.
- `yeast pull --list` and `yeast pull <image>`.
- QEMU runtime boundary with a QEMU/KVM implementation.
- `qemu-img` qcow2 overlay disk preparation.
- Deterministic QEMU command construction.
- QEMU process start, inspect, stop, and destroy behavior.
- Cloud-init SSH key discovery.
- Cloud-init `user-data` and `meta-data` rendering.
- Seed ISO creation using `genisoimage` or `mkisofs`.
- TCP readiness checks for SSH.
- `yeast doctor` host readiness checks.
- `yeast up` lifecycle workflow.
- `yeast status` with state reconciliation.
- `yeast ssh [instance]`.
- `yeast down`.
- `yeast destroy`.
- Human output renderer with Lip Gloss styling.
- Stable JSON success/error envelopes for core command output.
- CLI-level JSON contract tests.
- Fast unit test suite script at `scripts/test-fast.sh`.
- Fake-runtime app workflow test for `up -> status -> down -> status -> destroy -> status`.
- `examples/ubuntu-basic`.
- v0.1 docs:
  - quickstart
  - installation
  - config reference
  - troubleshooting
  - known limitations
  - architecture overview
- README banner and polished v0.1 README.

### Changed

- Rebuilt Yeast from the old prototype into a clean v2 architecture.
- Moved business workflows into `internal/app`.
- Kept CLI commands as wrappers around app workflows.
- Separated human terminal output from JSON output.
- Raised source-build requirement to Go 1.25+ because current Charm Lip Gloss v2 requires it.
- Replaced prototype implementation details with documented v2 architecture and tasks.

### Fixed

- State reconciliation no longer blindly trusts stale state when a VM process dies.
- QEMU process inspection treats Linux zombie processes as stopped.
- Image download failures clean up partial files.
- Destroy removes tracked runtime data without deleting the shared image cache.
- Repeated `yeast init` refuses to overwrite an existing project.

### Security

- Base image downloads are verified with SHA-256 before being moved into the cache.
- Project state updates use locking to avoid concurrent write corruption.
- Raw `user_data` behavior is documented because it replaces generated cloud-init.

### Known Limitations

- Linux host only.
- QEMU/KVM only.
- No VirtualBox backend.
- No Windows or macOS host support.
- No provisioning workflow yet beyond first-boot cloud-init bootstrap.
- No snapshots or restore yet.
- No private VM-to-VM lab networking yet.
- No guest `exec`, `copy`, `logs`, or `inspect` commands yet.
- No template system yet.
- No daemon, web API, LabsBackery contract, Yeast MCP, or Twarga Cloud worker mode yet.
- Final real-host manual lifecycle checklist is still required before tagging.
- Installer hardening is still required before release artifact build.

### Verification

Automated checks currently passing:

- `go test ./... -count=1`
- `go build ./...`
- `git diff --check`
- fast unit suite through `bash scripts/test-fast.sh`

Manual release verification still required:

- install host packages on a real Linux/KVM host
- run `yeast doctor`
- run `yeast init`
- run `yeast pull ubuntu-24.04`
- run `yeast up`
- run `yeast status`
- run `yeast ssh`
- run `yeast down`
- run `yeast up` again
- run `yeast destroy`

## Release Note Entry Format

For each changelog entry, include:

- What changed in one concise sentence.
- Affected area when relevant.
- Upgrade or migration note when behavior or contract changes.

## Versioning Policy

Yeast uses `MAJOR.MINOR.PATCH` versioning:

- `MAJOR`: breaking changes to stable CLI behavior, config schema, or output contracts.
- `MINOR`: backward-compatible features.
- `PATCH`: backward-compatible bug fixes and docs-only corrections.

Tags must use `vX.Y.Z`.
