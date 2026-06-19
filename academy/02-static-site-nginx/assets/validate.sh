#!/usr/bin/env bash
# Lab 02 validation — run from the lab folder: bash assets/validate.sh

set -euo pipefail

SSH_PORT=2202
SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check() {
  local label="$1"
  local cmd="$2"
  local expected="$3"

  actual=$(ssh -p "$SSH_PORT" \
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
echo "=== Lab 02: Static Site With Nginx ==="
echo ""

echo "OS"
check "hostname is web"       "hostname"                          "web"
check "timezone is UTC"       "timedatectl | grep 'Time zone'"    "UTC"

echo ""
echo "Nginx"
check "nginx installed"       "which nginx"                                    "/usr/sbin/nginx"
check "nginx active"          "sudo systemctl is-active nginx"                 "active"
check "nginx enabled"         "sudo systemctl is-enabled nginx"                "enabled"
check "nginx listening on 80" "sudo ss -tlnp | grep ':80'"                     "nginx"

echo ""
echo "Firewall"
check "ufw active"            "sudo ufw status | head -1"   "Status: active"
check "port 80 allowed"       "sudo ufw status"             "Nginx"

echo ""
echo "HTTP response"
check "nginx serves HTTP inside VM" "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1" "200"
check "response contains site content" "curl -s http://127.0.0.1" "Welcome"

echo ""
echo "Logs"
check "access log exists"  "test -f /var/log/nginx/access.log && echo exists"   "exists"
check "error log exists"   "test -f /var/log/nginx/error.log && echo exists"    "exists"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Re-read the relevant section in lab.md and fix before continuing."
  exit 1
fi

echo "All checks passed. Lab 02 complete."
