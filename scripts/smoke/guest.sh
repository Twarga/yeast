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

cleanup() {
  smoke_cleanup_project "${BIN}" "${PROJECT_DIR:-/dev/null}" 2>/dev/null || true
  rm -rf "${ROOT}" 2>/dev/null || true
}
trap cleanup EXIT

PROJECT_DIR="${ROOT}/guest-test"
mkdir -p "${PROJECT_DIR}"
cd "${PROJECT_DIR}"

"${BIN}" init >/dev/null 2>&1 || true
"${BIN}" up >/dev/null 2>&1

# --- Test: Exec basic ---
OUTPUT=$("${BIN}" exec web -- whoami 2>&1)
echo "${OUTPUT}" | grep -q "yeast" && pass "exec whoami returns yeast" || fail "exec whoami" "${OUTPUT}"

OUTPUT=$("${BIN}" exec web -- hostname 2>&1) && pass "exec hostname" || fail "exec hostname" "${OUTPUT}"

# --- Test: Exec with complex commands ---
OUTPUT=$("${BIN}" exec web -- bash -c 'for i in 1 2 3; do echo $i; done' 2>&1)
echo "${OUTPUT}" | grep -q "1" && echo "${OUTPUT}" | grep -q "2" && echo "${OUTPUT}" | grep -q "3" \
  && pass "exec loop output" || fail "exec loop" "${OUTPUT}"

# --- Test: Exec exit codes ---
"${BIN}" exec web -- true 2>/dev/null && pass "exec true returns 0" || fail "exec true" "non-zero exit"
"${BIN}" exec web -- false 2>/dev/null && fail "exec false" "expected non-zero" || pass "exec false returns non-zero"

# --- Test: Copy to guest ---
echo "test content for copy" > /tmp/smoke-local.txt
OUTPUT=$("${BIN}" copy web --to-guest /tmp/smoke-local.txt /home/yeast/copied.txt 2>&1) && pass "copy to guest" || fail "copy to guest" "${OUTPUT}"
OUTPUT=$("${BIN}" exec web -- cat /home/yeast/copied.txt 2>&1)
echo "${OUTPUT}" | grep -q "test content for copy" && pass "copied file content matches" || fail "copy content" "${OUTPUT}"

# --- Test: Copy from guest ---
"${BIN}" exec web -- bash -c 'echo from-guest > /tmp/guest-file.txt' 2>/dev/null
OUTPUT=$("${BIN}" copy web --from-guest /tmp/guest-file.txt /tmp/smoke-downloaded.txt 2>&1) && pass "copy from guest" || fail "copy from guest" "${OUTPUT}"
grep -q "from-guest" /tmp/smoke-downloaded.txt && pass "downloaded content matches" || fail "download content" "mismatch"

# --- Test: Logs ---
OUTPUT=$("${BIN}" logs web 2>&1) && pass "logs returns output" || fail "logs" "${OUTPUT}"

# --- Test: Inspect ---
OUTPUT=$("${BIN}" inspect web --json 2>&1)
echo "${OUTPUT}" | python3 -c "import json,sys; d=json.load(sys.stdin); assert d['data']['instance']['name']=='web'" 2>/dev/null \
  && pass "inspect --json valid" || fail "inspect" "${OUTPUT}"

# --- Test: Exec JSON mode ---
OUTPUT=$("${BIN}" exec web --json -- echo hi 2>&1)
echo "${OUTPUT}" | python3 -c "import json,sys; d=json.load(sys.stdin); assert 'stdout' in d['data']['run']" 2>/dev/null \
  && pass "exec --json valid" || fail "exec --json" "${OUTPUT}"

"${BIN}" down >/dev/null 2>&1

echo ""
echo "Guest Control: ${PASS} passed, ${FAIL} failed, ${TOTAL} total"
[[ ${FAIL} -eq 0 ]] || exit 1
