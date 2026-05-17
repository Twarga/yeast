#!/usr/bin/env bash
set -euo pipefail

BIN_PATH="${1:-}"
WORKDIR="${WORKDIR:-/tmp/yeast-v020-test}"
IMAGE_NAME="${IMAGE_NAME:-ubuntu-24.04}"
INSTANCE_NAME="${INSTANCE_NAME:-web}"
INSTANCE_HOSTNAME="${INSTANCE_HOSTNAME:-web-lab}"
INSTANCE_USER="${INSTANCE_USER:-yeast}"
INSTANCE_MEMORY="${INSTANCE_MEMORY:-1024}"
INSTANCE_CPUS="${INSTANCE_CPUS:-1}"
INSTANCE_DISK_SIZE="${INSTANCE_DISK_SIZE:-20G}"
INSTANCE_SSH_PORT="${INSTANCE_SSH_PORT:-2205}"

if [[ -z "${BIN_PATH}" ]]; then
  echo "usage: $0 /absolute/or/relative/path/to/yeast-binary" >&2
  exit 2
fi

if [[ ! -x "${BIN_PATH}" ]]; then
  echo "error: binary is not executable: ${BIN_PATH}" >&2
  exit 2
fi

if command -v realpath >/dev/null 2>&1; then
  BIN_PATH="$(realpath "${BIN_PATH}")"
fi

if command -v tput >/dev/null 2>&1 && [[ -t 1 ]]; then
  BOLD="$(tput bold)"
  DIM="$(tput dim)"
  RED="$(tput setaf 1)"
  GREEN="$(tput setaf 2)"
  YELLOW="$(tput setaf 3)"
  BLUE="$(tput setaf 4)"
  RESET="$(tput sgr0)"
else
  BOLD=""
  DIM=""
  RED=""
  GREEN=""
  YELLOW=""
  BLUE=""
  RESET=""
fi

RESULTS=()
PORT_VALUE=""

section() {
  printf "\n%s==> %s%s\n" "${BLUE}${BOLD}" "$1" "${RESET}"
}

ok() {
  printf "%s[ok]%s %s\n" "${GREEN}" "${RESET}" "$1"
}

warn() {
  printf "%s[warn]%s %s\n" "${YELLOW}" "${RESET}" "$1"
}

fail() {
  printf "%s[fail]%s %s\n" "${RED}" "${RESET}" "$1" >&2
  exit 1
}

record_pass() {
  RESULTS+=("PASS | $1")
}

record_fail() {
  RESULTS+=("FAIL | $1")
}

run_step() {
  local label="$1"
  shift
  section "${label}"
  printf "%s$ %s%s\n" "${DIM}" "$*" "${RESET}"
  if "$@"; then
    record_pass "${label}"
  else
    record_fail "${label}"
    fail "${label}"
  fi
}

extract_port_from_status_json() {
  local json="$1"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; data=json.load(sys.stdin); instances=data.get("data",{}).get("Instances",[]); print(instances[0].get("SSHPort","")) if instances else print("")' <<<"${json}"
    return
  fi
  if command -v jq >/dev/null 2>&1; then
    jq -r '.data.Instances[0].SSHPort // empty' <<<"${json}"
    return
  fi
  printf "%s" "${json}" | sed -n 's/.*"SSHPort":\([0-9][0-9]*\).*/\1/p' | head -n1
}

run_capture() {
  local label="$1"
  shift
  section "${label}"
  printf "%s$ %s%s\n" "${DIM}" "$*" "${RESET}"
  "$@"
}

write_config() {
  cat >"${WORKDIR}/yeast.yaml" <<EOF
version: 1
instances:
  - name: ${INSTANCE_NAME}
    hostname: ${INSTANCE_HOSTNAME}
    image: ${IMAGE_NAME}
    memory: ${INSTANCE_MEMORY}
    cpus: ${INSTANCE_CPUS}
    disk_size: ${INSTANCE_DISK_SIZE}
    ssh_port: ${INSTANCE_SSH_PORT}
    user: ${INSTANCE_USER}
    sudo: none
EOF
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" == *"${needle}"* ]]; then
    ok "${label}"
  else
    fail "${label}: expected to find '${needle}'"
  fi
}

print_summary() {
  section "Summary"
  printf "%sBinary:%s %s\n" "${BOLD}" "${RESET}" "${BIN_PATH}"
  printf "%sWorkdir:%s %s\n" "${BOLD}" "${RESET}" "${WORKDIR}"
  printf "%sRequested ssh_port:%s %s\n" "${BOLD}" "${RESET}" "${INSTANCE_SSH_PORT}"
  printf "%sExpected hostname:%s %s\n" "${BOLD}" "${RESET}" "${INSTANCE_HOSTNAME}"
  printf "\n"
  printf "%sManual smoke test results%s\n" "${BOLD}" "${RESET}"
  printf "%s\n" "${RESULTS[@]}"
}

section "Resolve binary"
ok "${BIN_PATH}"

run_capture "Version" "${BIN_PATH}" version
run_capture "Doctor" "${BIN_PATH}" doctor

section "Prepare clean project"
rm -rf "${WORKDIR}"
mkdir -p "${WORKDIR}"
cd "${WORKDIR}"
ok "using ${WORKDIR}"

run_capture "Init" "${BIN_PATH}" init

section "Write v0.2.0 config"
write_config
cat "${WORKDIR}/yeast.yaml"
record_pass "Write config"

run_capture "Pull image" "${BIN_PATH}" pull "${IMAGE_NAME}"
run_capture "Start VM" "${BIN_PATH}" up

STATUS_TEXT="$("${BIN_PATH}" status)"
section "Status"
printf "%s\n" "${STATUS_TEXT}"
assert_contains "${STATUS_TEXT}" "${INSTANCE_NAME}" "status includes instance"
assert_contains "${STATUS_TEXT}" "${INSTANCE_SSH_PORT}" "status includes requested ssh port"
record_pass "Status"

STATUS_JSON="$("${BIN_PATH}" status --json)"
section "Status JSON"
printf "%s\n" "${STATUS_JSON}"
PORT_VALUE="$(extract_port_from_status_json "${STATUS_JSON}")"
if [[ -z "${PORT_VALUE}" ]]; then
  fail "status json did not expose SSHPort"
fi
if [[ "${PORT_VALUE}" != "${INSTANCE_SSH_PORT}" ]]; then
  fail "expected SSHPort ${INSTANCE_SSH_PORT}, got ${PORT_VALUE}"
fi
ok "status json reports ssh port ${PORT_VALUE}"
record_pass "Status JSON"

section "Direct SSH checks"
SSH_BASE=(ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p "${INSTANCE_SSH_PORT}" "${INSTANCE_USER}@127.0.0.1")
printf "%s$ %s hostname%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
HOSTNAME_OUTPUT="$("${SSH_BASE[@]}" hostname)"
printf "%s\n" "${HOSTNAME_OUTPUT}"
if [[ "${HOSTNAME_OUTPUT}" != "${INSTANCE_HOSTNAME}" ]]; then
  fail "expected guest hostname ${INSTANCE_HOSTNAME}, got ${HOSTNAME_OUTPUT}"
fi
ok "guest hostname is ${INSTANCE_HOSTNAME}"

printf "%s$ %s whoami%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
WHOAMI_OUTPUT="$("${SSH_BASE[@]}" whoami)"
printf "%s\n" "${WHOAMI_OUTPUT}"
if [[ "${WHOAMI_OUTPUT}" != "${INSTANCE_USER}" ]]; then
  fail "expected guest user ${INSTANCE_USER}, got ${WHOAMI_OUTPUT}"
fi
ok "guest user is ${INSTANCE_USER}"
record_pass "Direct SSH checks"

run_capture "Stop VM" "${BIN_PATH}" down

STATUS_AFTER_DOWN="$("${BIN_PATH}" status)"
section "Status after down"
printf "%s\n" "${STATUS_AFTER_DOWN}"
assert_contains "${STATUS_AFTER_DOWN}" "stopped" "status after down is stopped"
record_pass "Status after down"

run_capture "Restart VM" "${BIN_PATH}" up
STATUS_AFTER_RESTART="$("${BIN_PATH}" status)"
section "Status after restart"
printf "%s\n" "${STATUS_AFTER_RESTART}"
assert_contains "${STATUS_AFTER_RESTART}" "${INSTANCE_SSH_PORT}" "status after restart keeps requested ssh port"
record_pass "Status after restart"

run_capture "Destroy VM" "${BIN_PATH}" destroy
FINAL_STATUS_JSON="$("${BIN_PATH}" status --json)"
section "Final status JSON"
printf "%s\n" "${FINAL_STATUS_JSON}"
if [[ "${FINAL_STATUS_JSON}" == *'"Instances":[]'* ]]; then
  ok "no instances remain after destroy"
else
  warn "status json still contains instance data after destroy"
fi
record_pass "Destroy"

print_summary
