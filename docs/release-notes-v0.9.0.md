# Yeast v0.9.0 Release Notes

Status: Draft

## Summary

Yeast v0.9.0 turns the v0.8 automation surface into a LabsBakery-ready engine contract.

This release does not add the LabsBakery web product to Yeast. It defines the boundary LabsBakery can depend on: stable lifecycle commands, browser-terminal-friendly status data, stop and destroy event streams, a lab package convention, and one concrete attacker/target package example.

## Added

- `docs/labsbackery-integration-contract.md`.
- `docs/labsbackery-lab-package.md`.
- `examples/labsbackery-attacker-target-basic`.
- `user` in `status --json` instance records.
- `user` in `inspect --json` instance records.
- JSON Lines event streaming for `yeast down --json --events`.
- JSON Lines event streaming for `yeast destroy --json --events`.
- Tests for LabsBakery package materialization.
- Tests for down and destroy event streams.

## Changed

- README current scope now reflects the v0.9 LabsBakery-ready integration surface.
- The JSON contract is documented as the v0.9 draft contract while keeping `schema_version: "yeast.v1"`.
- LabsBakery integration docs use Yeast status/inspect user metadata for browser terminal connection planning.

## Verification

- `go test ./... -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- `go test ./internal/templates ./internal/app -count=1`

## Known Limitations

- Yeast does not include the LabsBakery web UI.
- No daemon or web API is included.
- No packaged `.lbz` import/export command exists yet.
- No project-wide atomic snapshot/reset helper exists yet.
- Event history and progress percentages are not persisted or guaranteed.
- Remote workers and Twarga Cloud remain out of scope.
