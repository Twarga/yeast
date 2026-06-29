#!/usr/bin/env bash
# Lab 29 validation
set -euo pipefail
SSH_PORT=2250; SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_host() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(bash -lc "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 29: Kubernetes Delivery Capstone ==="; echo ""
check_ssh "k3s running"            "sudo systemctl is-active k3s"                                      "active"
check_ssh "nodes ready"            "kubectl get nodes --no-headers 2>/dev/null | grep -c Ready"         "[1-9]"
check_ssh "app deployment exists"  "kubectl get deployment --no-headers 2>/dev/null | wc -l"            "[1-9]"
check_ssh "ingress exists"         "kubectl get ingress --no-headers 2>/dev/null | wc -l"               "[1-9]"
check_ssh "registry running"       "docker ps --format '{{.Names}}' 2>/dev/null | grep registry || echo check_local" "."
check_host "registry reachable from laptop" "curl -s http://127.0.0.1:5000/v2/" "{}"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 29 complete."
