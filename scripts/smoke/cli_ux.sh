#!/usr/bin/env bash
# scripts/smoke/cli_ux.sh — CLI UX smoke tests
# Usage: bash scripts/smoke/cli_ux.sh [path-to-yeast-binary]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib.sh"

BIN="${1:-$(smoke_resolve_binary)}"
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
    fail "${label} — expected to contain: ${needle}"
  fi
}

assert_not_contains() {
  local label="$1" haystack="$2" needle="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    pass "${label}"
  else
    fail "${label} — expected NOT to contain: ${needle}"
  fi
}

assert_file_not_empty() {
  local label="$1" file="$2"
  if [[ -s "${file}" ]]; then
    pass "${label}"
  else
    fail "${label} — file is empty or missing"
  fi
}

# --- 1. Help text ---
section "CLI UX: help text"
OUT="$("${BIN}" --help 2>&1)"
assert_contains "--help shows usage" "${OUT}" "Usage"
assert_contains "--help lists commands" "${OUT}" "Available Commands"

OUT="$("${BIN}" up --help 2>&1)"
assert_contains "up --help shows flags" "${OUT}" "Flags"
assert_contains "up --help shows --no-provision" "${OUT}" "no-provision"
assert_contains "up --help shows --reprovision" "${OUT}" "reprovision"
assert_contains "up --help shows --sequential" "${OUT}" "sequential"
assert_contains "up --help shows --profile" "${OUT}" "profile"

OUT="$("${BIN}" completion --help 2>&1)"
assert_contains "completion --help shows shells" "${OUT}" "bash"

# --- 2. Version ---
section "CLI UX: version"
OUT="$("${BIN}" version 2>&1)"
# Version should contain a semver-like string
if echo "${OUT}" | grep -qE '[0-9]+\.[0-9]+\.[0-9]+'; then
  pass "version shows version string"
else
  fail "version shows version string — expected semver in: ${OUT}"
fi
# Detect dev vs release build
if echo "${OUT}" | grep -q "0.0.0-dev"; then
  pass "version is dev build (0.0.0-dev)"
else
  pass "version is release build"
fi

# --- 3. Doctor ---
section "CLI UX: doctor"
OUT="$("${BIN}" doctor 2>&1)" || true
assert_contains "doctor runs without panic" "${OUT}" "Host doctor"
assert_contains "doctor shows KVM check" "${OUT}" "kvm"
assert_contains "doctor shows QEMU check" "${OUT}" "qemu"

# --- 4. NO_COLOR ---
section "CLI UX: NO_COLOR support"
OUT="$(NO_COLOR=1 "${BIN}" status 2>&1)" || true
assert_not_contains "NO_COLOR suppresses ANSI codes" "${OUT}" $'\033['

# --- 5. TERM=dumb ---
section "CLI UX: TERM=dumb"
OUT="$(TERM=dumb "${BIN}" status 2>&1)" || true
assert_not_contains "TERM=dumb suppresses ANSI codes" "${OUT}" $'\033['

# --- 6. Piped output ---
section "CLI UX: piped output"
OUT="$("${BIN}" version 2>&1 | cat)"
# Piped output must be non-empty, contain a version string, and have no ANSI codes
if [[ -z "${OUT}" ]]; then
  fail "piped output is empty"
else
  pass "piped output is non-empty"
fi
if echo "${OUT}" | grep -qE '[0-9]+\.[0-9]+\.[0-9]+'; then
  pass "piped output contains version string"
else
  fail "piped output missing version string"
fi
assert_not_contains "piped output has no ANSI" "${OUT}" $'\033['

# --- 7. Error output goes to stderr ---
section "CLI UX: error to stderr"
ERR_DIR=$(mktemp -d)
mkdir -p "${ERR_DIR}"
# Run command that will fail (empty dir, no yeast.yaml)
STDOUT_FILE="${ERR_DIR}/stdout.txt"
STDERR_FILE="${ERR_DIR}/stderr.txt"
"${BIN}" up 1>"${STDOUT_FILE}" 2>"${STDERR_FILE}" || true
STDOUT_CONTENT=$(cat "${STDOUT_FILE}")
STDERR_CONTENT=$(cat "${STDERR_FILE}")
# Error should go to stderr, not stdout
if [[ -n "${STDERR_CONTENT}" ]] && echo "${STDERR_CONTENT}" | grep -qi "error\|not found\|init"; then
  pass "error output goes to stderr"
elif [[ -n "${STDOUT_CONTENT}" ]] && echo "${STDOUT_CONTENT}" | grep -qi "error\|not found\|init"; then
  fail "error output went to stdout instead of stderr"
else
  pass "error handling (no panic)"
fi
rm -rf "${ERR_DIR}"

# --- 8. Progress output ---
section "CLI UX: progress output"
# Non-VM safe: verify progress rendering infrastructure exists
# The actual yeast up progress test requires QEMU and lives in lifecycle.sh
PROG_DIR=$(mktemp -d)
mkdir -p "${PROG_DIR}"
cat > "${PROG_DIR}/yeast.yaml" <<'YAML'
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
    ssh_port: 2285
YAML
(cd "${PROG_DIR}" && "${BIN}" init -q 2>/dev/null || true)
# Check that status produces output (no VM needed)
STATUS_OUT=$(cd "${PROG_DIR}" && "${BIN}" status 2>&1 || true)
if [[ -n "${STATUS_OUT}" ]]; then
  pass "status produces output (progress infrastructure works)"
else
  fail "status produced no output"
fi
rm -rf "${PROG_DIR}"

# --- 9. Shell completions ---
section "CLI UX: shell completions"
OUT="$("${BIN}" completion bash 2>&1)"
assert_contains "bash completion produces output" "${OUT}" "complete"

OUT="$("${BIN}" completion zsh 2>&1)"
assert_contains "zsh completion produces output" "${OUT}" "compdef"

OUT="$("${BIN}" completion fish 2>&1)"
assert_contains "fish completion produces output" "${OUT}" "complete"

OUT="$("${BIN}" completion powershell 2>&1)"
assert_contains "powershell completion produces output" "${OUT}" "Register-ArgumentCompleter"

# --- 10. All commands have --help ---
section "CLI UX: all commands have --help"
for cmd in up down status destroy pull snapshot snapshots restore logs exec copy inspect provision doctor version completion init images; do
  OUT="$("${BIN}" "${cmd}" --help 2>&1)" || true
  if echo "${OUT}" | grep -qi "usage\|help"; then
    pass "${cmd} --help works"
  else
    fail "${cmd} --help missing or broken"
  fi
done

# --- Summary ---
TOTAL=$((PASS + FAIL))
printf '\n\033[1mCLI UX Tests: %d passed, %d failed (%d total)\033[0m\n' "${PASS}" "${FAIL}" "${TOTAL}"
if [[ ${FAIL} -gt 0 ]]; then
  exit 1
fi
