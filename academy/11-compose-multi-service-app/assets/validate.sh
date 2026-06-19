#!/usr/bin/env bash
# Lab 11 validation

set -euo pipefail

SSH_PORT=2217
SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check_ssh() {
  local label="$1"
  local cmd="$2"
  local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS  $label"; PASS=$((PASS+1))
  else
    echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1))
  fi
}

echo ""
echo "=== Lab 11: Compose Multi-Service App ==="
echo ""

echo "Compose stack"
check_ssh "app container running"  "docker compose -f /home/ubuntu/app/compose.yaml ps --format '{{.Name}}'"   "app"
check_ssh "db container running"   "docker compose -f /home/ubuntu/app/compose.yaml ps --format '{{.Name}}'"   "db"
check_ssh "proxy container running" "docker compose -f /home/ubuntu/app/compose.yaml ps --format '{{.Name}}'"  "proxy"

echo ""
echo "End-to-end"
check_ssh "HTTP returns 200"       "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080"  "200"
check_ssh "items endpoint works"   "curl -s http://127.0.0.1:8080/items"                           "items"

echo ""
echo "Volumes"
check_ssh "db volume persists" "docker volume ls" "pgdata"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 11 complete."
