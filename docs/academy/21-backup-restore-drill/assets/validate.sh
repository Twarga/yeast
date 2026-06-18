#!/usr/bin/env bash
# Lab 21 validation
set -euo pipefail
SSH_USER=ubuntu; SSH_HOST=127.0.0.1; PASS=0; FAIL=0

check_ssh() {
  local label="$1"; local port="$2"; local cmd="$3"; local expected="$4"
  actual=$(ssh -p "$port" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")
  if echo "$actual" | grep -q "$expected"; then echo "  PASS  $label"; PASS=$((PASS+1))
  else echo "  FAIL  $label"; echo "        expected: $expected"; echo "        got: $actual"; FAIL=$((FAIL+1)); fi
}

echo ""; echo "=== Lab 21: Backup And Restore Drill ==="; echo ""
check_ssh "primary postgres active" 2234 "sudo systemctl is-active postgresql" "active"
check_ssh "backup script exists"    2234 "test -f /home/ubuntu/backup.sh && echo ok" "ok"
check_ssh "backup ran"              2234 "ls /home/ubuntu/backups/*.sql.gz 2>/dev/null | head -1" ".sql.gz"
check_ssh "restore tested"          2235 "sudo -u postgres psql -d appdb_restored -c 'SELECT COUNT(*) FROM items;' 2>/dev/null" "count"
echo ""; echo "Results: ${PASS} passed, ${FAIL} failed"
[ "$FAIL" -gt 0 ] && echo "Some checks failed." && exit 1
echo "All checks passed. Lab 21 complete."
