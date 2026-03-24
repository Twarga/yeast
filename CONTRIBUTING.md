# Contributing to Yeast

Thanks for contributing to Yeast.

This document defines:

- contribution flow
- coding/testing standards
- release notes format
- versioning policy

## 1) Contribution Flow

1. Create or pick an issue describing the change.
2. Branch from `main` using this pattern:
   - `feature/<slug>`
   - `fix/<slug>`
   - `hotfix/<slug>`
   - `chore/<slug>`
   - `docs/<slug>`
   - `refactor/<slug>`
   - `test/<slug>`
   - `perf/<slug>`
   - `ci/<slug>`
3. Implement the change with tests and docs updates.
4. Run local checks before pushing:
   - `go test ./... -count=1`
   - `./scripts/static-analysis.sh artifacts`
   - `go build ./cmd/yeast`
5. Open a PR to `main` (not draft when ready).
6. Ensure required checks pass:
   - `CI / Test, Vet, Build`
   - `Branch Gates / Policy Checks`
7. After approval and green checks, squash or merge per maintainer guidance.

## 2) Coding Standards

- Go version: use the version declared in `go.mod`.
- Format all Go code with `gofmt`.
- Keep package boundaries clean (`cmd`, `pkg/*`).
- Return contextual errors (`fmt.Errorf("...: %w", err)`).
- Preserve CLI UX contracts:
  - default human-readable output
  - machine-readable `--json` contract compatibility
- Avoid introducing secrets, credentials, or unsafe defaults.
- Keep changes focused and minimal; avoid unrelated refactors.

## 3) Testing Standards

- Every behavior change should include tests.
- Prefer:
  - unit tests for package logic
  - integration tests for CLI lifecycle behavior
- Static/security checks are mandatory and blocking:
  - `go vet`
  - `golangci-lint` (from `.golangci.yml`)
  - `gosec` (severity/confidence threshold: `medium`)
- If output contracts change, update:
  - `docs/OUTPUT_CONTRACTS.md`
  - integration tests covering JSON schema behavior
- If performance-related behavior changes, run:
  - `scripts/benchmark.sh`
  - update `benchmarks/latest.json` and README metrics when appropriate

## 4) Commit Guidance

Recommended commit style:

- `feat: ...`
- `fix: ...`
- `docs: ...`
- `refactor: ...`
- `test: ...`
- `perf: ...`
- `ci: ...`
- `chore: ...`

Example:

```text
feat: add restart JSON contract counters
```

## 4.1) Suppression Standards

- Use narrow, justified suppressions only when a finding is a false positive or risk-accepted.
- `gosec` format:
  - `// #nosec Gxxx -- reason`
- `golangci-lint` format:
  - `//nolint:<linter> // reason`
- Never add blanket or reasonless suppressions.
- See `docs/SECURITY_STATIC_ANALYSIS.md` for severity policy and PR requirements.

## 5) Release Notes Format

Yeast release notes follow Keep a Changelog style categories:

- `Added`
- `Changed`
- `Fixed`
- `Security`
- `Performance`
- `Docs`

For each entry include:

- concise summary
- impacted commands/files (if relevant)
- migration notes (if behavior changed)

## 6) Versioning Policy

Yeast uses Semantic Versioning (`MAJOR.MINOR.PATCH`):

- `MAJOR`: breaking changes in CLI behavior, config schema, or output contracts
- `MINOR`: backward-compatible features
- `PATCH`: backward-compatible fixes and docs-only corrections

Tag format:

- `vX.Y.Z` (for example `v0.4.2`)

When a change is breaking:

1. document it explicitly in `CHANGELOG.md` under `Changed`
2. include upgrade/migration guidance in the same entry

## 7) PR Checklist

Before requesting review, confirm:

- [ ] Branch name follows policy
- [ ] Tests updated/added
- [ ] `go test`, `go vet`, `go build` pass
- [ ] Docs updated (`README`, `docs/*`, changelog entry if needed)
- [ ] No sensitive data or local machine artifacts committed
