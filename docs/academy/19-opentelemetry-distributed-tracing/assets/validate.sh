#!/usr/bin/env bash
# Lab 19 validation

set -euo pipefail
SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local port="$2"; local cmd="$3"; local expected="$4"
  actual=$(ssh -p "$port" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 19: OpenTelemetry Distributed Tracing ==="; echo ""
check_ssh "jaeger running"    2230 "docker ps --format '{{.Names}}'" "jaeger"
check_ssh "jaeger UI up"     2230 "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:16686" "200"
check_ssh "svc-a running"    2231 "sudo ss -tlnp | grep ':8001'" "python"
check_ssh "svc-b running"    2232 "sudo ss -tlnp | grep ':8002'" "python"
check_ssh "trace ends e2e"   2231 "curl -s http://127.0.0.1:8001 | grep trace" "trace"
check_ssh "jaeger has traces" 2230 "curl -s 'http://127.0.0.1:16686/api/services' | grep -c 'svc'" "."
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 19 complete."
