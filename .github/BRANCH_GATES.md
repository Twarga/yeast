# Branch Gates

This repository enforces two GitHub Actions checks for protected branches:

- `CI / Test, Vet, Build`
- `Branch Gates / Policy Checks`

## Required Branch Protection Settings

Configure branch protection for `main` (and `master` if used) with:

1. Require a pull request before merging.
2. Require status checks to pass before merging.
3. Select the required checks:
   - `CI / Test, Vet, Build`
   - `Branch Gates / Policy Checks`
4. Restrict direct pushes to protected branches.

## PR Gate Rules (`Branch Gates / Policy Checks`)

- PR must not be in draft state.
- Base branch must be `main` or `master`.
- Head branch must not be `main` or `master`.
- Head branch naming must match:

```text
<type>/<slug>
```

Allowed `<type>` values:

```text
feature|fix|hotfix|chore|docs|refactor|test|perf|ci
```

Exception:

- Bot branches prefixed with `dependabot/` or `renovate/` are allowed.
