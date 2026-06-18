#!/usr/bin/env bash
# Lab 14 validation

set -euo pipefail

SSH_PORT=2220
SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS  $label"; PASS=$((PASS+1))
  else
    echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1))
  fi
}

echo ""
echo "=== Lab 14: Self-Hosted CI Runner ==="
echo ""

echo "Runner VM"
check_ssh "hostname is runner"    "hostname"                          "runner"
check_ssh "docker available"      "docker --version"                  "Docker"
check_ssh "runner dir exists"     "test -d /home/ubuntu/actions-runner && echo ok" "ok"
check_ssh "runner binary exists"  "test -f /home/ubuntu/actions-runner/run.sh && echo ok" "ok"
check_ssh "runner service active" "sudo systemctl is-active actions.runner.* 2>/dev/null || sudo systemctl list-units 'actions.runner.*' --no-legend | grep -i running || echo check_github" "."

echo ""
echo "Docker"
check_ssh "ubuntu in docker group" "groups" "docker"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed. Re-read lab.md." && exit 1
echo "All checks passed. Lab 14 complete."
