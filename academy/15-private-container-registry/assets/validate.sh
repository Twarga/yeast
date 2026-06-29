#!/usr/bin/env bash
# Lab 15 validation

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

echo ""; echo "=== Lab 15: Private Container Registry ==="; echo ""
echo "Registry VM"
check_ssh "registry hostname" 2221 "hostname" "registry"
check_ssh "registry container running" 2221 "docker ps --format '{{.Names}}'" "registry"
check_ssh "registry listening :5000" 2221 "sudo ss -tlnp | grep ':5000'" "docker"

echo ""; echo "Deployer VM"
check_ssh "deployer hostname" 2222 "hostname" "deployer"
check_ssh "app running from registry" 2222 "docker ps --format '{{.Image}}'" "192.168.40.10:5000"

echo ""; echo "Image promotion"
check_ssh "registry has myapp images" 2221 "curl -s http://localhost:5000/v2/myapp/tags/list" "tags"
check_host "registry reachable from laptop" "curl -s http://127.0.0.1:5000/v2/_catalog" "repositories"
check_host "deployed app reachable from laptop" "curl -s http://127.0.0.1:8080" "version"

echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 15 complete."
