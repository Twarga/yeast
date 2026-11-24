#!/usr/bin/env bash
# scripts/smoke/superpowers.sh — v2 feature smoke: live snapshot/restore
# timing, fork, screenshot, mounts, mcp round-trip, qcow2 export.
# Requires a KVM host and a cached yeast image (skipped otherwise).
set -euo pipefail

SMOKE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SMOKE_DIR}/lib.sh"

BIN="${1:-$(smoke_resolve_binary)}"
ROOT="$(smoke_prepare_root)"
PASS=0
FAIL=0
BENCH=0
RESULTS_FILE="${SMOKE_REPO_ROOT}/artifacts/bench-superpowers.txt"

pass() { PASS=$((PASS+1)); printf '  ✓ PASS  %s\n' "$1"; }
fail() { FAIL=$((FAIL+1)); printf '  ✗ FAIL  %s\n' "$1"; printf '          → %s\n' "$2"; }
bench() {
  local label="$1" ms="$2"
  BENCH=$((BENCH+1))
  printf '  ● BENCH %-34s %s ms\n' "${label}" "${ms}"
  echo "${label}=${ms}ms" >> "${RESULTS_FILE}"
}
now_ms() { date +%s%3N; }

cleanup() {
  (cd "${ROOT}" 2>/dev/null && "${BIN}" destroy --json >/dev/null 2>&1) || true
  rm -rf "${ROOT}" 2>/dev/null || true
}
trap cleanup EXIT

PROJECT_DIR="${ROOT}/project"
mkdir -p "${PROJECT_DIR}" "${ROOT}/shared"
cd "${PROJECT_DIR}"

require_cmd() {
  local label="$1"; shift
  local output
  if output=$("$@" 2>&1); then return 0; fi
  fail "${label}" "${output}"; exit 1
}

# --- skip unless the host can run a real VM ---
if ! "${BIN}" doctor --json 2>/dev/null | grep -q '"kvm".*ok\|kvm.*pass'; then
  if [[ ! -w /dev/kvm ]]; then
    echo "  ⤷ SKIP  /dev/kvm unavailable — superpowers suite needs a real KVM host"
    exit 0
  fi
fi

require_cmd "init" "${BIN}" init
cat > yeast.yaml <<'YAML'
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 2
    ssh_port: 2297
    display:
      enabled: true
      type: vnc
    mounts:
      - source: SHARED_DIR_PLACEHOLDER
        target: /mnt/shared
YAML
sed -i "s|SHARED_DIR_PLACEHOLDER|${ROOT}/shared|" yeast.yaml

t0=$(now_ms)
require_cmd "up (display+mount)" "${BIN}" up
bench "boot" $(( $(now_ms) - t0 ))

# --- screenshot: QMP framebuffer dump → valid non-empty PNG ---
SHOT="${ROOT}/shot.png"
require_cmd "screenshot" "${BIN}" screenshot web -o "${SHOT}"
if [[ -s "${SHOT}" ]] && python3 -c "import sys;d=open(sys.argv[1],'rb').read(8);assert d[:8]==b'\x89PNG\r\n\x1a\n'" "${SHOT}" 2>/dev/null; then
  pass "screenshot is a valid non-empty PNG"
else
  fail "screenshot PNG" "missing or invalid magic bytes"
fi

# --- programmatic input (QMP send-key) ---
if "${BIN}" input web --key ctrl_l+alt_l+f1 >/dev/null 2>&1; then
  pass "input key chord accepted"
else
  fail "input" "send-key failed"
fi

# --- virtiofs/9p mount read/write ---
echo "host-writes" > "${ROOT}/shared/probe.txt"
if "${BIN}" exec web -- cat /mnt/shared/probe.txt 2>/dev/null | grep -q "host-writes"; then
  pass "shared mount readable in guest"
else
  fail "shared mount" "guest cannot read host file (mount driver missing?)"
fi
if "${BIN}" exec web -- sh -c 'echo guest-writes | sudo tee /mnt/shared/echo.txt' >/dev/null 2>&1 \
   && grep -q "guest-writes" "${ROOT}/shared/echo.txt" 2>/dev/null; then
  pass "shared mount writable both ways"
else
  fail "shared mount write" "guest write did not reach host"
fi

# --- live snapshot + restore timing ---
t0=$(now_ms)
if "${BIN}" snapshot web pre-fork >/dev/null 2>&1; then
  bench "live-snapshot" $(( $(now_ms) - t0 ))
  pass "live snapshot taken"
else
  fail "live snapshot" "command failed"
fi
echo gone > "${ROOT}/shared/probe.txt"
t0=$(now_ms)
if "${BIN}" restore web pre-fork >/dev/null 2>&1; then
  ms=$(( $(now_ms) - t0 ))
  bench "live-restore" "${ms}"
  if [[ ${ms} -lt 3000 ]]; then
    pass "live restore under 3s (${ms}ms)"
  else
    fail "live restore speed" "${ms}ms exceeds 3s target"
  fi
else
  fail "live restore" "command failed"
fi

# --- fork ---
t0=$(now_ms)
if "${BIN}" fork web --count 1 >/dev/null 2>&1; then
  bench "fork" $(( $(now_ms) - t0 ))
  pass "fork created"
else
  fail "fork" "command failed"
fi
if "${BIN}" status --json 2>/dev/null | grep -q "web-fork"; then
  pass "fork appears in status"
else
  fail "fork status" "fork not listed"
fi

# --- mcp round-trip over stdio ---
MCP_OUT="$(printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"1"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
  | "${BIN}" mcp 2>/dev/null || true)"
if echo "${MCP_OUT}" | grep -q '"vm_status"' && echo "${MCP_OUT}" | grep -q '"vm_exec"'; then
  pass "mcp initialize + tools/list round-trip"
else
  fail "mcp" "missing tools in response: ${MCP_OUT:0:200}"
fi

# --- audit trail exists ---
if [[ -s .yeast/audit.jsonl ]] && grep -q '"action"' .yeast/audit.jsonl; then
  pass "audit.jsonl recorded actions"
else
  fail "audit" "no audit entries written"
fi

# --- down + qcow2 export ---
require_cmd "down" "${BIN}" down
QCOW="${ROOT}/export-web.qcow2"
t0=$(now_ms)
if "${BIN}" export web --format qcow2 -o "${QCOW2}" >/dev/null 2>&1; then
  bench "export-qcow2" $(( $(now_ms) - t0 ))
  if python3 -c "import sys;d=open(sys.argv[1],'rb').read(4);assert d==b'QFI\xfb'" "${QCOW2}" 2>/dev/null; then
    pass "export qcow2 is a valid image"
  else
    fail "export qcow2" "invalid qcow2 magic"
  fi
else
  fail "export qcow2" "command failed"
fi

echo ""
echo "Superpowers: ${PASS} passed, ${FAIL} failed (${BENCH} benchmarks → ${RESULTS_FILE})"
[[ ${FAIL} -eq 0 ]] || exit 1
