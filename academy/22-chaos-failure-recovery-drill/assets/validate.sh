#!/usr/bin/env bash
# Lab 22 validation
set -euo pipefail
SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local port="$2"; local cmd="$3"; local expected="$4"
  actual=$(ssh -p "$port" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 22: Chaos And Failure Recovery ==="; echo ""
check_ssh "proxy nginx active"   2236 "sudo systemctl is-active nginx"        "active"
check_ssh "app postgres active"  2237 "sudo systemctl is-active postgresql"   "active"
check_ssh "app service active"   2237 "sudo ss -tlnp | grep ':8000'"          "python"
check_ssh "site serves 200"      2236 "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1" "200"
check_ssh "runbook exists"       2237 "test -f /home/ubuntu/runbook.md && echo ok" "ok"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 22 complete."
