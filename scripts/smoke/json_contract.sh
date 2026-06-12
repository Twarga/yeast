#!/usr/bin/env bash
# scripts/smoke/json_contract.sh — JSON contract smoke tests
# Usage: bash scripts/smoke/json_contract.sh [path-to-yeast-binary]
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

assert_valid_json() {
  local label="$1" payload="$2"
  if echo "${payload}" | python3 -m json.tool >/dev/null 2>&1; then
    pass "${label}"
  else
    fail "${label} — invalid JSON"
  fi
}

assert_json_field() {
  local label="$1" payload="$2" field="$3" expected="$4"
  local actual
  actual="$(echo "${payload}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d${field})" 2>/dev/null || echo "__MISSING__")"
  if [[ "${actual}" == "${expected}" ]]; then
    pass "${label}"
  else
    fail "${label} — expected ${field}=${expected}, got ${actual}"
  fi
}

assert_json_has_field() {
  local label="$1" payload="$2" field="$3"
  if echo "${payload}" | python3 -c "import json,sys; d=json.load(sys.stdin); d${field}" 2>/dev/null; then
    pass "${label}"
  else
    fail "${label} — field ${field} missing"
  fi
}

# --- Doctor ---
section "JSON: doctor"
OUT="$("${BIN}" doctor --json 2>&1)" || true
assert_valid_json "doctor --json is valid JSON" "${OUT}"
assert_json_field "doctor has ok field" "${OUT}" "['ok']" "True"

# --- Version ---
section "JSON: version"
OUT="$("${BIN}" version --json 2>&1)" || true
assert_valid_json "version --json is valid JSON" "${OUT}"

# --- Init ---
section "JSON: init"
INIT_DIR="${WORK_DIR}/json-init"
mkdir -p "${INIT_DIR}"
OUT="$(cd "${INIT_DIR}" && "${BIN}" init --json 2>&1)" || true
assert_valid_json "init --json is valid JSON" "${OUT}"
assert_json_field "init has ok=true" "${OUT}" "['ok']" "True"
rm -rf "${INIT_DIR}"

# --- Pull --list ---
section "JSON: pull --list"
OUT="$("${BIN}" pull --list --json 2>&1)" || true
assert_valid_json "pull --list --json is valid JSON" "${OUT}"
assert_json_field "pull --list has ok=true" "${OUT}" "['ok']" "True"

# --- Up + Status + Down ---
section "JSON: up/status/down lifecycle"
LIFE_DIR="${WORK_DIR}/json-lifecycle"
mkdir -p "${LIFE_DIR}"
(cd "${LIFE_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${LIFE_DIR}/yeast.yaml" <<'YAML'
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
    ssh_port: 2290
provision:
  packages:
    - curl
YAML

OUT="$(cd "${LIFE_DIR}" && "${BIN}" up --json 2>&1)" || true
assert_valid_json "up --json is valid JSON" "${OUT}"

OUT="$(cd "${LIFE_DIR}" && "${BIN}" status --json 2>&1)" || true
assert_valid_json "status --json is valid JSON" "${OUT}"

# --- Exec ---
section "JSON: exec"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" exec web --json -- echo test-json-contract 2>&1)" || true
assert_valid_json "exec --json is valid JSON" "${OUT}"
assert_json_has_field "exec has stdout field" "${OUT}" "['data']['run']['stdout']"
assert_json_has_field "exec has exit_code field" "${OUT}" "['data']['run']['exit_code']"
assert_json_has_field "exec has duration field" "${OUT}" "['data']['run']['duration']"

# --- Inspect ---
section "JSON: inspect"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" inspect web --json 2>&1)" || true
assert_valid_json "inspect --json is valid JSON" "${OUT}"
assert_json_has_field "inspect has name field" "${OUT}" "['data']['instance']['name']"

# --- Logs ---
section "JSON: logs"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" logs web --json 2>&1)" || true
assert_valid_json "logs --json is valid JSON" "${OUT}"

# --- Snapshot ---
section "JSON: snapshot"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" down --json 2>&1)" || true
assert_valid_json "down --json is valid JSON" "${OUT}"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" snapshot web json-test --json 2>&1)" || true
assert_valid_json "snapshot --json is valid JSON" "${OUT}"

# --- Snapshots ---
section "JSON: snapshots"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" snapshots web --json 2>&1)" || true
assert_valid_json "snapshots --json is valid JSON" "${OUT}"

# --- Restore ---
section "JSON: restore"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" restore web json-test --json 2>&1)" || true
assert_valid_json "restore --json is valid JSON" "${OUT}"

# --- Delete-snapshot ---
section "JSON: delete-snapshot"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" delete-snapshot web json-test --json 2>&1)" || true
assert_valid_json "delete-snapshot --json is valid JSON" "${OUT}"

# --- Destroy ---
section "JSON: destroy"
OUT="$(cd "${LIFE_DIR}" && "${BIN}" destroy --json 2>&1)" || true
assert_valid_json "destroy --json is valid JSON" "${OUT}"

smoke_cleanup_project "${BIN}" "${LIFE_DIR}"
rm -rf "${LIFE_DIR}"

# --- Event stream ---
section "JSON: event stream"
EVENT_DIR="${WORK_DIR}/json-events"
mkdir -p "${EVENT_DIR}"
(cd "${EVENT_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${EVENT_DIR}/yeast.yaml" <<'YAML'
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
    ssh_port: 2291
YAML

OUT="$(cd "${EVENT_DIR}" && "${BIN}" up --json --events 2>&1)" || true
# Each line should be valid JSON
FIRST_LINE="$(echo "${OUT}" | head -1)"
assert_valid_json "event stream line 1 is valid JSON" "${FIRST_LINE}"

LAST_LINE="$(echo "${OUT}" | tail -1)"
assert_valid_json "event stream last line is valid JSON" "${LAST_LINE}"

# Verify event envelope fields
assert_json_has_field "event has schema_version" "${FIRST_LINE}" "['schema_version']"
assert_json_has_field "event has type" "${FIRST_LINE}" "['type']"
assert_json_has_field "event has name" "${FIRST_LINE}" "['name']"
assert_json_has_field "event has time" "${FIRST_LINE}" "['time']"

# Verify first event is project.loaded
FIRST_EVENT_NAME="$(echo "${FIRST_LINE}" | python3 -c "import json,sys; print(json.load(sys.stdin)['name'])" 2>/dev/null || echo "")"
if [[ "${FIRST_EVENT_NAME}" == "project.loaded" ]]; then
  pass "first event is project.loaded"
else
  fail "first event expected project.loaded, got ${FIRST_EVENT_NAME}"
fi

# Verify last event is workflow.completed. The final line may be the command
# result envelope, so select the last JSON line that has an event name.
LAST_EVENT_LINE="$(echo "${OUT}" | python3 -c "import json,sys
last = ''
for line in sys.stdin:
    if not line.strip():
        continue
    obj = json.loads(line)
    if obj.get('type') == 'event' and 'name' in obj:
        last = line
print(last, end='')")"
LAST_EVENT_NAME="$(echo "${LAST_EVENT_LINE}" | python3 -c "import json,sys; print(json.load(sys.stdin)['name'])" 2>/dev/null || echo "")"
if [[ "${LAST_EVENT_NAME}" == "workflow.completed" ]]; then
  pass "last event is workflow.completed"
else
  fail "last event expected workflow.completed, got ${LAST_EVENT_NAME}"
fi

# Verify all lines are valid JSON
INVALID_LINES=0
while IFS= read -r line; do
  if [[ -n "${line}" ]] && ! echo "${line}" | python3 -m json.tool >/dev/null 2>&1; then
    INVALID_LINES=$((INVALID_LINES + 1))
  fi
done <<< "${OUT}"
if [[ ${INVALID_LINES} -eq 0 ]]; then
  pass "all event stream lines are valid JSON"
else
  fail "${INVALID_LINES} lines are not valid JSON"
fi

(cd "${EVENT_DIR}" && "${BIN}" down >/dev/null 2>&1 || true)
smoke_cleanup_project "${BIN}" "${EVENT_DIR}"
rm -rf "${EVENT_DIR}"

# --- Error JSON ---
section "JSON: error format"
ERR_DIR="${WORK_DIR}/json-error"
mkdir -p "${ERR_DIR}"
(cd "${ERR_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${ERR_DIR}/yeast.yaml" <<'YAML'
version: 1
instances:
  - name: web
    image: nonexistent-image-1.0
    memory: 512
    disk: 8
YAML

OUT="$(cd "${ERR_DIR}" && "${BIN}" up --json 2>&1)" || true
assert_valid_json "error response is valid JSON" "${OUT}"
assert_json_field "error has ok=false" "${OUT}" "['ok']" "False"

smoke_cleanup_project "${BIN}" "${ERR_DIR}"
rm -rf "${ERR_DIR}"

# --- Summary ---
rm -rf "${WORK_DIR}"
TOTAL=$((PASS + FAIL))
printf '\n\033[1mJSON Contract Tests: %d passed, %d failed (%d total)\033[0m\n' "${PASS}" "${FAIL}" "${TOTAL}"
if [[ ${FAIL} -gt 0 ]]; then
  exit 1
fi
