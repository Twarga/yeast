# Changelog

All notable changes to this project are documented in this file.

This format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.9.0] - 2026-05-26

### Summary

Yeast v0.9.0 makes the engine LabsBakery-ready without turning Yeast into the LabsBakery web product. This release defines the integration contract, adds browser-terminal-friendly status metadata, extends lifecycle event streaming to stop and destroy workflows, documents a lab package convention, and ships the first attacker/target LabsBakery example package.

### Added

- LabsBakery integration contract in `docs/labsbackery-integration-contract.md`.
- LabsBakery lab package convention in `docs/labsbackery-lab-package.md`.
- First LabsBakery-ready attacker/target example in `examples/labsbackery-attacker-target-basic`.
- `user` field in `status --json` instance records.
- `user` field in `inspect --json` instance records.
- JSON Lines event streaming for `yeast down --json --events`.
- JSON Lines event streaming for `yeast destroy --json --events`.
- `docs/release-notes-v0.9.0.md`.

### Changed

- README current scope now reflects v0.9 LabsBakery-ready integration behavior.
- JSON contract docs now describe the v0.9 draft contract while preserving `schema_version: "yeast.v1"`.

### Known Limitations

- no LabsBakery web UI inside Yeast
- no daemon or web API
- no packaged `.lbz` import/export command
- no project-wide atomic snapshot/reset helper
- no remote workers or Twarga Cloud features

## [0.8.0] - 2026-05-26

### Summary

Yeast v0.8.0 makes the CLI safer to use as an engine for LabsBakery, Yeast MCP, scripts, and future UIs. This release adds the first stable automation contract: versioned JSON envelopes, documented error codes, stable command data fields, lifecycle events, and JSON Lines event streaming.

### Added

- `schema_version: "yeast.v1"` on success and error JSON envelopes.
- Stable JSON contract documentation in `docs/json-contract.md`.
- Stable lower `snake_case` data fields for core command JSON outputs.
- Standard error code catalog for automation.
- Lifecycle event model for app workflows.
- JSON Lines event renderer.
- Global `--events` flag.
- Event streaming for `up`, `provision`, and `restore` when used with `--json`.
- `docs/release-notes-v0.8.0.md`.

### Changed

- Core command JSON data fields now use stable lower `snake_case` names.
- Guest control failures are classified as `guest_error`.
- Provisioning failures are classified as `provisioning_failed`.
- Runtime and timeout failures have explicit error categories.
- Manual smoke tests now parse the stable v0.8 JSON field names.

### Known Limitations

- event streaming is limited to `up`, `provision`, and `restore`
- no persisted event history
- no progress percentage contract yet
- no daemon or web API
- no remote workers or Twarga Cloud features

## [0.7.0] - 2026-05-26

### Summary

Yeast v0.7.0 adds the first template system to the local VM engine. Users can now list built-in starters and initialize a normal editable project from a built-in or local template directory.

This release keeps templates intentionally simple. Templates are project starters only and do not add remote downloads, registries, complex variables, or LabsBackery-specific packaging.

### Added

- `yeast init --list-templates`.
- `yeast init --template <name-or-path>`.
- Built-in template catalog and metadata model.
- Local template metadata loading.
- Template materialization service with no-overwrite behavior.
- Built-in templates:
  - `ubuntu-basic`
  - `caddy-single-vm`
  - `two-vm-lab`
- Human and JSON output for template listing.
- Template docs across README, quickstart, config reference, manual test docs, and embedded terminal docs.
- `docs/release-notes-v0.7.0.md`.

### Changed

- `yeast init` can now initialize from a template while preserving normal project metadata behavior.
- The positive real-host smoke path now starts the Caddy VM workflow from `yeast init --template caddy-single-vm`.
- Built-in template READMEs now document the template workflow instead of old manual copy instructions.

### Verification

- `go test ./... -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- `TEST_MODE=negative ./scripts/manual-smoke.sh /tmp/yeast-v07-doc-smoke`
- Positive real-host smoke with template init, Caddy provisioning, guest control, snapshot/restore, and two-VM networking.

### Known Limitations

- templates are project starters only
- no remote template downloads
- no template registry/search/update workflow
- no complex template variable engine
- no hidden provisioning bundles outside normal `yeast.yaml`
- no LabsBackery-specific lab package format yet

## [0.6.0] - Draft

### Summary

Yeast v0.6.0 adds the first guest-control surface to the local VM engine. It can now run one-shot commands inside guests, copy files in both directions, expose VM runtime logs, and return a structured inspect view for one instance.

This release stays intentionally narrow. Guest control is SSH-backed only, one instance at a time, with no log streaming, no directory sync, and no service health checks yet.

### Added

- `yeast exec [instance] -- <command...>`.
- `yeast copy <instance> --to-guest <source> <destination>`.
- `yeast copy <instance> --from-guest <source> <destination>`.
- `yeast logs <instance> [--tail N]`.
- `yeast inspect <instance>`.
- Shared guest-control result models for exec, copy, inspect, and logs.
- SSH transport download support for guest -> host copy.
- Human and JSON output coverage for the guest-control commands.
- `docs/release-notes-v0.6.0.md`.

### Changed

- README, quickstart, and manual test docs now describe guest control as a shipped `v0.6` feature.
- Known limitations now document the real guest-control limits instead of treating the surface as future work.
- The full smoke suite now proves exec, copy, inspect, and logs on a real VM alongside the existing lifecycle, provisioning, snapshot, networking, and negative JSON checks.

### Verification

- `go test ./internal/app -run 'Test(Inspect|Logs|TailLogContent|GuestControlResultShapes|Exec|Copy|ShellQuoteCommand)' -count=1`
- `go test ./cmd/yeast ./internal/output -count=1`
- `go test ./internal/provision/ssh -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- `INSTANCE_SSH_PORT=3045 RESTORE_SSH_PORT=3046 ATTACKER_SSH_PORT=3105 TARGET_SSH_PORT=3106 TEST_MODE=full ./scripts/manual-smoke.sh /tmp/yeast-v060-smoke`

### Known Limitations

- SSH-backed only
- one selected instance per command
- no recursive directory copy/sync
- no log streaming/follow mode
- no service health checks yet

## [0.5.0] - 2026-05-22

### Summary

Yeast v0.5.0 adds the first narrow private lab networking slice to the local VM engine. It can now attach instances to one project-level private network, assign static lab IPv4 addresses, keep management SSH separate from lab traffic, and show `LAB IP` in status output.

This is intentionally the first step, not the final networking model. It is enough for simple attacker/target workflows, but not yet enough for bridge mode, DHCP, or multi-network lab topologies.

### Added

- Project-level private lab network config in `yeast.yaml`.
- Per-instance static lab network attachments with:
  - `name`
  - `ipv4`
- Validation for:
  - one project network in `v0.5`
  - valid CIDR
  - valid static IPv4
  - duplicate lab IPv4 rejection
  - unknown network reference rejection
- Runtime network model for management SSH and one private lab network.
- QEMU command support for a second private lab NIC/backend.
- Cloud-init `network-config` generation for the lab NIC.
- `LAB IP` in `yeast status` human and JSON output.
- `examples/two-vm-lab`.
- `docs/release-notes-v0.5.0.md`.

### Changed

- `yeast up` now plans and persists configured private lab network state.
- Quickstart and README now include the first two-VM attacker/target lab flow.
- Known limitations now document shipped `v0.5` networking support instead of treating it as future work.
- The manual smoke suite now covers both:
  - the single-VM provisioning/reset loop
  - the two-VM private lab loop

### Verification

- `go test ./... -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- `TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64`

### Known Limitations

- one project-level private lab network only
- no bridge mode
- no DHCP
- no multiple private networks
- no multi-network topology support
- guest-to-guest validation is still manual from inside the VMs

## [0.4.0] - 2026-05-22

### Summary

Yeast v0.4.0 adds the first narrow snapshot and reset workflow to the local VM engine. It can now snapshot a stopped VM, list stored snapshots, restore a stopped VM back to a saved disk state, and delete old snapshots.

This release intentionally keeps the model conservative. Snapshot create and restore are stopped-VM only, per-instance only, and disk-copy based. That is enough for the first single-VM reset workflows, but not yet enough for multi-machine lab reset.

### Added

- `yeast snapshot <instance> <name>`.
- `yeast snapshots <instance>`.
- `yeast restore <instance> <name>`.
- `yeast delete-snapshot <instance> <name>`.
- Snapshot metadata in project state:
  - `name`
  - `created_at`
  - `description`
  - `disk_path`
  - `source_disk_size`
- QEMU runtime snapshot file helpers for create, restore, and delete.
- Snapshot app workflows and tests.
- Snapshot CLI human output and JSON renderer coverage.
- Reset walkthrough in `examples/caddy-single-vm`.
- `docs/release-notes-v0.4.0.md`.

### Changed

- Quickstart now includes the first stopped-VM snapshot and restore loop.
- Known limitations now describe shipped snapshot support instead of treating snapshots as future work.
- The manual smoke script now covers snapshot create, list, break, restore, and delete on one real VM.

### Verification

- `go test ./... -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`

### Known Limitations

- Snapshot create is stopped-VM only.
- Restore is stopped-VM only.
- Snapshot scope is one instance at a time.
- Immediate reuse of the same host `ssh_port` after restore can still be flaky on some Linux hosts with QEMU user-mode forwarding.
- Project-wide snapshot/restore is not implemented yet.
- Live snapshots and live restore are not supported.
- Multi-VM atomic reset is not supported.
- Private VM-to-VM networking is still not implemented.

## [0.3.0] - 2026-05-18

### Summary

Yeast v0.3.0 adds the first real provisioning workflow to the local VM engine. It can now boot a guest, install packages, copy files, run shell commands, rerun provisioning against an existing VM, and track provisioning state and logs.

### Added

- Top-level and per-instance `provision` config.
- SSH package provisioner for Ubuntu/Debian guests.
- SSH file provisioner with source-path resolution relative to the project root.
- SSH shell provisioner.
- Automatic provisioning during `yeast up`.
- `yeast provision [instance]`.
- Provisioning status model: `not_started`, `running`, `provisioned`, `failed`.
- Per-instance `provision.log`.
- `examples/caddy-single-vm`.
- Updated smoke script coverage for lifecycle plus provisioning.
- `docs/release-notes-v0.3.0.md`.

### Changed

- `yeast up` now treats "ready" as "booted and provisioned" when a provisioning plan exists.
- State and status output now track provisioning progress and log paths.
- Quickstart, config reference, limitations, and README now describe provisioning as a shipped feature instead of a planned one.

### Fixed

- The Caddy example and provisioning smoke path now verify guest HTTP service from inside the guest rather than incorrectly assuming host HTTP reachability.
- CLI test isolation around the global JSON flag is tighter by removing unsafe parallel mutation.

### Verification

- `go test ./... -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- `TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64`

### Known Limitations

- Linux host only.
- QEMU/KVM only.
- Package provisioning currently assumes Ubuntu/Debian `apt-get`.
- No snapshots or restore yet.
- No private VM-to-VM networking yet.
- No guest `exec`, `copy`, `logs`, or `inspect` yet.
- No templates, LabsBackery contract, MCP contract, or cloud worker mode yet.

## [0.2.0] - 2026-05-17

### Summary

Yeast v0.2.0 strengthens the local VM engine after the initial lifecycle release. It adds explicit instance controls for `disk_size`, `hostname`, and `ssh_port`, tightens app-level JSON error contracts across core commands, and ships a broader smoke-test workflow that covers both happy-path and negative-path validation.

This release still does not add provisioning, snapshots, private networking, guest control, LabsBackery integration, MCP, or cloud workers. It focuses on making the base local VM workflow more controlled and more dependable.

### Added

- Explicit `disk_size` support in `yeast.yaml`.
- Explicit `hostname` support in `yeast.yaml`, defaulting to the instance name when omitted.
- Explicit `ssh_port` support in `yeast.yaml`.
- Validation for invalid `disk_size`, `hostname`, and `ssh_port` values before runtime start.
- Requested `ssh_port` collision detection across instances.
- Release notes for the v0.2.0 feature set.
- Expanded manual smoke script with:
  - full positive-path lifecycle validation
  - negative-path JSON error-contract validation

### Changed

- `yeast up` now passes configured hostnames through cloud-init user-data and meta-data.
- `yeast up` now preserves explicit `ssh_port` choices through start and restart flows.
- Overlay disk creation now consistently uses normalized `disk_size` values.
- The unsupported-image JSON contract for `yeast pull` now preserves `invalid_argument` instead of collapsing to `unknown`.

### Fixed

- App-level error normalization across:
  - `yeast up`
  - `yeast status`
  - `yeast ssh`
  - `yeast pull`
  - `yeast init`
  - `yeast down`
  - `yeast destroy`
- Project-state and config-path failures now return stable JSON error codes instead of raw uncategorized errors in the main command surface.

### Verification

Automated checks passing:

- `go test ./... -count=1`
- `go build ./...`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`

Real host smoke validation passing:

- `TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64`

Covered by the full smoke suite:

- `doctor`
- `init`
- `pull`
- `up`
- `status`
- `status --json`
- direct SSH validation
- `down`
- restart
- `destroy`
- invalid config and bad-state JSON contract checks

### Known Limitations

- Linux host only.
- QEMU/KVM only.
- No provisioning workflows yet.
- No snapshots or restore yet.
- No private VM-to-VM networking yet.
- No guest `exec`, `copy`, `logs`, or `inspect` yet.
- No templates, daemon API, LabsBackery integration contract, MCP contract, or cloud worker mode yet.

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
