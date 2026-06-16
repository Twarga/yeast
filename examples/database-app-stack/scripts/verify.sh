#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo "  Database + App Stack Verification"
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
echo "1. Database"
check "PostgreSQL running" yeast exec db -- systemctl is-active postgresql
check "DB accepts connections" yeast exec db -- sudo -u postgres pg_isready -h localhost -p 5432
check "todo database exists" yeast exec db -- sudo -u postgres psql -d todo -c "SELECT 1"

echo ""
echo "2. Application"
check "Node app running" yeast exec app -- systemctl is-active todoapp
check "App HTTP 200" yeast exec app -- curl -sf http://localhost:3000/

echo ""
echo "3. API Endpoints"
RESULT=$(yeast exec app -- curl -sf http://localhost:3000/todos | jq '. | length')
if [ "$RESULT" -ge 3 ]; then
  echo "  [GET /todos returns items] ... OK ($RESULT items)"
else
  echo "  [GET /todos returns items] ... FAIL (got $RESULT items)"
  FAIL=1
fi

echo ""
echo "4. Cross-VM Reachability"
check "app pings db" yeast exec app -- ping -c 1 -W 3 192.168.2.50
check "db pings app" yeast exec db -- ping -c 1 -W 3 192.168.2.10

echo ""
echo "5. Data Persistence Check"
yeast exec app -- curl -sf -X POST http://localhost:3000/todos -H "Content-Type: application/json" -d '{"title":"test from verify"}' > /dev/null
echo -n "  [New todo created] ... "
if yeast exec app -- curl -sf http://localhost:3000/todos | jq -e 'map(select(.title == "test from verify")) | length > 0' > /dev/null 2>&1; then
  echo "OK"
else
  echo "FAIL"
  FAIL=1
fi

echo ""
echo "=========================================="
if [ "$FAIL" -eq 0 ]; then
  echo "  All checks passed."
else
  echo "  Some checks failed."
  exit 1
fi
echo "=========================================="
