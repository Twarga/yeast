#!/usr/bin/env bash
# Lab 05 validation — run from the lab folder: bash assets/validate.sh
# This lab is about troubleshooting, so the validation checks that you
# can correctly identify the state of things, not that everything is passing.

set -euo pipefail

SSH_PORT=2208
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
echo "=== Lab 05: Linux Troubleshooting Drill ==="
echo "Checking that the server has been recovered to a clean state..."
echo ""

check_ssh "hostname is drill"       "hostname"                              "drill"
check_ssh "nginx is active"         "sudo systemctl is-active nginx"        "active"
check_ssh "nginx is listening :80"  "sudo ss -tlnp | grep ':80'"            "nginx"
check_ssh "postgresql is active"    "sudo systemctl is-active postgresql"   "active"
check_ssh "ufw is active"           "sudo ufw status | head -1"             "active"
check_ssh "disk is not full"        "df -h / | awk 'NR==2{print \$5}'"      "[0-8][0-9]%"
check_ssh "no world-writable /etc"  "find /etc -maxdepth 1 -perm -o+w -type f 2>/dev/null | wc -l" "^0$"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some services are still in a broken state. Keep troubleshooting."
  exit 1
fi

echo "All checks passed. Server recovered. Lab 05 complete."
