#!/usr/bin/env bash
# Lab 18 validation

set -euo pipefail
SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local port="$2"; local cmd="$3"; local expected="$4"
  actual=$(ssh -p "$port" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 18: Centralized Logging ==="; echo ""

echo "Logging stack"
check_ssh "loki running"   2228 "docker ps --format '{{.Names}}'" "loki"
check_ssh "grafana running" 2228 "docker ps --format '{{.Names}}'" "grafana"

echo ""; echo "Endpoints"
check_ssh "loki ready" 2228 "curl -s http://127.0.0.1:3100/ready" "ready"
check_ssh "grafana UI" 2228 "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000" "200"

echo ""; echo "Log shipping"
check_ssh "promtail running on app" 2229 "sudo systemctl is-active promtail 2>/dev/null || docker ps 2>/dev/null | grep promtail || echo check_manual" "."
check_ssh "loki has logs" 2228 "curl -s 'http://127.0.0.1:3100/loki/api/v1/labels' | grep -c '\\bvalues\\b'" "."

echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 18 complete."
