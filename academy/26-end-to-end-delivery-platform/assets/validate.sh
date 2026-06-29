#!/usr/bin/env bash
# Lab 26 validation
set -euo pipefail
SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_host() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(bash -lc "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

check_ssh() {
  local label="$1"; local port="$2"; local cmd="$3"; local expected="$4"
  actual=$(ssh -p "$port" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 26: End-To-End Delivery Platform ==="; echo ""
check_ssh "proxy nginx active"       2244 "sudo systemctl is-active nginx"                  "active"
check_ssh "app docker running"       2245 "docker ps --format '{{.Names}}'"                 "app"
check_ssh "db postgres active"       2246 "sudo systemctl is-active postgresql"              "active"
check_ssh "monitoring prometheus"    2247 "docker ps --format '{{.Names}}'"                 "prometheus"
check_ssh "monitoring grafana"       2247 "docker ps --format '{{.Names}}'"                 "grafana"
check_ssh "app responds 200"         2244 "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1" "200"
check_ssh "prometheus scraping"      2247 "curl -s 'http://127.0.0.1:9090/api/v1/targets' | grep -o 'up'" "up"
check_host "proxy reachable from laptop" "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:9090/items" "200"
check_host "prometheus reachable from laptop" "curl -s http://127.0.0.1:9190/-/healthy" "Prometheus"
check_host "grafana reachable from laptop" "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:9300" "200"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 26 complete."
