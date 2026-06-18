#!/usr/bin/env bash
# Lab 13 validation — this lab is GitHub-based, so most checks are manual
# Run from the lab folder: bash assets/validate.sh

set -euo pipefail

PASS=0
FAIL=0

check() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(eval "$cmd" 2>/dev/null || echo "FAILED")
  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS  $label"; PASS=$((PASS+1))
  else
    echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1))
  fi
}

echo ""
echo "=== Lab 13: CI With GitHub Actions ==="
echo ""

echo "Tools"
check "gh cli installed"  "gh --version"  "gh version"
check "git installed"     "git --version" "git version"

echo ""
echo "Workflow file"
if [ -f ".github/workflows/ci.yml" ]; then
  echo "  PASS  .github/workflows/ci.yml exists"
  PASS=$((PASS+1))
else
  echo "  FAIL  .github/workflows/ci.yml not found — create it in your project repo"
  FAIL=$((FAIL+1))
fi

if [ -f ".github/workflows/ci.yml" ]; then
  check "workflow has on push"     "grep 'on:' .github/workflows/ci.yml"     "on:"
  check "workflow has jobs"        "grep 'jobs:' .github/workflows/ci.yml"   "jobs:"
  check "workflow uses docker"     "grep -i 'docker' .github/workflows/ci.yml" "docker"
fi

echo ""
echo "GitHub status (requires gh auth)"
if gh auth status &>/dev/null; then
  REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || echo "")
  if [ -n "$REPO" ]; then
    LAST_RUN=$(gh run list --limit 1 --json status,conclusion,name 2>/dev/null || echo "")
    if echo "$LAST_RUN" | grep -q "completed"; then
      echo "  PASS  last workflow run completed"
      PASS=$((PASS+1))
    else
      echo "  NOTE  no completed runs found yet or gh not configured for this repo"
    fi
  fi
else
  echo "  SKIP  gh not authenticated — run: gh auth login"
fi

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed. Re-read lab.md." && exit 1
echo "All checks passed. Lab 13 complete."
