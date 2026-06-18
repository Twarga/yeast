#!/usr/bin/env bash
# Lab 27 validation
set -euo pipefail
SSH_PORT=2241; SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 27: Kubernetes Foundations With k3s ==="; echo ""
check_ssh "k3s running"          "sudo systemctl is-active k3s"                                       "active"
check_ssh "nodes ready"          "kubectl get nodes --no-headers 2>/dev/null | grep -c Ready"          "[1-9]"
check_ssh "nginx deployment"     "kubectl get deployment nginx-demo --no-headers 2>/dev/null"          "nginx-demo"
check_ssh "nginx pods running"   "kubectl get pods -l app=nginx-demo --no-headers 2>/dev/null | grep -c Running" "[1-9]"
check_ssh "nginx service exists" "kubectl get service nginx-demo --no-headers 2>/dev/null"             "nginx-demo"
check_ssh "3 workers joined"     "kubectl get nodes --no-headers 2>/dev/null | wc -l"                  "[23456789]"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 27 complete."
