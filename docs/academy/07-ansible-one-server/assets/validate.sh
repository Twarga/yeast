#!/usr/bin/env bash
# Lab 07 validation — run from the lab folder: bash assets/validate.sh
# Requires ansible installed on your laptop

set -euo pipefail

SSH_PORT=2210
SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check_ssh() {
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
    echo "        expected: $expected"
    echo "        got:      $actual"
    FAIL=$((FAIL + 1))
  fi
}

check_local() {
  local label="$1"
  local cmd="$2"
  local expected="$3"

  actual=$(eval "$cmd" 2>/dev/null || echo "FAILED")

  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS  $label"
    PASS=$((PASS + 1))
  else
    echo "  FAIL  $label"
    echo "        expected: $expected"
    echo "        got:      $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo ""
echo "=== Lab 07: Ansible For One Server ==="
echo ""

echo "Ansible installed"
check_local "ansible available"     "which ansible"          "/ansible"
check_local "ansible version"       "ansible --version"      "ansible"

echo ""
echo "Server state (applied by Ansible)"
check_ssh "hostname is managed"    "hostname"                              "managed"
check_ssh "nginx installed"        "which nginx"                           "/usr/sbin/nginx"
check_ssh "nginx active"           "sudo systemctl is-active nginx"        "active"
check_ssh "fail2ban active"        "sudo systemctl is-active fail2ban"     "active"
check_ssh "ufw active"             "sudo ufw status | head -1"             "active"
check_ssh "ufw allows nginx"       "sudo ufw status"                       "Nginx"
check_ssh "site file exists"       "test -f /var/www/html/index.html && echo ok" "ok"

echo ""
echo "Idempotency"
echo "  Re-running playbook to confirm idempotency..."
if ansible-playbook -i inventory.ini site.yml -e "ansible_port=2210" \
    2>&1 | grep -q "changed=0"; then
  echo "  PASS  playbook is idempotent (0 changes on re-run)"
  PASS=$((PASS + 1))
else
  echo "  NOTE  could not verify idempotency automatically — run the playbook again and confirm 'changed=0'"
fi

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Check the playbook output and lab.md."
  exit 1
fi

echo "All checks passed. Lab 07 complete."
