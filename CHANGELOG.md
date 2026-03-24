# Changelog

All notable changes to this project are documented in this file.

This format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- VM log rotation with archived file naming (`vm.<timestamp>.<unixnano>.log`) on each new start.
- VM log retention policy (default keep last 5 archives per instance).
- Explicit networking modes behind flags: `user`, `private`, `bridge`.
- Go-focused static analysis baseline in CI: `golangci-lint`, `gosec`, and `go vet`.

### Changed
- Runtime now enforces per-instance VM log pruning at startup.
- VM startup/restart now validates networking flags before provisioning.
- CI now runs curated static analysis via `scripts/static-analysis.sh`.

### Fixed
- _None yet_

### Security
- _None yet_

### Performance
- _None yet_

### Docs
- Added `CONTRIBUTING.md` with contribution flow, coding/testing standards, release notes format, and versioning policy.
- Added this `CHANGELOG.md` template and release-note categories.
- Added `docs/LOGGING.md` with log format/levels/fields, naming, and retention policy.
- Added `docs/RUNBOOK_VM_FAILURES.md` with debugging steps and triage matrix.
- Added `docs/NETWORKING_MODES.md` with networking design, tradeoffs, and safe defaults.
- Added `docs/SECURITY_STATIC_ANALYSIS.md` with severity policy and suppression standards.

## Release Note Entry Format

For each changelog entry, include:

- What changed in one concise sentence.
- Affected area (`cmd/*`, `pkg/*`, docs, CI, etc.) when relevant.
- Upgrade or migration note when behavior/contract changes.

## Versioning Policy

Yeast uses `MAJOR.MINOR.PATCH` versioning:

- `MAJOR`: breaking changes (CLI behavior, config schema, output contracts).
- `MINOR`: backward-compatible features.
- `PATCH`: backward-compatible bug fixes and docs-only corrections.

Tags must use: `vX.Y.Z`.
