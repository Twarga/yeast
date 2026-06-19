#!/usr/bin/env bash
# Lab 04 validation — run from the lab folder: bash assets/validate.sh

set -euo pipefail

SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check_ssh() {
  local label="$1"
  local port="$2"
  local cmd="$3"
  local expected="$4"

  actual=$(ssh -p "$port" \
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
echo "=== Lab 04: Database-Backed App ==="
echo ""

echo "Proxy"
check_ssh "proxy hostname"   2205  "hostname"                        "proxy"
check_ssh "nginx active"     2205  "sudo systemctl is-active nginx"  "active"

echo ""
echo "App"
check_ssh "app hostname"           2206  "hostname"                              "app"
check_ssh "app listening on 8000"  2206  "sudo ss -tlnp | grep ':8000'"          "python"

echo ""
echo "Database"
check_ssh "db hostname"        2207  "hostname"                              "db"
check_ssh "postgres active"    2207  "sudo systemctl is-active postgresql"   "active"
check_ssh "postgres port 5432" 2207  "sudo ss -tlnp | grep ':5432'"          "postgres"

echo ""
echo "Connectivity"
check_ssh "app can reach db" 2206  "pg_isready -h 192.168.20.30 -p 5432 2>/dev/null || echo fail" "accepting"

echo ""
echo "End-to-end"
check_ssh "HTTP returns data"      2205 "curl -s http://127.0.0.1/items"       "items"
check_ssh "POST creates a record"  2205 "curl -s -X POST http://127.0.0.1/items -H 'Content-Type: application/json' -d '{\"name\":\"test\"}'" "name"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Re-read the relevant section in lab.md."
  exit 1
fi

echo "All checks passed. Lab 04 complete."
