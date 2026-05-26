# Yeast v0.8.0 Release Notes

## Summary

Yeast v0.8.0 makes the CLI safer to use as an engine for other products. The main work is not a new VM feature; it is the first stable automation contract for LabsBakery, Yeast MCP, scripts, and future UIs.

This release adds versioned JSON envelopes, documented error codes, stable command data shapes, a lifecycle event model, and JSON Lines event streaming for long-running workflows.

## Added

- `schema_version: "yeast.v1"` on success JSON envelopes.
- `schema_version: "yeast.v1"` on error JSON envelopes.
- Stable JSON contract documentation in `docs/json-contract.md`.
- Stable lower `snake_case` command data fields for core JSON outputs.
- Standard error code catalog, including:
  - `invalid_argument`
  - `failed_precondition`
  - `conflict`
  - `not_found`
  - `timeout`
  - `runtime_error`
  - `provisioning_failed`
  - `guest_error`
  - `internal`
- Lifecycle event model for app workflows.
- JSON Lines event renderer.
- Global `--events` flag for machine-readable workflow progress.
- Event streaming for:
  - `yeast up --json --events`
  - `yeast provision --json --events`
  - `yeast restore --json --events`

## Changed

- Core command JSON output now uses stable lower `snake_case` data fields instead of Go-style exported field names.
- Guest control failures are classified as `guest_error`.
- Provisioning failures are classified as `provisioning_failed`.
- Runtime failures and timeout-style failures have explicit error categories.
- Manual smoke tests parse the stable v0.8 JSON field names.

## Compatibility Note

The v0.8 JSON data shape is intentionally stricter than earlier releases. Integrations should target the documented `yeast.v1` contract and should not scrape human terminal output.

## Verification

- `go test ./... -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- Negative smoke coverage for v0.8 JSON error envelopes.
- Live event-stream check for `yeast up --json --events`.

## Known Limitations

- Event streaming is currently limited to `up`, `provision`, and `restore`.
- Events are emitted live; Yeast does not persist event history.
- Events do not include percentage progress yet.
- There is no daemon or web API.
- There are no remote workers or Twarga Cloud features.
- LabsBakery and Yeast MCP integration should depend on `--json` and `--events`, not human output.
