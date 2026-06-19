#!/usr/bin/env bash
# Lab 25 validation
set -euo pipefail
SSH_PORT=2240; SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 25: GitOps With Argo CD ==="; echo ""
check_ssh "k3s running"        "sudo systemctl is-active k3s 2>/dev/null || kubectl get nodes 2>/dev/null | grep Ready" "."
check_ssh "argocd namespace"   "kubectl get namespace argocd 2>/dev/null" "argocd"
check_ssh "argocd server pod"  "kubectl get pods -n argocd 2>/dev/null | grep argocd-server" "Running"
check_ssh "argocd UI"          "curl -sk https://127.0.0.1:8080 | head -5" "."
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 25 complete."
