# Yeast v0.7.0 Release Notes

## Summary

Yeast v0.7.0 adds the first template system to the local VM engine. Users can now list built-in starters and initialize a normal editable project from a built-in or local template directory.

This release keeps templates intentionally simple. Templates are project starters only: they copy `yeast.yaml` and declared project files into a new project during `yeast init`. They do not add a remote registry, update workflow, variable engine, hidden provisioning bundle system, or LabsBackery-specific package format.

## Added

- `yeast init --list-templates`
- `yeast init --template <name-or-path>`
- Built-in template catalog and metadata model.
- Local template metadata loading.
- Template materialization service with no-overwrite behavior.
- Built-in templates:
  - `ubuntu-basic`
  - `caddy-single-vm`
  - `two-vm-lab`
- Human and JSON output for template listing.
- Smoke coverage for template listing and built-in template initialization.
- Negative smoke coverage for missing template names returning `not_found`.

## Changed

- `yeast init` can now create a project from a template while preserving normal `.yeast/project.json` project metadata.
- The positive real-host smoke path now starts the Caddy VM workflow from `yeast init --template caddy-single-vm`.
- README, quickstart, config reference, manual test docs, and embedded terminal docs now describe templates as shipped `v0.7` behavior.
- Built-in template READMEs now document the real `yeast init --template <name>` workflow instead of old manual copy instructions.

## Verification

- `go test ./... -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- `TEST_MODE=negative ./scripts/manual-smoke.sh /tmp/yeast-v07-doc-smoke`
- Positive real-host smoke with:
  - template listing
  - `caddy-single-vm` template init
  - Caddy provisioning
  - guest control
  - snapshot/restore
  - two-VM private networking

## Known Limitations

- Templates are project starters only.
- Generated files are copied once and then become normal editable project files.
- No remote template downloads.
- No template registry/search/update workflow.
- No complex template variable engine.
- No hidden provisioning bundle system outside normal `yeast.yaml`.
- No LabsBackery-specific lab package format yet.
