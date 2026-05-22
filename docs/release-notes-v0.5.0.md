# Yeast v0.5.0 Release Notes

## Summary

Yeast v0.5.0 adds the first private lab networking slice to the local VM engine.

It now supports:

- one project-level private lab network
- one static IPv4 per attached instance
- separate management SSH and guest-to-guest lab traffic
- `LAB IP` visibility in `yeast status`

This is still the narrow first pass. It is enough for the first attacker/target workflows, but not yet enough for bridge mode, DHCP, or multi-network lab topologies.

## Added

- Project-level network config in `yeast.yaml`.
- Per-instance network attachments with:
  - `name`
  - `ipv4`
- Config validation for:
  - one private network in `v0.5`
  - valid CIDR
  - valid static IPv4
  - duplicate lab IPv4 rejection
  - unknown network reference rejection
- Runtime network model for:
  - management SSH forwarding
  - one private lab network
  - per-instance lab IPv4
- QEMU command support for a second lab NIC/backend.
- Cloud-init `network-config` generation for the lab NIC.
- `LAB IP` in status output and JSON.
- `examples/two-vm-lab`.
- `docs/release-notes-v0.5.0.md`.

## Changed

- `yeast up` now plans and persists first-pass lab network state when configured.
- Quickstart, limitations, and README now document the first attacker/target networking flow.
- The smoke suite now covers both:
  - the single-VM provisioning/reset path
  - the two-VM private lab path

## Verification

- `go test ./... -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- `TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64`

## Known Limitations

- only one project-level private lab network
- no bridge mode
- no DHCP
- no multiple private networks
- no guest `exec`, `copy`, `logs`, or `inspect` yet
- deeper guest-to-guest validation is still manual from inside the VMs
