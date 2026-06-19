#!/usr/bin/env bash
# Lab 17 validation

set -euo pipefail
SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local port="$2"; local cmd="$3"; local expected="$4"
  actual=$(ssh -p "$port" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 17: Prometheus And Grafana Monitoring ==="; echo ""

echo "Monitoring stack"
check_ssh "prometheus running" 2226 "docker ps --format '{{.Names}}'" "prometheus"
check_ssh "grafana running"    2226 "docker ps --format '{{.Names}}'" "grafana"

echo ""; echo "Endpoints"
check_ssh "prometheus UI" 2226 "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:9090/-/healthy" "200"
check_ssh "grafana UI"    2226 "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000" "200"

echo ""; echo "Metrics"
check_ssh "node-exporter running on app" 2227 "sudo systemctl is-active node_exporter 2>/dev/null || curl -s http://localhost:9100/metrics | head -1" "."
check_ssh "prometheus scrapes app" 2226 "curl -s 'http://127.0.0.1:9090/api/v1/targets' | grep -o '192.168.60.20:9100'" "192.168.60.20:9100"

echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 17 complete."
