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
  rm -rf "${ROOT}" 2>/dev/null || true
}
trap cleanup EXIT

# --- Test: List templates ---
OUTPUT=$("${BIN}" init --list-templates 2>&1)
echo "${OUTPUT}" | grep -q "ubuntu-basic" && pass "list-templates shows ubuntu-basic" || fail "list-templates" "${OUTPUT}"

OUTPUT=$("${BIN}" init --list-templates --json 2>&1)
echo "${OUTPUT}" | python3 -c "import json,sys; d=json.load(sys.stdin); t=d.get('data',{}).get('templates',d); assert isinstance(t,list) and len(t)>0" 2>/dev/null \
  && pass "list-templates --json valid" || fail "list-templates --json" "${OUTPUT}"

# --- Test: Each built-in template ---
for tmpl in ubuntu-basic caddy-single-vm two-vm-lab; do
  TMPL_DIR="${ROOT}/tmpl-${tmpl}"
  mkdir -p "${TMPL_DIR}"
  cd "${TMPL_DIR}"
  OUTPUT=$("${BIN}" init --template "${tmpl}" 2>&1) && pass "init --template ${tmpl}" || fail "init ${tmpl}" "${OUTPUT}"
  [[ -f "${TMPL_DIR}/yeast.yaml" ]] && pass "${tmpl} creates yeast.yaml" || fail "${tmpl} yeast.yaml" "not found"
done

# --- Test: Init without template ---
DEFAULT_DIR="${ROOT}/tmpl-default"
mkdir -p "${DEFAULT_DIR}"
cd "${DEFAULT_DIR}"
OUTPUT=$("${BIN}" init 2>&1) && pass "init without template" || fail "init default" "${OUTPUT}"
[[ -f "${DEFAULT_DIR}/yeast.yaml" ]] && pass "default init creates yeast.yaml" || fail "default yeast.yaml" "not found"

# --- Test: Init in non-empty directory ---
OUTPUT=$("${BIN}" init 2>&1) && fail "init in non-empty dir" "expected warning or error" || pass "init non-empty returns error"

echo ""
echo "Templates: ${PASS} passed, ${FAIL} failed, ${TOTAL} total"
[[ ${FAIL} -eq 0 ]] || exit 1
