#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/smoke/lib.sh
source "${SCRIPT_DIR}/lib.sh"

REQUESTED_LAB="${1:-all}"
YEAST_BIN_PATH="$(smoke_resolve_binary)"
SMOKE_ROOT_DIR="$(smoke_prepare_root)"
LOG_DIR="${SMOKE_ROOT_DIR}/logs"
WORK_DIR="${SMOKE_ROOT_DIR}/workspace"
mkdir -p "${LOG_DIR}" "${WORK_DIR}"

LAB_RESULTS=()
LAB_PROJECT_DIRS=()
CURRENT_LAB=""
CURRENT_LOG=""

register_project_dir() {
  LAB_PROJECT_DIRS+=("$1")
}

cleanup_current_lab() {
  local project_dir
  for project_dir in "${LAB_PROJECT_DIRS[@]:-}"; do
    smoke_cleanup_project "${YEAST_BIN_PATH}" "${project_dir}"
  done
}

on_exit() {
  local status="$1"
  if [[ ${status} -ne 0 && -n "${CURRENT_LAB}" ]]; then
    smoke_warn "${CURRENT_LAB} failed; attempting cleanup"
    cleanup_current_lab
    if [[ -n "${CURRENT_LOG}" ]]; then
      printf '\n---- %s log tail ----\n' "${CURRENT_LAB}" >&2
      smoke_last_lines "${CURRENT_LOG}" 80 >&2 || true
    fi
  fi
}
trap 'on_exit $?' EXIT

run_logged() {
  local log_path="$1"
  shift
  printf '$ %s\n' "$*" >>"${log_path}"
  "$@" 2>&1 | tee -a "${log_path}"
}

run_capture_logged() {
  local log_path="$1"
  shift
  printf '$ %s\n' "$*" >>"${log_path}"
  local output_file
  local status
  output_file="$(mktemp)"
  set +e
  "$@" >"${output_file}" 2>&1
  status=$?
  set -e
  cat "${output_file}" >>"${log_path}"
  local output
  output="$(cat "${output_file}")"
  rm -f "${output_file}"
  printf '%s' "${output}"
  return "${status}"
}

new_project_dir() {
  local name="$1"
  local dir="${WORK_DIR}/${CURRENT_LAB}/${name}"
  mkdir -p "${dir}"
  register_project_dir "${dir}"
  printf '%s\n' "${dir}"
}

copy_example_file() {
  local from="$1"
  local to="$2"
  mkdir -p "$(dirname "${to}")"
  cp "${SMOKE_REPO_ROOT}/${from}" "${to}"
}

lab01_first_vm() {
  local project_dir
  project_dir="$(new_project_dir "01-first-vm")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init"
  copy_example_file "examples/ubuntu-basic/yeast.yaml" "${project_dir}/yeast.yaml"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" doctor"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" pull ubuntu-24.04"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" status"
  local uname_out
  uname_out="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" exec web -- uname -a")"
  smoke_assert_contains "${uname_out}" "Linux"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy"
  smoke_assert_project_released "${project_dir}"
}

lab02_provisioning() {
  local project_dir
  project_dir="$(new_project_dir "02-provisioning")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init"
  copy_example_file "examples/caddy-single-vm/yeast.yaml" "${project_dir}/yeast.yaml"
  copy_example_file "examples/caddy-single-vm/site/index.html" "${project_dir}/site/index.html"
  copy_example_file "examples/caddy-single-vm/site/Caddyfile" "${project_dir}/site/Caddyfile"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up"
  local page
  page="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" exec web -- curl -fsS http://127.0.0.1")"
  smoke_assert_contains "${page}" "Yeast v0.4 provisioning works."
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy"
  smoke_assert_project_released "${project_dir}"
}

lab03_snapshot_restore() {
  local project_dir
  project_dir="$(new_project_dir "03-snapshots")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init --template caddy-single-vm"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" snapshot web clean --description 'Provisioned Caddy baseline'"
  local snapshots
  snapshots="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" snapshots web")"
  smoke_assert_contains "${snapshots}" "clean"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" exec web -- sudo rm -f /var/www/html/index.html"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" restore web clean"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up"
  local restored
  restored="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" exec web -- curl -fsS http://127.0.0.1")"
  smoke_assert_contains "${restored}" "Yeast v0.4 provisioning works."
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" delete-snapshot web clean"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy"
  smoke_assert_project_released "${project_dir}"
}

lab04_multi_vm_network() {
  local project_dir
  project_dir="$(new_project_dir "04-multi-vm-lab")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init"
  copy_example_file "examples/two-vm-lab/yeast.yaml" "${project_dir}/yeast.yaml"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up"
  local status
  status="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" status")"
  smoke_assert_contains "${status}" "10.10.10.10"
  smoke_assert_contains "${status}" "10.10.10.20"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" ssh attacker -- bash -lc 'ip -4 addr show yeastlab0 && ping -c 2 10.10.10.20'"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" ssh target -- bash -lc 'ip -4 addr show yeastlab0 && ping -c 2 10.10.10.10'"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy"
  smoke_assert_project_released "${project_dir}"
}

lab05_guest_control() {
  local project_dir
  project_dir="$(new_project_dir "05-guest-control")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init --template ubuntu-basic"
  printf 'smoke-artifact\n' >"${project_dir}/artifact.txt"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up --no-provision"
  local whoami_out
  whoami_out="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" exec web -- whoami")"
  smoke_assert_contains "${whoami_out}" "yeast"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" copy web --to-guest ./artifact.txt /home/yeast/artifact.txt"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" copy web --from-guest /home/yeast/artifact.txt ./artifact-out.txt"
  cmp "${project_dir}/artifact.txt" "${project_dir}/artifact-out.txt"
  local inspect_out
  inspect_out="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" inspect web")"
  smoke_assert_contains "${inspect_out}" "runtime_dir"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" logs web --tail 20"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy"
  smoke_assert_project_released "${project_dir}"
}

lab06_labsbackery_reset() {
  local project_dir
  project_dir="$(new_project_dir "06-labsbackery-lab")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init --template \"${SMOKE_REPO_ROOT}/examples/labsbackery-attacker-target-basic\""
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up --json --events > up.jsonl"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" status --json > status.json"
  local attacker_check
  attacker_check="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" exec attacker --json -- bash -lc 'echo > /dev/tcp/10.20.30.20/22'")"
  smoke_assert_contains "${attacker_check}" '"ok":true'
  local target_check
  target_check="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" exec target --json -- bash -lc 'test -f /home/yeast/labsbackery-target.txt && grep -q labsbackery-ready /home/yeast/labsbackery-target.txt'")"
  smoke_assert_contains "${target_check}" '"ok":true'
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down --json --events > down.jsonl"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" snapshot attacker clean --description 'Clean attacker baseline' --json > snapshot-attacker.json"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" snapshot target clean --description 'Clean target baseline' --json > snapshot-target.json"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" restore attacker clean --json --events > restore-attacker.jsonl"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" restore target clean --json --events > restore-target.jsonl"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up --json --events > up-restored.jsonl"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy --json --events > destroy.jsonl"
  smoke_assert_project_released "${project_dir}"
}

lab07_templates() {
  local template_root
  template_root="$(new_project_dir "07-templates")"
  local list_out
  list_out="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${template_root}\" && \"${YEAST_BIN_PATH}\" init --list-templates")"
  smoke_assert_contains "${list_out}" "ubuntu-basic"
  smoke_assert_contains "${list_out}" "caddy-single-vm"
  smoke_assert_contains "${list_out}" "two-vm-lab"
  mkdir -p "${template_root}/ubuntu-basic" "${template_root}/caddy-single-vm" "${template_root}/two-vm-lab"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${template_root}/ubuntu-basic\" && \"${YEAST_BIN_PATH}\" init --template ubuntu-basic"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${template_root}/caddy-single-vm\" && \"${YEAST_BIN_PATH}\" init --template caddy-single-vm"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${template_root}/two-vm-lab\" && \"${YEAST_BIN_PATH}\" init --template two-vm-lab"
  [[ -f "${template_root}/ubuntu-basic/yeast.yaml" ]] || smoke_die "ubuntu-basic template did not create yeast.yaml"
  [[ -d "${template_root}/caddy-single-vm/site" ]] || smoke_die "caddy template did not create site directory"
  smoke_assert_file_contains "${template_root}/two-vm-lab/yeast.yaml" "attacker"
  smoke_assert_file_contains "${template_root}/two-vm-lab/yeast.yaml" "target"
}

lab08_json_events() {
  local project_dir
  project_dir="$(new_project_dir "08-json-automation")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init --template ubuntu-basic"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up --json --events > up.jsonl"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" status --json > status.json"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down --json --events > down.jsonl"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy --json --events > destroy.jsonl"
  smoke_assert_file_contains "${project_dir}/up.jsonl" '"workflow.completed"'
  smoke_assert_file_contains "${project_dir}/up.jsonl" '"ok":true'
  local ssh_port
  ssh_port="$(smoke_json_query "${project_dir}/status.json" "data.instances[0].ssh_port")"
  [[ "${ssh_port}" =~ ^[0-9]+$ ]] || smoke_die "status.json did not include ssh_port"
  smoke_assert_file_contains "${project_dir}/destroy.jsonl" '"workflow.completed"'
  smoke_assert_project_released "${project_dir}"
}

cleanup_broken_yaml() {
  local project_dir
  project_dir="$(new_project_dir "cleanup-broken-yaml")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init --template ubuntu-basic"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up --no-provision"
  printf 'version: 1\ninstances:\n  - name:\n' >"${project_dir}/yeast.yaml"
  local output
  output="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" clean")"
  smoke_assert_contains "${output}" "Project cleaned"
  smoke_assert_port_unbound "127.0.0.1" 2222
  smoke_assert_project_released "${project_dir}"
}

cleanup_orphan_qemu() {
  local project_dir
  project_dir="$(new_project_dir "cleanup-orphan-qemu")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init --template ubuntu-basic"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up --no-provision"
  local pid
  pid="$(python3 - <<'PY' "${project_dir}"
import json
import pathlib
import sys

project_dir = pathlib.Path(sys.argv[1])
meta = json.loads((project_dir / '.yeast' / 'project.json').read_text())
state_path = pathlib.Path.home() / '.yeast' / 'projects' / meta['id'] / 'state.json'
state = json.loads(state_path.read_text())
print(state['instances']['web']['pid'])
PY
)"
  kill -9 "${pid}"
  sleep 1
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" clean"
  local status_out
  status_out="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" status --json")"
  smoke_assert_contains "${status_out}" '"instances":[]'
  smoke_assert_project_released "${project_dir}"
}

forwarded_ports() {
  local project_dir
  project_dir="$(new_project_dir "forwarded-ports")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init --template caddy-single-vm"
  python3 - <<'PY' "${project_dir}/yeast.yaml"
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
text = path.read_text(encoding='utf-8')
needle = '    ssh_port: 2205\n'
replacement = needle + '    ports:\n      - host_port: 8080\n        guest_port: 80\n'
if needle not in text:
    raise SystemExit('expected ssh_port line in yeast.yaml')
path.write_text(text.replace(needle, replacement, 1), encoding='utf-8')
PY
  local up_out
  up_out="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up")"
  smoke_assert_contains "${up_out}" "http://127.0.0.1:8080 -> guest:80"
  local page
  page="$(smoke_wait_http "http://127.0.0.1:8080")"
  smoke_assert_contains "${page}" "Yeast v0.4 provisioning works."
  local status_out
  status_out="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" status")"
  smoke_assert_contains "${status_out}" "http://127.0.0.1:8080 -> guest:80"
  local inspect_out
  inspect_out="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" inspect web")"
  smoke_assert_contains "${inspect_out}" "http://127.0.0.1:8080 -> guest:80"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy"
  if curl -fsS "http://127.0.0.1:8080" >/dev/null 2>&1; then
    smoke_die "expected forwarded port 8080 to be released after destroy"
  fi
  smoke_assert_project_released "${project_dir}"
}

repeat_lifecycle() {
  local project_dir
  project_dir="$(new_project_dir "repeat-lifecycle")"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" init --template caddy-single-vm"

  local up1 up2 up3
  local start end

  start="$(smoke_now)"
  up1="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up")"
  end="$(smoke_now)"
  local cold_seconds="$(( end - start ))"
  smoke_assert_contains "${up1}" "provision: ran (first boot)"
  smoke_assert_contains "${up1}" "mode: cold start"

  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"

  start="$(smoke_now)"
  up2="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up")"
  end="$(smoke_now)"
  local warm1_seconds="$(( end - start ))"
  smoke_assert_contains "${up2}" "provision: skipped, config unchanged"
  smoke_assert_contains "${up2}" "mode: warm start"

  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"

  start="$(smoke_now)"
  up3="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" up")"
  end="$(smoke_now)"
  local warm2_seconds="$(( end - start ))"
  smoke_assert_contains "${up3}" "provision: skipped, config unchanged"
  smoke_assert_contains "${up3}" "mode: warm start"

  local caddy_state
  caddy_state="$(run_capture_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" exec web -- systemctl is-active caddy")"
  smoke_assert_contains "${caddy_state}" "active"

  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" down"
  run_logged "${CURRENT_LOG}" bash -lc "cd \"${project_dir}\" && \"${YEAST_BIN_PATH}\" destroy"
  smoke_assert_project_released "${project_dir}"

  {
    printf 'benchmark cold_up_seconds=%s\n' "${cold_seconds}"
    printf 'benchmark warm_up_1_seconds=%s\n' "${warm1_seconds}"
    printf 'benchmark warm_up_2_seconds=%s\n' "${warm2_seconds}"
  } >>"${CURRENT_LOG}"
}

run_lab() {
  local name="$1"
  CURRENT_LAB="${name}"
  CURRENT_LOG="${LOG_DIR}/${name}.log"
  LAB_PROJECT_DIRS=()
  mkdir -p "$(dirname "${CURRENT_LOG}")"
  : >"${CURRENT_LOG}"
  local started_at
  started_at="$(smoke_now)"
  printf '==> %s\n' "${name}"
  if "${name}"; then
    local duration
    duration="$(( $(smoke_now) - started_at ))"
    LAB_RESULTS+=("PASS | ${name} | ${duration}s | ${CURRENT_LOG}")
    printf '[pass] %s (%ss)\n' "${name}" "${duration}"
  else
    local duration
    duration="$(( $(smoke_now) - started_at ))"
    LAB_RESULTS+=("FAIL | ${name} | ${duration}s | ${CURRENT_LOG}")
    return 1
  fi
}

print_summary() {
  local started_at="$1"
  local total
  total="$(( $(smoke_now) - started_at ))"
  printf '\nSummary\n'
  printf '  workspace: %s\n' "${SMOKE_ROOT_DIR}"
  printf '  binary: %s\n' "${YEAST_BIN_PATH}"
  printf '  total: %ss\n' "${total}"
  local result
  for result in "${LAB_RESULTS[@]}"; do
    printf '  %s\n' "${result}"
  done
}

main() {
  local start_all
  start_all="$(smoke_now)"
  local labs=(
    lab01_first_vm
    lab02_provisioning
    lab03_snapshot_restore
    lab04_multi_vm_network
    lab05_guest_control
    lab06_labsbackery_reset
    lab07_templates
    lab08_json_events
    cleanup_broken_yaml
    cleanup_orphan_qemu
    forwarded_ports
    repeat_lifecycle
  )

  case "${REQUESTED_LAB}" in
    lab01) REQUESTED_LAB="lab01_first_vm" ;;
    lab02) REQUESTED_LAB="lab02_provisioning" ;;
    lab03) REQUESTED_LAB="lab03_snapshot_restore" ;;
    lab04) REQUESTED_LAB="lab04_multi_vm_network" ;;
    lab05) REQUESTED_LAB="lab05_guest_control" ;;
    lab06) REQUESTED_LAB="lab06_labsbackery_reset" ;;
    lab07) REQUESTED_LAB="lab07_templates" ;;
    lab08) REQUESTED_LAB="lab08_json_events" ;;
  esac

  if [[ "${REQUESTED_LAB}" == "all" ]]; then
    local lab
    for lab in "${labs[@]}"; do
      run_lab "${lab}"
    done
  else
    local known=false
    local lab
    for lab in "${labs[@]}"; do
      if [[ "${lab}" == "${REQUESTED_LAB}" ]]; then
        known=true
        break
      fi
    done
    if [[ "${known}" != true ]]; then
      smoke_die "unknown lab: ${REQUESTED_LAB}"
    fi
    run_lab "${REQUESTED_LAB}"
  fi

  print_summary "${start_all}"
}

main "$@"
