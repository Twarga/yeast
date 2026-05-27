# Yeast v1.0.0 Release Candidate Checklist

Status: passed for v1.0.0 release candidate validation.

Validation date: 2026-05-27

Validated candidate:

- Source commit: `8bb5f19`
- Release candidate binary: `dist/yeast-linux-amd64`
- Candidate version: `v1.0.0-rc1`
- Host: Linux KVM workstation

## Build Artifact

Result: passed.

Commands:

```bash
bash scripts/build-release.sh v1.0.0-rc1
./dist/yeast-linux-amd64 version
cd dist && sha256sum -c yeast-linux-amd64.sha256
```

Observed:

- `yeast version` returned `v1.0.0-rc1`
- release checksum verified successfully

## Automated Tests

Result: passed.

Command:

```bash
go test ./... -count=1
```

Observed:

- all Go packages passed

## Static Checks

Result: passed.

Commands:

```bash
bash -n install.sh scripts/manual-smoke.sh scripts/build-release.sh
git diff --check
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
go install github.com/securego/gosec/v2/cmd/gosec@v2.22.1
export PATH="$(go env GOPATH)/bin:$PATH"
./scripts/static-analysis.sh artifacts
```

Observed:

- shell syntax checks passed
- whitespace diff check passed
- `go vet` passed
- `golangci-lint` passed
- `gosec` passed with `Issues: 0`

## Installer Path

Result: passed with a non-system install harness.

The agent cannot provide the maintainer sudo password, so the installer was validated with:

- temporary user-writable install directory
- temporary user-writable Go install root
- local repository clone source
- temporary sudo shim that skips package database mutation while allowing non-root filesystem steps

Command shape:

```bash
PATH="$tmpbin:$PATH" \
YEAST_REPO_URL="file:///home/twarga/Projects/yeast" \
YEAST_REF=main \
YEAST_INSTALL_DIR=/tmp/yeast-installer-bin \
YEAST_GO_INSTALL_ROOT=/tmp/yeast-installer-go \
YEAST_KEEP_LOGS=1 \
bash install.sh

/tmp/yeast-installer-bin/yeast version
/tmp/yeast-installer-bin/yeast doctor
```

Observed:

- dependency verification reached and passed against the real host tools
- source clone passed
- CLI build passed
- binary install passed to `/tmp/yeast-installer-bin/yeast`
- installed binary verification passed
- user path setup passed
- SSH key check passed
- KVM access check passed
- post-install `yeast doctor` passed
- installed harness binary reported `0.0.0-dev` from `main`, as expected for an unreleased branch build

Before tagging v1.0.0, run the public installer once with a real sudo session after the tag exists:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | \
  YEAST_REF=v1.0.0 YEAST_EXPECTED_VERSION=v1.0.0 bash
```

## Full Real-Host Smoke

Result: passed.

Command:

```bash
TEST_MODE=full WORKDIR=/tmp/yeast-v1-full-smoke ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

Observed successful coverage:

- built-in template list
- built-in Caddy template initialization
- LabsBakery attacker/target package materialization
- Ubuntu image pull
- Caddy VM boot through QEMU/KVM
- provisioning status reaches `provisioned`
- direct SSH host/user checks
- Caddy service health check
- guest HTTP content check
- `yeast exec` JSON result
- `yeast copy --to-guest`
- `yeast copy --from-guest`
- `yeast inspect`
- `yeast logs`
- `yeast provision` rerun
- VM down/up lifecycle
- snapshot creation and listing
- guest break phase
- snapshot restore
- restored guest content check
- snapshot deletion
- destroy cleanup
- two-VM lab boot
- static lab IP reporting
- guest-side lab NIC checks
- guest-to-guest TCP reachability
- two-VM down/destroy cleanup
- negative JSON error cases

The full smoke script reported PASS for all listed checks.

## Negative Cases Covered

Result: passed.

The full smoke covered:

- status in uninitialized directory
- repeated init conflict
- unsupported image pull
- missing template
- corrupt project metadata
- state project mismatch
- missing `yeast.yaml`
- invalid `disk_size`
- invalid hostname
- invalid `ssh_port`
- duplicate requested `ssh_port`
- invalid network CIDR
- unknown network reference
- invalid network IPv4
- duplicate network IPv4
- missing provision source file

## Release Gate

The v1.0.0 candidate is ready for the public documentation refresh and release notes pass.

Remaining before final tag:

- refresh public README/docs for v1.0.0
- prepare v1.0.0 changelog and release notes
- build final `v1.0.0` artifact
- run the public installer command against the `v1.0.0` tag
- publish the GitHub release
