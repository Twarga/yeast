#!/usr/bin/env bash
set -euo pipefail

BIN_PATH="${1:-}"
WORKDIR="${WORKDIR:-/tmp/yeast-v100-test}"
IMAGE_NAME="${IMAGE_NAME:-ubuntu-24.04}"
INSTANCE_NAME="${INSTANCE_NAME:-web}"
INSTANCE_HOSTNAME="${INSTANCE_HOSTNAME:-caddy-lab}"
INSTANCE_USER="${INSTANCE_USER:-yeast}"
INSTANCE_MEMORY="${INSTANCE_MEMORY:-1024}"
INSTANCE_CPUS="${INSTANCE_CPUS:-1}"
INSTANCE_DISK_SIZE="${INSTANCE_DISK_SIZE:-20G}"
INSTANCE_SSH_PORT="${INSTANCE_SSH_PORT:-2205}"
RESTORE_SSH_PORT="${RESTORE_SSH_PORT:-$((INSTANCE_SSH_PORT + 1))}"
ATTACKER_NAME="${ATTACKER_NAME:-attacker}"
TARGET_NAME="${TARGET_NAME:-target}"
ATTACKER_HOSTNAME="${ATTACKER_HOSTNAME:-attacker-lab}"
TARGET_HOSTNAME="${TARGET_HOSTNAME:-target-lab}"
ATTACKER_SSH_PORT="${ATTACKER_SSH_PORT:-2305}"
TARGET_SSH_PORT="${TARGET_SSH_PORT:-2306}"
LAB_NETWORK_NAME="${LAB_NETWORK_NAME:-lab}"
LAB_CIDR="${LAB_CIDR:-10.10.10.0/24}"
ATTACKER_LAB_IP="${ATTACKER_LAB_IP:-10.10.10.10}"
TARGET_LAB_IP="${TARGET_LAB_IP:-10.10.10.20}"
TEST_MODE="${TEST_MODE:-full}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROVISION_SENTINEL_ONE="Yeast v0.4 provisioning works."
PROVISION_SENTINEL_TWO="Yeast reprovisioned content."
SNAPSHOT_NAME="${SNAPSHOT_NAME:-clean}"
SNAPSHOT_DESCRIPTION="${SNAPSHOT_DESCRIPTION:-Provisioned reset baseline}"
SSH_RETRY_CONNECT_TIMEOUT="${SSH_RETRY_CONNECT_TIMEOUT:-5}"
GUEST_CONTROL_UPLOAD_FILE="${GUEST_CONTROL_UPLOAD_FILE:-guest-control-upload.txt}"
GUEST_CONTROL_UPLOAD_SENTINEL="${GUEST_CONTROL_UPLOAD_SENTINEL:-guest control upload works}"

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
POSITIVE_DIR="${WORKDIR}/happy-path"
NETWORK_DIR="${WORKDIR}/network-path"
TEMPLATE_DIR="${WORKDIR}/template-path"
LABSBACKERY_DIR="${WORKDIR}/labsbackery-package-path"
NEGATIVE_ROOT="${WORKDIR}/negative-cases"

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

require_test_mode() {
  case "${TEST_MODE}" in
    full|positive|negative|templates) ;;
    *)
      fail "invalid TEST_MODE=${TEST_MODE} (expected full, positive, negative, or templates)"
      ;;
  esac
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
    python3 -c 'import json,sys; data=json.load(sys.stdin); instances=data.get("data",{}).get("instances",[]); print(instances[0].get("ssh_port","")) if instances else print("")' <<<"${json}"
    return
  fi
  if command -v jq >/dev/null 2>&1; then
    jq -r '.data.instances[0].ssh_port // empty' <<<"${json}"
    return
  fi
  printf "%s" "${json}" | sed -n 's/.*"ssh_port":\([0-9][0-9]*\).*/\1/p' | head -n1
}

run_capture() {
  local label="$1"
  shift
  section "${label}"
  printf "%s$ %s%s\n" "${DIM}" "$*" "${RESET}"
  "$@"
}

run_capture_with_port_retry() {
  local label="$1"
  local attempts="$2"
  shift 2

  section "${label}"

  local output=""
  local status=0
  local attempt=1
  while (( attempt <= attempts )); do
    printf "%s$ %s%s\n" "${DIM}" "$*" "${RESET}"
    set +e
    output="$("$@" 2>&1)"
    status=$?
    set -e
    printf "%s\n" "${output}"
    if [[ ${status} -eq 0 ]]; then
      record_pass "${label}"
      return 0
    fi
    if [[ "${output}" == *"already bound on the host"* ]] && (( attempt < attempts )); then
      warn "${label}: requested ssh_port still settling on host, retrying (${attempt}/${attempts})"
      sleep 1
      ((attempt++))
      continue
    fi
    record_fail "${label}"
    fail "${label}"
  done
}

run_ssh_capture_with_retry() {
  local label="$1"
  local attempts="$2"
  shift 2

  printf "\n%s==> %s%s\n" "${BLUE}${BOLD}" "${label}" "${RESET}" >&2

  local output=""
  local stderr_file=""
  local status=0
  local attempt=1
  stderr_file="$(mktemp)"
  while (( attempt <= attempts )); do
    printf "%s$ %s%s\n" "${DIM}" "$*" "${RESET}" >&2
    set +e
    output="$("$@" 2>"${stderr_file}")"
    status=$?
    set -e
    if [[ -s "${stderr_file}" ]]; then
      cat "${stderr_file}" >&2
      : >"${stderr_file}"
    fi
    if [[ ${status} -eq 0 ]]; then
      rm -f "${stderr_file}"
      printf "%s" "${output}"
      return 0
    fi
    if (( attempt < attempts )); then
      printf "%s[warn]%s %s\n" "${YELLOW}" "${RESET}" "${label}: ssh command not ready yet, retrying (${attempt}/${attempts})" >&2
      sleep 1
      ((attempt++))
      continue
    fi
    fail "${label}: ssh command failed after ${attempts} attempts"
  done
  rm -f "${stderr_file}"
}

write_config() {
  local target_dir="$1"
  cat >"${target_dir}/yeast.yaml" <<EOF
version: 1
provision:
  packages:
    - caddy
  files:
    - source: ./site/index.html
      destination: /home/${INSTANCE_USER}/site/index.html
      permissions: "0644"
    - source: ./site/Caddyfile
      destination: /home/${INSTANCE_USER}/site/Caddyfile
      permissions: "0644"
  shell:
    - sudo install -D -m 0644 /home/${INSTANCE_USER}/site/index.html /var/www/html/index.html
    - sudo install -D -m 0644 /home/${INSTANCE_USER}/site/Caddyfile /etc/caddy/Caddyfile
    - sudo systemctl enable caddy
    - sudo systemctl restart caddy
instances:
  - name: ${INSTANCE_NAME}
    hostname: ${INSTANCE_HOSTNAME}
    image: ${IMAGE_NAME}
    memory: ${INSTANCE_MEMORY}
    cpus: ${INSTANCE_CPUS}
    disk_size: ${INSTANCE_DISK_SIZE}
    ssh_port: ${INSTANCE_SSH_PORT}
    user: ${INSTANCE_USER}
    sudo: nopasswd
EOF
}

write_network_config() {
  local target_dir="$1"
  cat >"${target_dir}/yeast.yaml" <<EOF
version: 1
networks:
  - name: ${LAB_NETWORK_NAME}
    cidr: ${LAB_CIDR}
instances:
  - name: ${ATTACKER_NAME}
    hostname: ${ATTACKER_HOSTNAME}
    image: ${IMAGE_NAME}
    memory: ${INSTANCE_MEMORY}
    cpus: ${INSTANCE_CPUS}
    ssh_port: ${ATTACKER_SSH_PORT}
    user: ${INSTANCE_USER}
    sudo: nopasswd
    networks:
      - name: ${LAB_NETWORK_NAME}
        ipv4: ${ATTACKER_LAB_IP}
  - name: ${TARGET_NAME}
    hostname: ${TARGET_HOSTNAME}
    image: ${IMAGE_NAME}
    memory: ${INSTANCE_MEMORY}
    cpus: ${INSTANCE_CPUS}
    ssh_port: ${TARGET_SSH_PORT}
    user: ${INSTANCE_USER}
    sudo: nopasswd
    networks:
      - name: ${LAB_NETWORK_NAME}
        ipv4: ${TARGET_LAB_IP}
EOF
}

prepare_caddy_example() {
  local target_dir="$1"
  mkdir -p "${target_dir}/site"
  cp "${REPO_ROOT}/examples/caddy-single-vm/site/index.html" "${target_dir}/site/index.html"
  cp "${REPO_ROOT}/examples/caddy-single-vm/site/Caddyfile" "${target_dir}/site/Caddyfile"
}

ssh_args_for_port() {
  local port="$1"
  printf '%s\n' \
    ssh \
    -o "BatchMode=yes" \
    -o "ConnectTimeout=${SSH_RETRY_CONNECT_TIMEOUT}" \
    -o "ConnectionAttempts=1" \
    -o "LogLevel=ERROR" \
    -o "StrictHostKeyChecking=no" \
    -o "UserKnownHostsFile=/dev/null" \
    -p "${port}" \
    "${INSTANCE_USER}@127.0.0.1"
}

write_file() {
  local path="$1"
  shift
  printf '%s\n' "$@" >"${path}"
}

rewrite_ssh_port() {
  local target_dir="$1"
  local new_port="$2"
  python3 - <<'PY' "${target_dir}/yeast.yaml" "${new_port}"
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
port = sys.argv[2]
text = path.read_text(encoding="utf-8")
updated, count = re.subn(r'(^\s*ssh_port:\s*)\d+\s*$', r'\g<1>' + port, text, flags=re.MULTILINE)
if count != 1:
    raise SystemExit(f"expected exactly one ssh_port entry in {path}, found {count}")
path.write_text(updated, encoding="utf-8")
PY
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

json_extract() {
  local query="$1"
  local payload="$2"
  if command -v python3 >/dev/null 2>&1; then
    JSON_QUERY="${query}" python3 -c '
import json
import os
import sys

payload = json.load(sys.stdin)
query = os.environ["JSON_QUERY"]

current = payload
for token in query.split("."):
    if token == "":
        continue
    if isinstance(current, dict):
        current = current.get(token)
    else:
        current = None
        break

if current is None:
    print("")
elif isinstance(current, bool):
    print("true" if current else "false")
else:
    print(current)
' <<<"${payload}"
    return
  fi

  case "${query}" in
    ok)
      printf "%s" "${payload}" | sed -n 's/.*"ok":[[:space:]]*\(true\|false\).*/\1/p' | head -n1
      ;;
    error.code)
      printf "%s" "${payload}" | sed -n 's/.*"code":"\([^"]*\)".*/\1/p' | head -n1
      ;;
    error.message)
      printf "%s" "${payload}" | sed -n 's/.*"message":"\([^"]*\)".*/\1/p' | head -n1
      ;;
    *)
      printf ""
      ;;
  esac
}

assert_json_error_code() {
  local payload="$1"
  local expected_code="$2"
  local label="$3"
  local ok_value
  local code_value

  ok_value="$(json_extract "ok" "${payload}")"
  code_value="$(json_extract "error.code" "${payload}")"

  if [[ "${ok_value}" != "false" ]]; then
    fail "${label}: expected ok=false payload"
  fi
  if [[ "${code_value}" != "${expected_code}" ]]; then
    fail "${label}: expected error.code=${expected_code}, got ${code_value:-<empty>}"
  fi
  ok "${label}: error.code=${code_value}"
}

run_expect_json_error() {
  local label="$1"
  local expected_code="$2"
  shift 2

  section "${label}"
  printf "%s$ %s%s\n" "${DIM}" "$*" "${RESET}"

  local output=""
  local status=0
  set +e
  output="$("$@" 2>&1)"
  status=$?
  set -e

  printf "%s\n" "${output}"
  if [[ ${status} -eq 0 ]]; then
    record_fail "${label}"
    fail "${label}: expected command failure"
  fi

  assert_json_error_code "${output}" "${expected_code}" "${label}"
  record_pass "${label}"
}

run_expect_json_error_in_dir() {
  local label="$1"
  local expected_code="$2"
  local dir="$3"
  shift 3

  section "${label}"
  printf "%s$ (cd %s && %s)%s\n" "${DIM}" "${dir}" "$*" "${RESET}"

  local output=""
  local status=0
  set +e
  output="$(
    cd "${dir}"
    "$@" 2>&1
  )"
  status=$?
  set -e

  printf "%s\n" "${output}"
  if [[ ${status} -eq 0 ]]; then
    record_fail "${label}"
    fail "${label}: expected command failure"
  fi

  assert_json_error_code "${output}" "${expected_code}" "${label}"
  record_pass "${label}"
}

new_case_dir() {
  local name="$1"
  local dir="${NEGATIVE_ROOT}/${name}"
  rm -rf "${dir}"
  mkdir -p "${dir}"
  printf "%s" "${dir}"
}

init_case_project() {
  local dir="$1"
  mkdir -p "${dir}"
  (cd "${dir}" && "${BIN_PATH}" init >/dev/null)
}

project_id_from_dir() {
  local dir="$1"
  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY' "${dir}/.yeast/project.json"
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as handle:
    print(json.load(handle)["id"])
PY
    return
  fi
  sed -n 's/.*"id":"\([^"]*\)".*/\1/p' "${dir}/.yeast/project.json" | head -n1
}

run_template_suite() {
  section "List built-in templates"
  TEMPLATE_LIST_JSON="$("${BIN_PATH}" init --list-templates --json)"
  printf "%s\n" "${TEMPLATE_LIST_JSON}"
  assert_contains "${TEMPLATE_LIST_JSON}" '"name":"ubuntu-basic"' "template list includes ubuntu-basic"
  assert_contains "${TEMPLATE_LIST_JSON}" '"name":"caddy-single-vm"' "template list includes caddy-single-vm"
  assert_contains "${TEMPLATE_LIST_JSON}" '"name":"two-vm-lab"' "template list includes two-vm-lab"
  record_pass "List built-in templates"

  section "Initialize built-in Caddy template"
  rm -rf "${TEMPLATE_DIR}"
  mkdir -p "${TEMPLATE_DIR}"
  cd "${TEMPLATE_DIR}"
  "${BIN_PATH}" init --template caddy-single-vm
  for generated_file in yeast.yaml README.md site/Caddyfile site/index.html .yeast/project.json; do
    if [[ ! -f "${TEMPLATE_DIR}/${generated_file}" ]]; then
      fail "expected template file ${generated_file}"
    fi
  done
  ok "template generated expected project files"
  assert_contains "$(cat "${TEMPLATE_DIR}/yeast.yaml")" "caddy" "template config includes caddy provisioning"
  assert_contains "$(cat "${TEMPLATE_DIR}/README.md")" "yeast init --template caddy-single-vm" "template README documents template init"
  record_pass "Initialize built-in Caddy template"
}

run_labsbackery_package_suite() {
  section "Materialize LabsBakery example package"
  rm -rf "${LABSBACKERY_DIR}"
  mkdir -p "${LABSBACKERY_DIR}"
  cd "${LABSBACKERY_DIR}"

  "${BIN_PATH}" init --template "${REPO_ROOT}/examples/labsbackery-attacker-target-basic"

  for generated_file in yeast.yaml lab.yaml README.md scenario/instructions.md scenario/checks.yaml files/target/flag.txt .yeast/project.json; do
    if [[ ! -f "${LABSBACKERY_DIR}/${generated_file}" ]]; then
      fail "expected LabsBakery package file ${generated_file}"
    fi
  done

  assert_contains "$(cat "${LABSBACKERY_DIR}/lab.yaml")" "schema: labsbakery.lab.v1" "lab package includes LabsBakery schema"
  assert_contains "$(cat "${LABSBACKERY_DIR}/yeast.yaml")" "${ATTACKER_NAME}" "lab package config includes attacker"
  assert_contains "$(cat "${LABSBACKERY_DIR}/yeast.yaml")" "${TARGET_NAME}" "lab package config includes target"
  assert_contains "$(cat "${LABSBACKERY_DIR}/scenario/checks.yaml")" "attacker-reaches-target-ssh" "lab package checks include attacker reachability check"
  assert_contains "$(cat "${LABSBACKERY_DIR}/files/target/flag.txt")" "labsbackery-ready" "lab package includes target marker file"
  record_pass "Materialize LabsBakery example package"
}

run_positive_suite() {
  section "Prepare clean project"
  rm -rf "${POSITIVE_DIR}"
  mkdir -p "${POSITIVE_DIR}"
  cd "${POSITIVE_DIR}"
  ok "using ${POSITIVE_DIR}"

  run_capture "Init Caddy template" "${BIN_PATH}" init --template caddy-single-vm
  if [[ "${INSTANCE_SSH_PORT}" != "2205" ]]; then
    rewrite_ssh_port "${POSITIVE_DIR}" "${INSTANCE_SSH_PORT}"
  fi
  section "Generated template config"
  cat "${POSITIVE_DIR}/yeast.yaml"
  assert_contains "$(cat "${POSITIVE_DIR}/yeast.yaml")" "caddy" "generated template config includes caddy"
  record_pass "Generated template config"

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
    fail "status json did not expose ssh_port"
  fi
  if [[ "${PORT_VALUE}" != "${INSTANCE_SSH_PORT}" ]]; then
    fail "expected ssh_port ${INSTANCE_SSH_PORT}, got ${PORT_VALUE}"
  fi
  ok "status json reports ssh port ${PORT_VALUE}"
  assert_contains "${STATUS_JSON}" '"provisioning_status":"provisioned"' "status json reports provisioned state"
  assert_contains "${STATUS_JSON}" "\"user\":\"${INSTANCE_USER}\"" "status json reports guest user"
  record_pass "Status JSON"

  local active_ssh_port="${INSTANCE_SSH_PORT}"

  section "Direct SSH checks"
  mapfile -t SSH_BASE < <(ssh_args_for_port "${active_ssh_port}")
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

  printf "%s$ %s sudo systemctl is-active caddy%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
  CADDY_STATUS="$("${SSH_BASE[@]}" sudo systemctl is-active caddy)"
  printf "%s\n" "${CADDY_STATUS}"
  if [[ "${CADDY_STATUS}" != "active" ]]; then
    fail "expected caddy service active, got ${CADDY_STATUS}"
  fi
  ok "caddy service is active"

  printf "%s$ %s curl -fsS http://127.0.0.1%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
  CADDY_PAGE="$("${SSH_BASE[@]}" curl -fsS http://127.0.0.1)"
  printf "%s\n" "${CADDY_PAGE}"
  assert_contains "${CADDY_PAGE}" "${PROVISION_SENTINEL_ONE}" "caddy serves initial provisioned content"
  record_pass "Direct SSH checks"

  section "Guest control commands"

  EXEC_JSON="$("${BIN_PATH}" --json exec "${INSTANCE_NAME}" -- whoami)"
  printf "%s\n" "${EXEC_JSON}"
  if [[ "$(json_extract "data.instance" "${EXEC_JSON}")" != "${INSTANCE_NAME}" ]]; then
    fail "exec json did not report instance ${INSTANCE_NAME}"
  fi
  if [[ "$(json_extract "data.run.exit_code" "${EXEC_JSON}")" != "0" ]]; then
    fail "exec json did not report exit code 0"
  fi
  if [[ "$(json_extract "data.run.stdout" "${EXEC_JSON}")" != "${INSTANCE_USER}" ]]; then
    fail "exec json did not report stdout ${INSTANCE_USER}"
  fi
  ok "exec returns structured result"

  printf '%s\n' "${GUEST_CONTROL_UPLOAD_SENTINEL}" > "${POSITIVE_DIR}/${GUEST_CONTROL_UPLOAD_FILE}"
  run_capture "Copy host file to guest" \
    "${BIN_PATH}" copy "${INSTANCE_NAME}" --to-guest "./${GUEST_CONTROL_UPLOAD_FILE}" "/home/${INSTANCE_USER}/${GUEST_CONTROL_UPLOAD_FILE}"

  EXEC_CAT_JSON="$("${BIN_PATH}" --json exec "${INSTANCE_NAME}" -- cat "/home/${INSTANCE_USER}/${GUEST_CONTROL_UPLOAD_FILE}")"
  printf "%s\n" "${EXEC_CAT_JSON}"
  assert_contains "${EXEC_CAT_JSON}" "${GUEST_CONTROL_UPLOAD_SENTINEL}" "copied host file is readable in guest"

  mkdir -p "${POSITIVE_DIR}/downloads"
  run_capture "Copy guest file to host" \
    "${BIN_PATH}" copy "${INSTANCE_NAME}" --from-guest "/home/${INSTANCE_USER}/${GUEST_CONTROL_UPLOAD_FILE}" "./downloads/copied-back.txt"
  COPIED_BACK_CONTENT="$(cat "${POSITIVE_DIR}/downloads/copied-back.txt")"
  if [[ "${COPIED_BACK_CONTENT}" != "${GUEST_CONTROL_UPLOAD_SENTINEL}" ]]; then
    fail "copied-back guest file content mismatch"
  fi
  ok "guest file copied back to host"

  INSPECT_JSON="$("${BIN_PATH}" --json inspect "${INSTANCE_NAME}")"
  printf "%s\n" "${INSPECT_JSON}"
  if [[ "$(json_extract "data.instance.name" "${INSPECT_JSON}")" != "${INSTANCE_NAME}" ]]; then
    fail "inspect json did not report instance ${INSTANCE_NAME}"
  fi
  if [[ "$(json_extract "data.instance.provisioning_status" "${INSPECT_JSON}")" != "provisioned" ]]; then
    fail "inspect json did not report provisioned status"
  fi
  if [[ "$(json_extract "data.instance.user" "${INSPECT_JSON}")" != "${INSTANCE_USER}" ]]; then
    fail "inspect json did not report user ${INSTANCE_USER}"
  fi
  ok "inspect returns instance detail"

  LOGS_JSON="$("${BIN_PATH}" --json logs "${INSTANCE_NAME}" --tail 20)"
  printf "%s\n" "${LOGS_JSON}"
  assert_contains "${LOGS_JSON}" "vm.log" "logs json includes vm log path"
  ok "logs exposes vm log access"
  record_pass "Guest control commands"

  section "Update content and rerun provisioning"
  cat >"${POSITIVE_DIR}/site/index.html" <<EOF
<!doctype html>
<html lang="en">
  <body>
    <h1>${PROVISION_SENTINEL_TWO}</h1>
  </body>
</html>
EOF
  printf "%s$ %s provision %s%s\n" "${DIM}" "${BIN_PATH}" "${INSTANCE_NAME}" "${RESET}"
  "${BIN_PATH}" provision "${INSTANCE_NAME}"
  printf "%s$ %s curl -fsS http://127.0.0.1%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
  REPROVISIONED_PAGE="$("${SSH_BASE[@]}" curl -fsS http://127.0.0.1)"
  printf "%s\n" "${REPROVISIONED_PAGE}"
  assert_contains "${REPROVISIONED_PAGE}" "${PROVISION_SENTINEL_TWO}" "yeast provision updates guest content"
  record_pass "Provision rerun"

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

  STATUS_JSON_AFTER_RESTART="$("${BIN_PATH}" status --json)"
  assert_contains "${STATUS_JSON_AFTER_RESTART}" '"provisioning_status":"provisioned"' "status json after restart remains provisioned"

  printf "%s$ %s curl -fsS http://127.0.0.1%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
  RESTARTED_PAGE="$("${SSH_BASE[@]}" curl -fsS http://127.0.0.1)"
  printf "%s\n" "${RESTARTED_PAGE}"
  assert_contains "${RESTARTED_PAGE}" "${PROVISION_SENTINEL_TWO}" "restarted guest still serves provisioned content"
  record_pass "Restarted service check"

  run_capture "Stop before snapshot" "${BIN_PATH}" down

  run_capture "Create snapshot" "${BIN_PATH}" snapshot "${INSTANCE_NAME}" "${SNAPSHOT_NAME}" --description "${SNAPSHOT_DESCRIPTION}"

  SNAPSHOTS_TEXT="$("${BIN_PATH}" snapshots "${INSTANCE_NAME}")"
  section "Snapshots"
  printf "%s\n" "${SNAPSHOTS_TEXT}"
  assert_contains "${SNAPSHOTS_TEXT}" "${SNAPSHOT_NAME}" "snapshot list includes snapshot name"
  assert_contains "${SNAPSHOTS_TEXT}" "${SNAPSHOT_DESCRIPTION}" "snapshot list includes snapshot description"
  record_pass "Snapshots"

  run_capture "Start VM for break phase" "${BIN_PATH}" up

  section "Break guest content"
  printf "%s$ %s sudo rm -f /var/www/html/index.html%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
  "${SSH_BASE[@]}" sudo rm -f /var/www/html/index.html
  printf "%s$ %s test ! -f /var/www/html/index.html%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
  "${SSH_BASE[@]}" test ! -f /var/www/html/index.html
  ok "guest content removed after snapshot"
  record_pass "Break guest content"

  run_capture "Stop before restore" "${BIN_PATH}" down
  run_capture "Restore snapshot" "${BIN_PATH}" restore "${INSTANCE_NAME}" "${SNAPSHOT_NAME}"
  section "Switch restored host ssh port"
  rewrite_ssh_port "${POSITIVE_DIR}" "${RESTORE_SSH_PORT}"
  cat "${POSITIVE_DIR}/yeast.yaml"
  active_ssh_port="${RESTORE_SSH_PORT}"
  mapfile -t SSH_BASE < <(ssh_args_for_port "${active_ssh_port}")
  record_pass "Switch restored host ssh port"
  run_capture "Start restored VM" "${BIN_PATH}" up

  section "Verify restored content"
  printf "%s$ %s curl -fsS http://127.0.0.1%s\n" "${DIM}" "${SSH_BASE[*]}" "${RESET}"
  RESTORED_PAGE="$("${SSH_BASE[@]}" curl -fsS http://127.0.0.1)"
  printf "%s\n" "${RESTORED_PAGE}"
  assert_contains "${RESTORED_PAGE}" "${PROVISION_SENTINEL_TWO}" "restored guest serves snapshotted content"
  record_pass "Restore"

  run_capture "Stop before snapshot delete" "${BIN_PATH}" down
  run_capture "Delete snapshot" "${BIN_PATH}" delete-snapshot "${INSTANCE_NAME}" "${SNAPSHOT_NAME}"

  SNAPSHOTS_AFTER_DELETE="$("${BIN_PATH}" snapshots "${INSTANCE_NAME}")"
  section "Snapshots after delete"
  printf "%s\n" "${SNAPSHOTS_AFTER_DELETE}"
  if [[ "${SNAPSHOTS_AFTER_DELETE}" == *"${SNAPSHOT_NAME}"* ]]; then
    fail "snapshot list still contains deleted snapshot ${SNAPSHOT_NAME}"
  fi
  ok "snapshot deleted from list"
  record_pass "Delete snapshot"

  run_capture "Destroy VM" "${BIN_PATH}" destroy
  FINAL_STATUS_JSON="$("${BIN_PATH}" status --json)"
  section "Final status JSON"
  printf "%s\n" "${FINAL_STATUS_JSON}"
  if [[ "${FINAL_STATUS_JSON}" == *'"instances":[]'* ]]; then
    ok "no instances remain after destroy"
  else
    warn "status json still contains instance data after destroy"
  fi
  record_pass "Destroy"
}

run_network_suite() {
  section "Prepare two-VM lab project"
  rm -rf "${NETWORK_DIR}"
  mkdir -p "${NETWORK_DIR}"
  cd "${NETWORK_DIR}"
  ok "using ${NETWORK_DIR}"

  run_capture "Init two-VM lab" "${BIN_PATH}" init

  section "Write network config"
  write_network_config "${NETWORK_DIR}"
  cat "${NETWORK_DIR}/yeast.yaml"
  record_pass "Write network config"

  run_capture "Start two-VM lab" "${BIN_PATH}" up

  NETWORK_STATUS_TEXT="$("${BIN_PATH}" status)"
  section "Two-VM status"
  printf "%s\n" "${NETWORK_STATUS_TEXT}"
  assert_contains "${NETWORK_STATUS_TEXT}" "${ATTACKER_NAME}" "status includes attacker"
  assert_contains "${NETWORK_STATUS_TEXT}" "${TARGET_NAME}" "status includes target"
  assert_contains "${NETWORK_STATUS_TEXT}" "${ATTACKER_LAB_IP}" "status includes attacker lab ip"
  assert_contains "${NETWORK_STATUS_TEXT}" "${TARGET_LAB_IP}" "status includes target lab ip"
  record_pass "Two-VM status"

  NETWORK_STATUS_JSON="$("${BIN_PATH}" status --json)"
  section "Two-VM status JSON"
  printf "%s\n" "${NETWORK_STATUS_JSON}"
  assert_contains "${NETWORK_STATUS_JSON}" "\"lab_ip\":\"${ATTACKER_LAB_IP}\"" "status json includes attacker lab ip"
  assert_contains "${NETWORK_STATUS_JSON}" "\"lab_ip\":\"${TARGET_LAB_IP}\"" "status json includes target lab ip"
  record_pass "Two-VM status JSON"

  local attacker_ssh=()
  local target_ssh=()
  mapfile -t attacker_ssh < <(ssh_args_for_port "${ATTACKER_SSH_PORT}")
  mapfile -t target_ssh < <(ssh_args_for_port "${TARGET_SSH_PORT}")

  section "Guest-side lab NIC checks"
  local attacker_hostname
  attacker_hostname="$(run_ssh_capture_with_retry "Attacker hostname" 5 "${attacker_ssh[@]}" hostname)"
  if [[ "${attacker_hostname}" != "${ATTACKER_HOSTNAME}" ]]; then
    fail "unexpected attacker hostname ${attacker_hostname}"
  fi
  local target_hostname
  target_hostname="$(run_ssh_capture_with_retry "Target hostname" 5 "${target_ssh[@]}" hostname)"
  if [[ "${target_hostname}" != "${TARGET_HOSTNAME}" ]]; then
    fail "unexpected target hostname ${target_hostname}"
  fi
  local attacker_addr
  attacker_addr="$(run_ssh_capture_with_retry "Attacker lab NIC" 5 "${attacker_ssh[@]}" ip -4 addr show yeastlab0)"
  assert_contains "${attacker_addr}" "${ATTACKER_LAB_IP}" "attacker guest has lab ip"
  local target_addr
  target_addr="$(run_ssh_capture_with_retry "Target lab NIC" 5 "${target_ssh[@]}" ip -4 addr show yeastlab0)"
  assert_contains "${target_addr}" "${TARGET_LAB_IP}" "target guest has lab ip"
  record_pass "Guest-side lab NIC checks"

  section "Guest-to-guest lab TCP reachability"
  run_ssh_capture_with_retry "Attacker reaches target over lab network" 5 "${attacker_ssh[@]}" bash -lc "echo > /dev/tcp/${TARGET_LAB_IP}/22"
  ok "attacker reaches target SSH over lab network"
  run_ssh_capture_with_retry "Target reaches attacker over lab network" 5 "${target_ssh[@]}" bash -lc "echo > /dev/tcp/${ATTACKER_LAB_IP}/22"
  ok "target reaches attacker SSH over lab network"
  record_pass "Guest-to-guest lab TCP reachability"

  run_capture "Stop two-VM lab" "${BIN_PATH}" down
  run_capture "Destroy two-VM lab" "${BIN_PATH}" destroy
  record_pass "Destroy two-VM lab"
}

run_negative_suite() {
  local dir=""
  local project_id=""
  local state_path=""

  section "Prepare negative test root"
  rm -rf "${NEGATIVE_ROOT}"
  mkdir -p "${NEGATIVE_ROOT}"
  ok "using ${NEGATIVE_ROOT}"

  dir="$(new_case_dir "status-uninitialized")"
  run_expect_json_error_in_dir "Status fails in uninitialized directory" "failed_precondition" \
    "${dir}" "${BIN_PATH}" status --json

  dir="$(new_case_dir "init-conflict")"
  init_case_project "${dir}"
  run_expect_json_error_in_dir "Init fails on repeated project init" "conflict" \
    "${dir}" "${BIN_PATH}" init --json

  run_expect_json_error "Pull rejects unsupported image" "invalid_argument" \
    "${BIN_PATH}" pull does-not-exist --json

  dir="$(new_case_dir "init-missing-template")"
  run_expect_json_error_in_dir "Init rejects missing template" "not_found" \
    "${dir}" "${BIN_PATH}" init --template does-not-exist --json

  dir="$(new_case_dir "status-corrupt-metadata")"
  init_case_project "${dir}"
  write_file "${dir}/.yeast/project.json" '{"id":'
  run_expect_json_error_in_dir "Status classifies corrupt project metadata" "internal" \
    "${dir}" "${BIN_PATH}" status --json

  dir="$(new_case_dir "status-state-mismatch")"
  init_case_project "${dir}"
  project_id="$(project_id_from_dir "${dir}")"
  state_path="${HOME}/.yeast/projects/${project_id}/state.json"
  mkdir -p "$(dirname "${state_path}")"
  write_file "${state_path}" '{' '  "schema": "yeast.state.v1",' '  "project_id": "wrong-project",' '  "instances": {}' '}'
  run_expect_json_error_in_dir "Status classifies state project mismatch" "internal" \
    "${dir}" "${BIN_PATH}" status --json

  dir="$(new_case_dir "up-missing-config")"
  init_case_project "${dir}"
  rm -f "${dir}/yeast.yaml"
  run_expect_json_error_in_dir "Up fails when yeast.yaml is missing" "failed_precondition" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-invalid-disk-size")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'instances:' \
    '  - name: web' \
    '    image: ubuntu-24.04' \
    '    disk_size: not-a-size'
  run_expect_json_error_in_dir "Up rejects invalid disk_size" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-invalid-hostname")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'instances:' \
    '  - name: web' \
    '    image: ubuntu-24.04' \
    '    hostname: bad host!'
  run_expect_json_error_in_dir "Up rejects invalid hostname" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-invalid-ssh-port")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'instances:' \
    '  - name: web' \
    '    image: ubuntu-24.04' \
    '    ssh_port: 70000'
  run_expect_json_error_in_dir "Up rejects invalid ssh_port" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-duplicate-ssh-port")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'instances:' \
    '  - name: web' \
    '    image: ubuntu-24.04' \
    '    ssh_port: 2222' \
    '  - name: api' \
    '    image: ubuntu-24.04' \
    '    ssh_port: 2222'
  run_expect_json_error_in_dir "Up rejects duplicate requested ssh_port values" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-invalid-network-cidr")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'networks:' \
    '  - name: lab' \
    '    cidr: bad-cidr' \
    'instances:' \
    '  - name: web' \
    '    image: ubuntu-24.04'
  run_expect_json_error_in_dir "Up rejects invalid network cidr" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-unknown-network-reference")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'instances:' \
    '  - name: web' \
    '    image: ubuntu-24.04' \
    '    networks:' \
    '      - name: lab' \
    '        ipv4: 10.10.10.10'
  run_expect_json_error_in_dir "Up rejects unknown network reference" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-invalid-network-ipv4")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'networks:' \
    '  - name: lab' \
    '    cidr: 10.10.10.0/24' \
    'instances:' \
    '  - name: web' \
    '    image: ubuntu-24.04' \
    '    networks:' \
    '      - name: lab' \
    '        ipv4: not-an-ip'
  run_expect_json_error_in_dir "Up rejects invalid network ipv4" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-duplicate-network-ipv4")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'networks:' \
    '  - name: lab' \
    '    cidr: 10.10.10.0/24' \
    'instances:' \
    '  - name: attacker' \
    '    image: ubuntu-24.04' \
    '    networks:' \
    '      - name: lab' \
    '        ipv4: 10.10.10.10' \
    '  - name: target' \
    '    image: ubuntu-24.04' \
    '    networks:' \
    '      - name: lab' \
    '        ipv4: 10.10.10.10'
  run_expect_json_error_in_dir "Up rejects duplicate network ipv4 values" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json

  dir="$(new_case_dir "up-missing-provision-source")"
  init_case_project "${dir}"
  write_file "${dir}/yeast.yaml" \
    'version: 1' \
    'provision:' \
    '  files:' \
    '    - source: ./site/missing.txt' \
    '      destination: /tmp/missing.txt' \
    'instances:' \
    '  - name: web' \
    '    image: ubuntu-24.04'
  run_expect_json_error_in_dir "Up rejects missing provision source file" "invalid_argument" \
    "${dir}" "${BIN_PATH}" up --json
}

print_summary() {
  section "Summary"
  printf "%sBinary:%s %s\n" "${BOLD}" "${RESET}" "${BIN_PATH}"
  printf "%sWorkdir:%s %s\n" "${BOLD}" "${RESET}" "${WORKDIR}"
  printf "%sMode:%s %s\n" "${BOLD}" "${RESET}" "${TEST_MODE}"
  printf "%sRequested ssh_port:%s %s\n" "${BOLD}" "${RESET}" "${INSTANCE_SSH_PORT}"
  printf "%sRestored ssh_port:%s %s\n" "${BOLD}" "${RESET}" "${RESTORE_SSH_PORT}"
  printf "%sExpected hostname:%s %s\n" "${BOLD}" "${RESET}" "${INSTANCE_HOSTNAME}"
  printf "%sLab network:%s %s (%s, %s)\n" "${BOLD}" "${RESET}" "${LAB_NETWORK_NAME}" "${ATTACKER_LAB_IP}" "${TARGET_LAB_IP}"
  printf "%sSnapshot name:%s %s\n" "${BOLD}" "${RESET}" "${SNAPSHOT_NAME}"
  printf "\n"
  printf "%sManual smoke test results%s\n" "${BOLD}" "${RESET}"
  printf "%s\n" "${RESULTS[@]}"
}

require_test_mode

section "Resolve binary"
ok "${BIN_PATH}"

run_capture "Version" "${BIN_PATH}" version
run_capture "Doctor" "${BIN_PATH}" doctor

rm -rf "${WORKDIR}"
mkdir -p "${WORKDIR}"

if [[ "${TEST_MODE}" == "full" || "${TEST_MODE}" == "positive" || "${TEST_MODE}" == "templates" ]]; then
  run_template_suite
  run_labsbackery_package_suite
fi

if [[ "${TEST_MODE}" == "full" || "${TEST_MODE}" == "positive" ]]; then
  run_positive_suite
  run_network_suite
fi

if [[ "${TEST_MODE}" == "full" || "${TEST_MODE}" == "negative" ]]; then
  run_negative_suite
fi

print_summary
