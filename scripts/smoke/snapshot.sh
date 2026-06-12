#!/usr/bin/env bash
set -euo pipefail

SMOKE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SMOKE_DIR}/lib.sh"

BIN="${1:-$(smoke_resolve_binary)}"
ROOT="$(smoke_prepare_root)"
PASS=0
FAIL=0
TOTAL=0

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); printf '  ✓ PASS  %s\n' "$1"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); printf '  ✗ FAIL  %s\n' "$1"; printf '          → %s\n' "$2"; }

require_cmd() {
  local label="$1"
  shift
  local output
  if output=$("$@" 2>&1); then
    return 0
  fi
  fail "${label}" "${output}"
  exit 1
}

cleanup() {
  smoke_cleanup_project "${BIN}" "${PROJECT_DIR:-/dev/null}" 2>/dev/null || true
  rm -rf "${ROOT}" 2>/dev/null || true
}
trap cleanup EXIT

PROJECT_DIR="${ROOT}/snapshot-test"
mkdir -p "${PROJECT_DIR}"
cd "${PROJECT_DIR}"

# Init project
"${BIN}" init >/dev/null 2>&1 || true

# --- Test: Single-VM snapshot ---
require_cmd "initial up" "${BIN}" up
require_cmd "create before-snap marker" "${BIN}" exec web --timeout 3m -- touch /home/yeast/before-snap
require_cmd "down before snapshot" "${BIN}" down
OUTPUT=$("${BIN}" snapshot web clean --description "test baseline" 2>&1) && pass "snapshot creates" || fail "snapshot" "${OUTPUT}"

# --- Test: Snapshots list ---
OUTPUT=$("${BIN}" snapshots web 2>&1)
echo "${OUTPUT}" | grep -q "clean" && pass "snapshot listed" || fail "snapshots list" "${OUTPUT}"

# --- Test: Restore and verify ---
require_cmd "up before restore marker" "${BIN}" up
require_cmd "create after-snap marker" "${BIN}" exec web --timeout 3m -- touch /home/yeast/after-snap
require_cmd "down before restore" "${BIN}" down
OUTPUT=$("${BIN}" restore web clean 2>&1) && pass "restore succeeds" || fail "restore" "${OUTPUT}"
require_cmd "up after restore" "${BIN}" up
OUTPUT=$("${BIN}" exec web --timeout 3m -- ls /home/yeast/before-snap 2>&1)
echo "${OUTPUT}" | grep -q "before-snap" && pass "before-snap exists after restore" || fail "restore state" "${OUTPUT}"
OUTPUT=$("${BIN}" exec web --timeout 3m -- ls /home/yeast/after-snap 2>&1) && fail "after-snap should not exist" "file still exists" || pass "after-snap removed by restore"

# --- Test: Snapshot delete ---
require_cmd "down before delete-snapshot" "${BIN}" down
OUTPUT=$("${BIN}" delete-snapshot web clean 2>&1) && pass "delete-snapshot" || fail "delete-snapshot" "${OUTPUT}"
OUTPUT=$("${BIN}" snapshots web 2>&1)
echo "${OUTPUT}" | grep -q "clean" && fail "snapshot still listed" "should be deleted" || pass "snapshot deleted"

# --- Test: Restore non-existent snapshot ---
OUTPUT=$("${BIN}" restore web nonexistent 2>&1) && fail "restore nonexistent" "expected error" || pass "restore non-existent returns error"

echo ""
echo "Snapshots: ${PASS} passed, ${FAIL} failed, ${TOTAL} total"
[[ ${FAIL} -eq 0 ]] || exit 1
