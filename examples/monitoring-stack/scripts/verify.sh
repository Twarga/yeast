#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo "  Monitoring Stack Verification"
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
echo "1. Monitor Stack"
check "Prometheus UI" yeast exec monitor -- curl -sf http://localhost:9090
check "Grafana UI" yeast exec monitor -- curl -sf http://localhost:3000

echo ""
echo "2. Node Exporters"
check "web exporter" yeast exec web -- systemctl is-active prometheus-node-exporter
check "db exporter" yeast exec db -- systemctl is-active prometheus-node-exporter
check "cache exporter" yeast exec cache -- systemctl is-active prometheus-node-exporter

echo ""
echo "3. Prometheus Scrapes"
for target in 192.168.2.11:9100 192.168.2.12:9100 192.168.2.13:9100; do
  echo -n "  [Scrape $target] ... "
  if yeast exec monitor -- curl -sf "http://localhost:9090/api/v1/query?query=up{instance=\"$target\"}" | grep -q '"value":\["1"'; then
    echo "OK"
  else
    echo "FAIL"
    FAIL=1
  fi
done

echo ""
echo "4. Services"
check "PostgreSQL running" yeast exec db -- systemctl is-active postgresql
check "Redis running" yeast exec cache -- systemctl is-active redis-server

echo ""
echo "=========================================="
if [ "$FAIL" -eq 0 ]; then
  echo "  All checks passed."
else
  echo "  Some checks failed."
  exit 1
fi
echo "=========================================="
