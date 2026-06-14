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

PROJECT_DIR="${ROOT}/network-test"
mkdir -p "${PROJECT_DIR}"
cd "${PROJECT_DIR}"

"${BIN}" init >/dev/null 2>&1

# Create two-VM lab config
cat > yeast.yaml <<'EOF'
version: 1
instances:
  - name: attacker
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    networks:
      - name: labnet
        ipv4: 10.10.10.10
  - name: target
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    networks:
      - name: labnet
        ipv4: 10.10.10.20

networks:
  - name: labnet
    cidr: 10.10.10.0/24
EOF

# --- Test: Lab network creation ---
OUTPUT=$("${BIN}" up 2>&1) && pass "two-VM lab boots" || fail "up" "${OUTPUT}"

# --- Test: Lab IPs in status ---
STATUS=$("${BIN}" status --json 2>&1)
echo "${STATUS}" | python3 -c "import json,sys; d=json.load(sys.stdin); ips=[i.get('lab_ip','') for i in d['data']['instances']]; assert '10.10.10.10' in ips and '10.10.10.20' in ips" 2>/dev/null \
  && pass "lab IPs assigned" || fail "lab IPs" "missing IPs in status"

# --- Test: VM-to-VM connectivity ---
OUTPUT=$("${BIN}" exec attacker --timeout 3m -- ping -c 2 -W 5 10.10.10.20 2>&1) && pass "attacker can ping target" || fail "ping" "${OUTPUT}"
OUTPUT=$("${BIN}" exec target --timeout 3m -- ping -c 2 -W 5 10.10.10.10 2>&1) && pass "target can ping attacker" || fail "ping" "${OUTPUT}"

# --- Test: Network survives restart ---
"${BIN}" down >/dev/null 2>&1
OUTPUT=$("${BIN}" up 2>&1) && pass "restart succeeds" || fail "restart" "${OUTPUT}"
OUTPUT=$("${BIN}" exec attacker --timeout 3m -- ping -c 2 -W 5 10.10.10.20 2>&1) && pass "ping works after restart" || fail "ping restart" "${OUTPUT}"

# --- Test: Management vs lab separation ---
OUTPUT=$("${BIN}" exec attacker --timeout 3m -- ip addr show yeastlab0 2>&1)
echo "${OUTPUT}" | grep -q "10.10.10.10" && pass "lab interface has correct IP" || fail "lab interface" "${OUTPUT}"

echo ""
echo "Networking: ${PASS} passed, ${FAIL} failed, ${TOTAL} total"
[[ ${FAIL} -eq 0 ]] || exit 1
