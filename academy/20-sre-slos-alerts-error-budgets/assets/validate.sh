#!/usr/bin/env bash
# Lab 20 validation

set -euo pipefail
SSH_PORT=2233; SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 20: SRE — SLOs, Alerts, Error Budgets ==="; echo ""
check_ssh "prometheus running"    "docker ps --format '{{.Names}}'"  "prometheus"
check_ssh "alertmanager running"  "docker ps --format '{{.Names}}'"  "alertmanager"
check_ssh "grafana running"       "docker ps --format '{{.Names}}'"  "grafana"
check_ssh "prometheus healthy"  "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:9090/-/healthy" "200"
check_ssh "alertmanager healthy" "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:9093/-/healthy" "200"
check_ssh "slo rules file exists" "test -f /home/ubuntu/monitoring/rules/slo.yml && echo ok" "ok"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 20 complete."
