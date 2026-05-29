# Yeast v1.0.1 Release Notes

Status: Released

Release date: 2026-05-29

## Summary

Yeast v1.0.1 is a patch release from the post-v1 massive validation pass.

It fixes one race-detector issue in TCP readiness tests and upgrades `golang.org/x/net` to remove a reachable vulnerability reported by `govulncheck` through the CLI docs rendering path.

## Fixed

- Removed a package-level readiness test dial hook that caused `go test -race ./...` to report a data race.
- Added an explicit optional `ReadinessOptions.Dial` function for test injection instead of mutating shared package state.
- Upgraded `golang.org/x/net` from `v0.33.0` to `v0.38.0`.

## Security

`govulncheck` reported a reachable vulnerability before the dependency upgrade:

- `GO-2025-3595`
- module: `golang.org/x/net`
- found in: `v0.33.0`
- fixed in: `v0.38.0`
- reachable path: CLI docs rendering through Glamour's HTML tokenizer path

After the upgrade, `govulncheck ./...` reports:

```text
Your code is affected by 0 vulnerabilities.
```

## Verification

The validation pass included:

- `gofmt` check
- `go test ./... -count=1`
- `go test -race ./... -count=1`
- `go test ./... -coverprofile=/tmp/yeast-cover.out -count=1`
- `govulncheck ./...`
- `go vet`
- `golangci-lint`
- `gosec`
- shell syntax checks
- `git diff --check`
- release artifact build with version injection
- checksum verification
- full real-host KVM smoke test

The full smoke test covered:

- template list and materialization
- LabsBakery attacker/target package materialization
- Caddy VM provisioning
- guest control commands
- reprovisioning
- down/up lifecycle
- snapshot, restore, and delete
- two-VM private lab networking
- guest-to-guest TCP reachability
- negative JSON error cases

## Compatibility

This release does not intentionally change the v1 command, config, JSON, event, or LabsBakery integration contracts.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

Explicit v1.0.1 install:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | \
  YEAST_REF=v1.0.1 YEAST_EXPECTED_VERSION=v1.0.1 bash
```
