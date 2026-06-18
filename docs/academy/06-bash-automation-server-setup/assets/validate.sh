#!/usr/bin/env bash
# Lab 06 validation — run from the lab folder: bash assets/validate.sh

set -euo pipefail

SSH_PORT=2209
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
echo "=== Lab 06: Bash Automation For Server Setup ==="
echo ""

echo "Base"
check_ssh "hostname"       "hostname"                              "automate"
check_ssh "timezone UTC"   "timedatectl | grep 'Time zone'"        "UTC"

echo ""
echo "Script results"
check_ssh "nginx installed by script"      "which nginx"                             "/usr/sbin/nginx"
check_ssh "nginx active"                   "sudo systemctl is-active nginx"          "active"
check_ssh "fail2ban active"                "sudo systemctl is-active fail2ban"       "active"
check_ssh "ufw allows Nginx"               "sudo ufw status"                         "Nginx"
check_ssh "setup log written"              "test -f /var/log/server-setup.log && echo ok" "ok"
check_ssh "setup log records completion"   "grep -i 'complete\|done\|finished' /var/log/server-setup.log" "."
check_ssh "idempotent: nginx still active" "sudo systemctl is-active nginx"          "active"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Check the setup script output and server-setup.log."
  exit 1
fi

echo "All checks passed. Lab 06 complete."
