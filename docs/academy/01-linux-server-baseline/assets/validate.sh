#!/usr/bin/env bash
# Lab 01 validation — run from the lab folder: bash assets/validate.sh

set -euo pipefail

SSH_PORT=2201
SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check() {
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
    echo "        expected to find: $expected"
    echo "        got:              $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo ""
echo "=== Lab 01: Linux Server Baseline ==="
echo ""

echo "OS"
check "hostname is baseline"      "hostname"                              "baseline"
check "timezone is UTC"           "timedatectl | grep 'Time zone'"        "UTC"
check "OS is Ubuntu 22.04"        "grep PRETTY_NAME /etc/os-release"      "22.04"

echo ""
echo "Packages"
check "curl installed"   "which curl"   "/usr/bin/curl"
check "wget installed"   "which wget"   "/usr/bin/wget"
check "vim installed"    "which vim"    "/usr/bin/vim"
check "htop installed"   "which htop"   "/usr/bin/htop"

echo ""
echo "Firewall"
check "ufw active"          "sudo ufw status | head -1"   "Status: active"
check "OpenSSH allowed"     "sudo ufw status"             "OpenSSH"

echo ""
echo "Services"
check "fail2ban active"             "sudo systemctl is-active fail2ban"            "active"
check "unattended-upgrades active"  "sudo systemctl is-active unattended-upgrades" "active"

echo ""
echo "Sudo"
check "ubuntu can sudo"   "sudo whoami"   "root"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Re-read the relevant section in lab.md and fix before continuing."
  exit 1
fi

echo "All checks passed. Lab 01 complete."
