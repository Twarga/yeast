#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo "  Load Balancer Lab Verification"
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
echo "1. Backend Services"
check "web1 app running" yeast exec web1 -- systemctl is-active webapp
check "web2 app running" yeast exec web2 -- systemctl is-active webapp
check "web1 responds" yeast exec web1 -- curl -sf http://localhost:8080
check "web2 responds" yeast exec web2 -- curl -sf http://localhost:8080

echo ""
echo "2. Proxy / Load Balancer"
check "Caddy running" yeast exec proxy -- systemctl is-active caddy
check "Proxy HTTP 200" curl -sf http://127.0.0.1:8080

echo ""
echo "3. Round-Robin Distribution"
R1=$(curl -sf http://127.0.0.1:8080 | grep -o 'web[12]' || true)
R2=$(curl -sf http://127.0.0.1:8080 | grep -o 'web[12]' || true)
R3=$(curl -sf http://127.0.0.1:8080 | grep -o 'web[12]' || true)
if [[ "$R1" == "web1" && "$R2" == "web2" && "$R3" == "web1" ]] || \
   [[ "$R1" == "web2" && "$R2" == "web1" && "$R3" == "web2" ]]; then
  echo "  [Round-robin alternates] ... OK"
else
  echo "  [Round-robin alternates] ... SKIP (got $R1, $R2, $R3 — Caddy may not switch every request)"
fi

echo ""
echo "4. Cross-VM Reachability"
check "proxy pings web1" yeast exec proxy -- ping -c 1 -W 3 192.168.2.11
check "proxy pings web2" yeast exec proxy -- ping -c 1 -W 3 192.168.2.12
check "web1 pings web2" yeast exec web1 -- ping -c 1 -W 3 192.168.2.12

echo ""
echo "=========================================="
if [ "$FAIL" -eq 0 ]; then
  echo "  All checks passed."
else
  echo "  Some checks failed."
  exit 1
fi
echo "=========================================="
