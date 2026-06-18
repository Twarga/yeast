#!/usr/bin/env bash
# Lab 30 validation
set -euo pipefail
SSH_PORT=2252; SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 30: AI-Assisted DevOps And Local LLM Ops ==="; echo ""
check_ssh "ollama running"       "sudo systemctl is-active ollama"           "active"
check_ssh "ollama API healthy"   "curl -s http://127.0.0.1:11434/api/tags"  "models"
check_ssh "model pulled"         "ollama list 2>/dev/null | head -5"         "."
check_ssh "assist script exists" "test -f /home/ubuntu/ops-assist.sh && echo ok" "ok"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 30 complete."
