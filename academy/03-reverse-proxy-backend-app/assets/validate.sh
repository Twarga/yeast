#!/usr/bin/env bash
# Lab 03 validation — run from the lab folder: bash assets/validate.sh

set -euo pipefail

PROXY_SSH_PORT=2203
BACKEND_SSH_PORT=2204
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
    echo "        expected to find: $expected"
    echo "        got:              $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo ""
echo "=== Lab 03: Reverse Proxy To Backend App ==="
echo ""

echo "Proxy VM"
check_ssh "proxy hostname"        "$PROXY_SSH_PORT"   "hostname"                          "proxy"
check_ssh "nginx active"          "$PROXY_SSH_PORT"   "sudo systemctl is-active nginx"    "active"
check_ssh "nginx listening on 80" "$PROXY_SSH_PORT"   "sudo ss -tlnp | grep ':80'"        "nginx"

echo ""
echo "Backend VM"
check_ssh "backend hostname"            "$BACKEND_SSH_PORT"  "hostname"                    "backend"
check_ssh "python app listening :8000"  "$BACKEND_SSH_PORT"  "sudo ss -tlnp | grep ':8000'" "python"

echo ""
echo "Proxy → Backend connectivity"
check_ssh "proxy can reach backend:8000" "$PROXY_SSH_PORT" "curl -s http://192.168.10.20:8000" "Hello"

echo ""
echo "End-to-end HTTP"
check_ssh "proxy returns 200"             "$PROXY_SSH_PORT" "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1" "200"
check_ssh "response comes from backend"   "$PROXY_SSH_PORT" "curl -s http://127.0.0.1"                                 "Hello"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Re-read the relevant section in lab.md and fix before continuing."
  exit 1
fi

echo "All checks passed. Lab 03 complete."
