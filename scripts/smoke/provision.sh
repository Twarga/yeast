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

PROJECT_DIR="${ROOT}/provision-test"
mkdir -p "${PROJECT_DIR}"
cd "${PROJECT_DIR}"

# Create a test file for file provisioning
mkdir -p site
echo "hello from smoke test" > site/test.txt

# Init project metadata before replacing the generated config.
"${BIN}" init >/dev/null 2>&1

# Create yeast.yaml with all provision types
cat > yeast.yaml <<'EOF'
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    provision:
      files:
        - source: ./site/test.txt
          destination: /home/yeast/test.txt
      shell:
        - "echo smoke-sentinel > /tmp/shell-test.txt"
EOF

# --- Test: Full provision on first boot ---
if OUTPUT=$("${BIN}" up 2>&1); then
  pass "up with provisioning succeeds"
else
  fail "up" "${OUTPUT}"
  exit 1
fi

# --- Test: File provisioning ---
OUTPUT=$("${BIN}" exec web -- cat /home/yeast/test.txt 2>&1)
echo "${OUTPUT}" | grep -q "hello from smoke test" && pass "file provisioned correctly" || fail "file provision" "${OUTPUT}"

# --- Test: Shell step execution ---
OUTPUT=$("${BIN}" exec web -- cat /tmp/shell-test.txt 2>&1)
echo "${OUTPUT}" | grep -q "smoke-sentinel" && pass "shell step executed" || fail "shell step" "${OUTPUT}"

# --- Test: Config change triggers re-provision ---
sed -i '/echo smoke-sentinel/a\        - "echo re-provisioned > /tmp/reprov-test.txt"' yeast.yaml
if OUTPUT=$("${BIN}" provision web 2>&1); then
  pass "provision after config change"
else
  fail "re-provision command" "${OUTPUT}"
  exit 1
fi
OUTPUT=$("${BIN}" exec web -- cat /tmp/reprov-test.txt 2>&1)
echo "${OUTPUT}" | grep -q "re-provisioned" && pass "re-provision executed" || fail "re-provision" "${OUTPUT}"

# --- Test: Provision skip on warm boot ---
"${BIN}" down >/dev/null 2>&1
START=$(smoke_now)
if OUTPUT=$("${BIN}" up 2>&1); then
  :
else
  fail "warm boot" "${OUTPUT}"
  exit 1
fi
WARM_TIME=$(( $(smoke_now) - START ))
pass "warm boot completes in ${WARM_TIME}s"

# --- Test: Provision failure handling ---
cat > yeast.yaml <<'EOF'
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    provision:
      shell:
        - "exit 1"
EOF
"${BIN}" down >/dev/null 2>&1
OUTPUT=$("${BIN}" up 2>&1) && fail "up with failing provision" "expected error" || pass "failing provision returns error"

echo ""
echo "Provisioning: ${PASS} passed, ${FAIL} failed, ${TOTAL} total"
[[ ${FAIL} -eq 0 ]] || exit 1
