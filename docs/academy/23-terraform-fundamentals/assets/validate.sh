#!/usr/bin/env bash
# Lab 23 validation
set -euo pipefail
SSH_PORT=2238; SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 23: Terraform Fundamentals ==="; echo ""
check_ssh "terraform installed"   "terraform version"                         "Terraform"
check_ssh "project dir exists"    "test -d /home/ubuntu/tf-lab && echo ok"    "ok"
check_ssh "main.tf exists"        "test -f /home/ubuntu/tf-lab/main.tf && echo ok" "ok"
check_ssh "terraform.tfstate exists" "test -f /home/ubuntu/tf-lab/terraform.tfstate && echo ok" "ok"
check_ssh "state has resources"   "grep -c '\"type\"' /home/ubuntu/tf-lab/terraform.tfstate 2>/dev/null || echo 0" "[1-9]"
check_ssh "outputs.tf exists"     "test -f /home/ubuntu/tf-lab/outputs.tf && echo ok" "ok"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 23 complete."
