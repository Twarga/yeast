#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo "  GitOps / CI Lab Verification"
echo "=========================================="

FAIL=0

check() {
  local name="$1"
  shift
  echo -n "  [$name] ... "
  if "$@" > /dev/null 2>&1; then
    echo "OK"
  else
    echo "FAIL"
    FAIL=1
  fi
}

echo ""
echo "1. Gitea Git Server"
check "Gitea UI" yeast exec gitea -- curl -sf http://localhost:3000
check "Gitea API up" yeast exec gitea -- curl -sf http://localhost:3000/api/v1/node/info

echo ""
echo "2. Docker Registry"
check "Registry UI" yeast exec registry -- curl -sf http://localhost:5000/v2/

echo ""
echo "3. Runner"
check "Runner Docker" yeast exec runner -- docker ps | grep -q drone
check "Runner reaches Gitea" yeast exec runner -- curl -sf http://192.168.2.50:3000

echo ""
echo "4. Cross-VM"
check "runner pings gitea" yeast exec runner -- ping -c 1 -W 3 192.168.2.50
check "gitea pings registry" yeast exec gitea -- ping -c 1 -W 3 192.168.2.12
check "registry pings runner" yeast exec registry -- ping -c 1 -W 3 192.168.2.11

echo ""
echo "=========================================="
if [ "$FAIL" -eq 0 ]; then
  echo "  All checks passed."
else
  echo "  Some checks failed."
fi
echo "=========================================="
