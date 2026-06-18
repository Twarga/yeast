#!/usr/bin/env bash
# Lab 09 validation — run from the lab folder: bash assets/validate.sh

set -euo pipefail

SSH_PORT=2215
SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check_ssh() {
  local label="$1"
  local cmd="$2"
  local expected="$3"

  actual=$(ssh -p "$SSH_PORT" \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=5 \
    -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")

  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS  $label"
    PASS=$((PASS + 1))
  else
    echo "  FAIL  $label"
    echo "        expected: $expected"
    echo "        got:      $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo ""
echo "=== Lab 09: Secrets And Configuration Management ==="
echo ""

echo "No hardcoded secrets in app code"
if ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" \
    "grep -r 'password\s*=\s*[\"'\''][^$]' /home/ubuntu/app/ 2>/dev/null" | grep -qi "password"; then
  echo "  FAIL  hardcoded password found in app files"
  FAIL=$((FAIL + 1))
else
  echo "  PASS  no hardcoded passwords in app files"
  PASS=$((PASS + 1))
fi

echo ""
echo "Environment file"
check_ssh ".env file exists"        "test -f /home/ubuntu/app/.env && echo ok"    "ok"
check_ssh ".env not world-readable" "stat -c '%a' /home/ubuntu/app/.env"          "600"

echo ""
echo "App"
check_ssh "app service active"      "sudo systemctl is-active app"                "active"
check_ssh "app reads DB_HOST"       "sudo systemctl show app -p Environment"      "DB_"

echo ""
echo "Vault"
if command -v ansible-vault &>/dev/null; then
  echo "  PASS  ansible-vault available"
  PASS=$((PASS + 1))
else
  echo "  SKIP  ansible-vault not installed (install ansible)"
fi

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Re-read lab.md secrets section."
  exit 1
fi

echo "All checks passed. Lab 09 complete."
