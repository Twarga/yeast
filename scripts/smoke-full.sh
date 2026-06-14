#!/usr/bin/env bash
set -euo pipefail

SMOKE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/smoke"
# shellcheck source=lib.sh
source "${SMOKE_DIR}/lib.sh"

BIN="${1:-}"
if [[ -z "${BIN}" ]]; then
  BIN="$(smoke_resolve_binary)"
fi
CATEGORY="${SMOKE_CATEGORY:-all}"
TOTAL_PASS=0
TOTAL_FAIL=0
TOTAL_BENCH=0

echo "═══════════════════════════════════════════════════════════"
echo "YEAST SMOKE TEST REPORT"
echo "Binary: ${BIN}"
echo "Host:   $(uname -s) $(uname -r)"
echo "Date:   $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
echo "═══════════════════════════════════════════════════════════"
echo ""

run_category() {
  local name="$1"
  local script="$2"
  if [[ ! -f "${script}" ]]; then
    printf '[%s] SKIP (script not found)\n\n' "${name}"
    return 0
  fi
  printf '[%s]\n' "${name}"
  local output
  if output=$(bash "${script}" "${BIN}" 2>&1); then
    echo "${output}"
    local p f
    p=$(echo "${output}" | grep -c "✓ PASS" || true)
    f=$(echo "${output}" | grep -c "✗ FAIL" || true)
    TOTAL_PASS=$((TOTAL_PASS + p))
    TOTAL_FAIL=$((TOTAL_FAIL + f))
  else
    echo "${output}"
    TOTAL_FAIL=$((TOTAL_FAIL + 1))
  fi
  echo ""
}

case "${CATEGORY}" in
  all)
    run_category "LIFECYCLE" "${SMOKE_DIR}/lifecycle.sh"
    run_category "PROVISIONING" "${SMOKE_DIR}/provision.sh"
    run_category "SNAPSHOTS" "${SMOKE_DIR}/snapshot.sh"
    run_category "NETWORKING" "${SMOKE_DIR}/network.sh"
    run_category "GUEST CONTROL" "${SMOKE_DIR}/guest.sh"
    run_category "JSON CONTRACT" "${SMOKE_DIR}/json_contract.sh"
    run_category "TEMPLATES" "${SMOKE_DIR}/template.sh"
    run_category "ERROR PATHS" "${SMOKE_DIR}/error_paths.sh"
    run_category "CLI UX" "${SMOKE_DIR}/cli_ux.sh"
    run_category "PERFORMANCE" "${SMOKE_DIR}/perf.sh"
    ;;
  lifecycle)   run_category "LIFECYCLE" "${SMOKE_DIR}/lifecycle.sh" ;;
  provision)   run_category "PROVISIONING" "${SMOKE_DIR}/provision.sh" ;;
  snapshot)    run_category "SNAPSHOTS" "${SMOKE_DIR}/snapshot.sh" ;;
  network)     run_category "NETWORKING" "${SMOKE_DIR}/network.sh" ;;
  guest)       run_category "GUEST CONTROL" "${SMOKE_DIR}/guest.sh" ;;
  json)        run_category "JSON CONTRACT" "${SMOKE_DIR}/json_contract.sh" ;;
  template)    run_category "TEMPLATES" "${SMOKE_DIR}/template.sh" ;;
  error)       run_category "ERROR PATHS" "${SMOKE_DIR}/error_paths.sh" ;;
  cli-ux)      run_category "CLI UX" "${SMOKE_DIR}/cli_ux.sh" ;;
  perf)        run_category "PERFORMANCE" "${SMOKE_DIR}/perf.sh" ;;
  *)           echo "Unknown category: ${CATEGORY}"; exit 1 ;;
esac

echo "═══════════════════════════════════════════════════════════"
echo "RESULTS: ${TOTAL_PASS} passed, ${TOTAL_FAIL} failed"
echo "═══════════════════════════════════════════════════════════"

[[ ${TOTAL_FAIL} -eq 0 ]] || exit 1
