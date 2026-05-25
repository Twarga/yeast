# Yeast v0.6.0 Release Notes

## Summary

Yeast v0.6.0 adds the first guest-control surface to the local VM engine. It can now run one-shot commands inside a guest, copy files in both directions, expose the VM runtime log, and return a structured per-instance inspect view.

This release is still intentionally narrow. Guest control is SSH-backed only, one instance at a time, with no log streaming, no recursive directory sync, and no service health-check framework yet.

## Added

- `yeast exec [instance] -- <command...>`
- `yeast copy <instance> --to-guest <source> <destination>`
- `yeast copy <instance> --from-guest <source> <destination>`
- `yeast logs <instance> [--tail N]`
- `yeast inspect <instance>`
- Shared guest-control result models for:
  - command execution
  - copy operations
  - inspect output
  - log reads
- SSH transport download support for guest -> host copy
- Human and JSON output coverage for all guest-control commands
- Guest-control smoke coverage in `scripts/manual-smoke.sh`

## Changed

- Quickstart, README, and manual test docs now describe the guest-control surface as shipped behavior.
- Known limitations now document the actual `v0.6` constraints instead of treating guest control as future work.

## Verification

- `go test ./internal/app -run 'Test(Inspect|Logs|TailLogContent|GuestControlResultShapes|Exec|Copy|ShellQuoteCommand)' -count=1`
- `go test ./cmd/yeast ./internal/output -count=1`
- `go test ./internal/provision/ssh -count=1`
- `git diff --check`
- `bash -n scripts/manual-smoke.sh`
- full real-host smoke:
  - `INSTANCE_SSH_PORT=3045 RESTORE_SSH_PORT=3046 ATTACKER_SSH_PORT=3105 TARGET_SSH_PORT=3106 TEST_MODE=full ./scripts/manual-smoke.sh /tmp/yeast-v060-smoke`

## Known Limitations

- SSH-backed only
- one selected instance per command
- no recursive directory copy/sync
- no log streaming/follow mode
- no service health checks yet
- no multi-instance fanout execution
