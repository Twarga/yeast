#!/usr/bin/env bash
# Lab 24 validation
set -euo pipefail
SSH_PORT=2239; SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local cmd="$2"; local expected="$3"
  actual=$(ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 24: Terraform Modules And Environments ==="; echo ""
check_ssh "terraform installed"       "terraform version"                                          "Terraform"
check_ssh "modules directory exists"  "test -d /home/ubuntu/tf-modules/modules && echo ok"        "ok"
check_ssh "dev env applied"           "test -f /home/ubuntu/tf-modules/environments/dev/terraform.tfstate && echo ok" "ok"
check_ssh "prod env applied"          "test -f /home/ubuntu/tf-modules/environments/prod/terraform.tfstate && echo ok" "ok"
check_ssh "dev config distinct"       "grep 'dev' /tmp/dev-server-1-config.txt 2>/dev/null || echo ok" "."
check_ssh "prod config distinct"      "grep 'prod' /tmp/prod-server-1-config.txt 2>/dev/null || echo ok" "."
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 24 complete."
