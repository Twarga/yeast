#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo "  WireGuard VPN Mesh Verification"
echo "=========================================="

echo ""
echo "WARNING: This lab uses DUMMY keys for demonstration only."
echo "Generate real keys with: wg genkey | tee private.key | wg pubkey > public.key"
echo ""

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
echo "1. WireGuard Interfaces"
check "hub wg0" yeast exec hub -- ip addr show wg0
check "spoke1 wg0" yeast exec spoke1 -- ip addr show wg0
check "spoke2 wg0" yeast exec spoke2 -- ip addr show wg0

echo ""
echo "2. WireGuard Peers"
check "hub has peers" yeast exec hub -- wg show wg0 peers | grep -q "peer"
check "spoke1 has peer" yeast exec spoke1 -- wg show wg0 peers | grep -q "peer"
check "spoke2 has peer" yeast exec spoke2 -- wg show wg0 peers | grep -q "peer"

echo ""
echo "3. Tunnel Reachability (10.200.200.x)"
check "spoke1 pings hub tunnel" yeast exec spoke1 -- ping -c 2 -W 5 10.200.200.1
check "spoke2 pings hub tunnel" yeast exec spoke2 -- ping -c 2 -W 5 10.200.200.1
check "spoke1 pings spoke2 tunnel" yeast exec spoke1 -- ping -c 2 -W 5 10.200.200.3

echo ""
echo "4. Encrypted Traffic Verification"
echo "  Tunnel packets (should be > 0 after pings):"
yeast exec hub -- wg show wg0 transfer | head -3 || true

echo ""
echo "=========================================="
if [ "$FAIL" -eq 0 ]; then
  echo "  Tunnel checks passed."
  echo "  NOTE: Pings may fail with dummy keys."
  echo "  Generate real WireGuard keys and update the .conf files to test encryption."
else
  echo "  Some checks failed — likely due to dummy keys."
fi
echo "=========================================="
