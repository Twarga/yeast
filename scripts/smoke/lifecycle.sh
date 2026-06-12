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

# --- Test: Fresh init ---
PROJECT_DIR="${ROOT}/lifecycle-init"
mkdir -p "${PROJECT_DIR}"
cd "${PROJECT_DIR}"

OUTPUT=$("${BIN}" init 2>&1) && pass "init creates yeast.yaml and .yeast/project.json" || fail "init" "${OUTPUT}"
[[ -f "${PROJECT_DIR}/yeast.yaml" ]] && pass "yeast.yaml exists" || fail "yeast.yaml" "file not found"
[[ -f "${PROJECT_DIR}/.yeast/project.json" ]] && pass "project.json exists" || fail "project.json" "file not found"

# --- Test: Pull image ---
OUTPUT=$("${BIN}" pull ubuntu-24.04 2>&1) && pass "pull downloads image" || fail "pull" "${OUTPUT}"
IMAGE_FILE="${HOME}/.yeast/cache/images/ubuntu-24.04/image.qcow2"
[[ -f "${IMAGE_FILE}" ]] && pass "image file cached" || fail "image cache" "file not found"

# --- Test: First boot (cold) ---
OUTPUT=$("${BIN}" up 2>&1) && pass "up boots VM" || fail "up" "${OUTPUT}"
STATUS_OUTPUT=$("${BIN}" status --json 2>&1)
echo "${STATUS_OUTPUT}" | python3 -c "import json,sys; d=json.load(sys.stdin); assert d['data']['instances'][0]['status']=='running'" 2>/dev/null \
  && pass "status shows running" || fail "status" "not running"

# --- Test: Warm boot ---
"${BIN}" down >/dev/null 2>&1
OUTPUT=$("${BIN}" up 2>&1) && pass "warm boot succeeds" || fail "warm boot" "${OUTPUT}"

# --- Test: Stop ---
OUTPUT=$("${BIN}" down 2>&1) && pass "down stops VM" || fail "down" "${OUTPUT}"

# --- Test: Destroy ---
OUTPUT=$("${BIN}" destroy 2>&1) && pass "destroy removes runtime" || fail "destroy" "${OUTPUT}"

# --- Test: Idempotency ---
"${BIN}" down >/dev/null 2>&1 && pass "down when already stopped" || fail "down idempotent" "error"
"${BIN}" destroy >/dev/null 2>&1 && pass "destroy when already destroyed" || fail "destroy idempotent" "error"

# --- Test: Doctor ---
OUTPUT=$("${BIN}" doctor 2>&1) && pass "doctor runs" || fail "doctor" "${OUTPUT}"

echo ""
echo "Lifecycle: ${PASS} passed, ${FAIL} failed, ${TOTAL} total"
[[ ${FAIL} -eq 0 ]] || exit 1
