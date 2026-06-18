#!/usr/bin/env bash
# Lab 28 validation
set -euo pipefail
SSH_PORT=2248; SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 28: Kubernetes Networking, Config, And Storage ==="; echo ""
check_ssh "k3s running"           "sudo systemctl is-active k3s"                                     "active"
check_ssh "ingress resource"      "kubectl get ingress --no-headers 2>/dev/null | wc -l"             "[1-9]"
check_ssh "configmap exists"      "kubectl get configmap app-config --no-headers 2>/dev/null"        "app-config"
check_ssh "secret exists"         "kubectl get secret app-secret --no-headers 2>/dev/null"           "app-secret"
check_ssh "pvc exists"            "kubectl get pvc --no-headers 2>/dev/null | wc -l"                 "[1-9]"
check_ssh "pvc bound"             "kubectl get pvc --no-headers 2>/dev/null | grep -c Bound"         "[1-9]"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 28 complete."
