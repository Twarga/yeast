#!/usr/bin/env bash
# Lab 08 validation — run from the lab folder: bash assets/validate.sh

set -euo pipefail

SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check_ssh() {
  local label="$1"
  local port="$2"
  local cmd="$3"
  local expected="$4"

  actual=$(ssh -p "$port" \
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

echo ""
echo "=== Lab 08: Ansible Multi-VM Web Cluster ==="
echo ""

echo "Load balancer"
check_ssh "lb hostname"       2211  "hostname"                        "lb"
check_ssh "nginx active"      2211  "sudo systemctl is-active nginx"  "active"
check_ssh "nginx proxying"    2211  "sudo grep -r 'upstream' /etc/nginx/sites-enabled/" "upstream"

echo ""
echo "Web nodes"
check_ssh "web1 hostname"     2212  "hostname"                        "web1"
check_ssh "web1 nginx active" 2212  "sudo systemctl is-active nginx"  "active"
check_ssh "web2 hostname"     2213  "hostname"                        "web2"
check_ssh "web2 nginx active" 2213  "sudo systemctl is-active nginx"  "active"
check_ssh "web3 hostname"     2214  "hostname"                        "web3"
check_ssh "web3 nginx active" 2214  "sudo systemctl is-active nginx"  "active"

echo ""
echo "Load distribution"
RESPONSES=""
for i in 1 2 3 4 5 6; do
  r=$(ssh -p 2211 -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "curl -s http://127.0.0.1" 2>/dev/null || echo "FAIL")
  RESPONSES="$RESPONSES $r"
done

if echo "$RESPONSES" | grep -q "web1" && echo "$RESPONSES" | grep -q "web2" && echo "$RESPONSES" | grep -q "web3"; then
  echo "  PASS  requests distributed across web1, web2, web3"
  PASS=$((PASS + 1))
else
  echo "  FAIL  not all web nodes received requests"
  echo "        responses: $RESPONSES"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Check playbook output and lab.md."
  exit 1
fi

echo "All checks passed. Lab 08 complete."
