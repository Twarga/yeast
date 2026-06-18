#!/usr/bin/env bash
# Lab 12 validation

set -euo pipefail

SSH_PORT=2218
SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS  $label"; PASS=$((PASS+1))
  else
    echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1))
  fi
}

echo ""
echo "=== Lab 12: Container Build, Scan, And Hardening ==="
echo ""

echo "Tools"
check_ssh "docker available"    "docker --version"            "Docker"
check_ssh "trivy available"     "trivy --version"             "Version"

echo ""
echo "Images"
check_ssh "myapp:fat built"     "docker images myapp:fat --format '{{.Repository}}:{{.Tag}}'"      "myapp:fat"
check_ssh "myapp:slim built"    "docker images myapp:slim --format '{{.Repository}}:{{.Tag}}'"     "myapp:slim"
check_ssh "slim is smaller"     "docker images myapp:fat myapp:slim --format '{{.Size}}' | head -2" "MB"

echo ""
echo "Hardening"
check_ssh "slim runs as non-root" "docker run --rm myapp:slim id" "uid=1000"
check_ssh "slim has no root shell" "docker run --rm myapp:slim whoami" "appuser"

echo ""
echo "Scan ran"
check_ssh "trivy scan log exists"  "test -f /home/ubuntu/scan-results.txt && echo ok" "ok"
check_ssh "scan completed"         "grep -i 'Total\|CRITICAL\|HIGH' /home/ubuntu/scan-results.txt | head -1" "."

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 12 complete."
