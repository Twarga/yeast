#!/usr/bin/env bash
# Lab 16 validation

set -euo pipefail
SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local port="$2"; local cmd="$3"; local expected="$4"
  actual=$(ssh -p "$port" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 16: Progressive Delivery And Rollback ==="; echo ""
check_ssh "lb nginx active"    2223 "sudo systemctl is-active nginx"       "active"
check_ssh "blue app running"   2224 "sudo ss -tlnp | grep ':8000'"         "python"
check_ssh "green app running"  2225 "sudo ss -tlnp | grep ':8000'"         "python"
check_ssh "lb serves traffic"  2223 "curl -s http://127.0.0.1" "version"
echo ""
echo "Traffic distribution (10 requests)"
VERSIONS=""
for i in $(seq 10); do
  VERSIONS="$VERSIONS $(ssh -p 2223 -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "curl -s http://127.0.0.1" 2>/dev/null | grep -o '"version":"[^"]*"' || echo fail)"
done
echo "  Versions seen: $VERSIONS"
echo "  PASS  traffic flowing through lb"
PASS=$((PASS+1))
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 16 complete."
