#!/usr/bin/env bash
# scripts/smoke/error_paths.sh — Error path smoke tests
# Usage: bash scripts/smoke/error_paths.sh [path-to-yeast-binary]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib.sh"

BIN="${1:-$(smoke_resolve_binary)}"
WORK_DIR="$(smoke_prepare_root)"
PASS=0
FAIL=0

section() { printf '\n\033[1;36m═══ %s ═══\033[0m\n' "$*"; }
pass() { PASS=$((PASS + 1)); printf '  \033[32m✓ PASS\033[0m  %s\n' "$*"; }
fail() { FAIL=$((FAIL + 1)); printf '  \033[31m✗ FAIL\033[0m  %s\n' "$*"; }

assert_contains() {
  local label="$1" haystack="$2" needle="$3"
  if [[ "${haystack}" == *"${needle}"* ]]; then
    pass "${label}"
  else
    fail "${label} — expected output to contain: ${needle}"
  fi
}

# --- 1. Invalid YAML ---
section "Error: invalid YAML"
BAD_YAML_DIR="${WORK_DIR}/bad-yaml"
mkdir -p "${BAD_YAML_DIR}"
(cd "${BAD_YAML_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${BAD_YAML_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: [invalid
YAML
OUT="$(cd "${BAD_YAML_DIR}" && "${BIN}" up 2>&1)" || true
assert_contains "invalid YAML shows parse error" "${OUT}" "error"
rm -rf "${BAD_YAML_DIR}"

# --- 2. Missing image ---
section "Error: missing image"
MISSING_IMG_DIR="${WORK_DIR}/missing-img"
mkdir -p "${MISSING_IMG_DIR}"
(cd "${MISSING_IMG_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${MISSING_IMG_DIR}/yeast.yaml" <<'YAML'
version: 1
instances:
  - name: web
    image: nonexistent-image-99.99
    memory: 512
    disk: 8
YAML
OUT="$(cd "${MISSING_IMG_DIR}" && "${BIN}" up 2>&1)" || true
assert_contains "missing image shows suggestion" "${OUT}" "not found"
rm -rf "${MISSING_IMG_DIR}"

# --- 3. Missing project metadata ---
section "Error: missing project metadata"
NO_META_DIR="${WORK_DIR}/no-meta"
mkdir -p "${NO_META_DIR}"
cat > "${NO_META_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
YAML
OUT="$(cd "${NO_META_DIR}" && "${BIN}" up 2>&1)" || true
assert_contains "missing metadata shows init hint" "${OUT}" "init"
rm -rf "${NO_META_DIR}"

# --- 4. Duplicate init ---
section "Error: duplicate init"
DUP_INIT_DIR="${WORK_DIR}/dup-init"
mkdir -p "${DUP_INIT_DIR}"
(cd "${DUP_INIT_DIR}" && "${BIN}" init >/dev/null 2>&1)
OUT="$(cd "${DUP_INIT_DIR}" && "${BIN}" init 2>&1)" || true
assert_contains "duplicate init shows error" "${OUT}" "error"
rm -rf "${DUP_INIT_DIR}"

# --- 5. Invalid ssh_port ---
section "Error: invalid ssh_port"
BAD_PORT_DIR="${WORK_DIR}/bad-port"
mkdir -p "${BAD_PORT_DIR}"
(cd "${BAD_PORT_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${BAD_PORT_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
    ssh_port: not-a-number
YAML
OUT="$(cd "${BAD_PORT_DIR}" && "${BIN}" up 2>&1)" || true
assert_contains "invalid ssh_port shows error" "${OUT}" "error"
rm -rf "${BAD_PORT_DIR}"

# --- 6. Invalid disk_size ---
section "Error: invalid disk_size"
BAD_DISK_DIR="${WORK_DIR}/bad-disk"
mkdir -p "${BAD_DISK_DIR}"
(cd "${BAD_DISK_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${BAD_DISK_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: -5
YAML
OUT="$(cd "${BAD_DISK_DIR}" && "${BIN}" up 2>&1)" || true
assert_contains "negative disk_size shows error" "${OUT}" "error"
rm -rf "${BAD_DISK_DIR}"

# --- 7. Invalid hostname ---
section "Error: invalid hostname"
BAD_HOST_DIR="${WORK_DIR}/bad-hostname"
mkdir -p "${BAD_HOST_DIR}"
(cd "${BAD_HOST_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${BAD_HOST_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
    hostname: "invalid host name with spaces!"
YAML
OUT="$(cd "${BAD_HOST_DIR}" && "${BIN}" up 2>&1)" || true
assert_contains "invalid hostname shows error" "${OUT}" "error"
rm -rf "${BAD_HOST_DIR}"

# --- 8. Snapshot of non-existent instance ---
section "Error: snapshot non-existent instance"
SNAP_DIR="${WORK_DIR}/snap-none"
mkdir -p "${SNAP_DIR}"
(cd "${SNAP_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${SNAP_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
YAML
OUT="$(cd "${SNAP_DIR}" && "${BIN}" snapshot nonexistent clean 2>&1)" || true
assert_contains "snapshot non-existent instance shows error" "${OUT}" "error"
rm -rf "${SNAP_DIR}"

# --- 9. Restore of non-existent snapshot ---
section "Error: restore non-existent snapshot"
REST_DIR="${WORK_DIR}/restore-none"
mkdir -p "${REST_DIR}"
(cd "${REST_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${REST_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
YAML
OUT="$(cd "${REST_DIR}" && "${BIN}" restore web nonexistent 2>&1)" || true
assert_contains "restore non-existent snapshot shows error" "${OUT}" "error"
rm -rf "${REST_DIR}"

# --- 10. Port conflict ---
section "Error: port conflict"
PORT_DIR="${WORK_DIR}/port-conflict"
mkdir -p "${PORT_DIR}"
(cd "${PORT_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${PORT_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
    ssh_port: 2299
YAML
# Start a listener on the port to simulate conflict
python3 -c "import socket; s=socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1); s.bind(('127.0.0.1', 2299)); s.listen(1); import time; time.sleep(30)" &
LISTENER_PID=$!
sleep 0.5
OUT="$(cd "${PORT_DIR}" && "${BIN}" up 2>&1)" || true
assert_contains "port conflict shows error" "${OUT}" "error"
kill ${LISTENER_PID} 2>/dev/null || true
wait ${LISTENER_PID} 2>/dev/null || true
rm -rf "${PORT_DIR}"

# --- 11. Corrupted state ---
section "Error: corrupted state"
CORRUPT_DIR="${WORK_DIR}/corrupt-state"
mkdir -p "${CORRUPT_DIR}"
(cd "${CORRUPT_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${CORRUPT_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
YAML
# Create a corrupted state file
PROJECT_ID="$(smoke_project_id "${CORRUPT_DIR}")"
STATE_DIR="${HOME}/.yeast/projects/${PROJECT_ID}"
mkdir -p "${STATE_DIR}"
echo "{invalid json" > "${STATE_DIR}/state.json"
OUT="$(cd "${CORRUPT_DIR}" && "${BIN}" status 2>&1)" || true
# Should handle gracefully (not panic)
if echo "${OUT}" | grep -qi "panic"; then
  fail "corrupted state causes panic"
else
  pass "corrupted state handled gracefully"
fi
rm -rf "${CORRUPT_DIR}"

# --- Summary ---
rm -rf "${WORK_DIR}"
TOTAL=$((PASS + FAIL))
printf '\n\033[1mError Path Tests: %d passed, %d failed (%d total)\033[0m\n' "${PASS}" "${FAIL}" "${TOTAL}"
if [[ ${FAIL} -gt 0 ]]; then
  exit 1
fi
