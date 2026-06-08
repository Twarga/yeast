#!/usr/bin/env bash
set -euo pipefail

SMOKE_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SMOKE_REPO_ROOT="$(cd "${SMOKE_LIB_DIR}/../.." && pwd)"

smoke_now() {
  date +%s
}

smoke_timestamp() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

smoke_info() {
  printf '[info] %s\n' "$*"
}

smoke_warn() {
  printf '[warn] %s\n' "$*" >&2
}

smoke_error() {
  printf '[error] %s\n' "$*" >&2
}

smoke_die() {
  smoke_error "$*"
  exit 1
}

smoke_resolve_binary() {
  if [[ -n "${YEAST_BIN:-}" ]]; then
    if [[ ! -x "${YEAST_BIN}" ]]; then
      smoke_die "YEAST_BIN is not executable: ${YEAST_BIN}"
    fi
    printf '%s\n' "${YEAST_BIN}"
    return 0
  fi

  if [[ -x "${SMOKE_REPO_ROOT}/dist/yeast-linux-amd64" ]]; then
    printf '%s\n' "${SMOKE_REPO_ROOT}/dist/yeast-linux-amd64"
    return 0
  fi

  local built_bin="/tmp/opencode/yeast-smoke-bin"
  smoke_info "building local yeast binary at ${built_bin}" >&2
  (cd "${SMOKE_REPO_ROOT}" && go build -o "${built_bin}" ./cmd/yeast)
  printf '%s\n' "${built_bin}"
}

smoke_prepare_root() {
  if [[ -n "${SMOKE_ROOT:-}" ]]; then
    mkdir -p "${SMOKE_ROOT}"
    printf '%s\n' "${SMOKE_ROOT}"
    return 0
  fi
  mktemp -d "${TMPDIR:-/tmp}/yeast-smoke-XXXXXX"
}

smoke_assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    smoke_die "expected output to contain: ${needle}"
  fi
}

smoke_assert_file_contains() {
  local path="$1"
  local needle="$2"
  if ! grep -Fq "${needle}" "${path}"; then
    smoke_die "expected file ${path} to contain: ${needle}"
  fi
}

smoke_wait_http() {
  local url="$1"
  local attempts="${2:-15}"
  local delay="${3:-2}"
  local output=""
  local attempt
  for ((attempt = 1; attempt <= attempts; attempt++)); do
    if output="$(curl -fsS "${url}")"; then
      printf '%s' "${output}"
      return 0
    fi
    sleep "${delay}"
  done
  smoke_die "timed out waiting for ${url}"
}

smoke_assert_port_unbound() {
  local host="$1"
  local port="$2"
  python3 - <<'PY' "${host}" "${port}"
import socket
import sys

host = sys.argv[1]
port = int(sys.argv[2])
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.settimeout(1)
try:
    sock.connect((host, port))
except OSError:
    sys.exit(0)
else:
    sys.exit(1)
finally:
    sock.close()
PY
  if [[ $? -ne 0 ]]; then
    smoke_die "expected ${host}:${port} to be unbound"
  fi
}

smoke_last_lines() {
  local path="$1"
  local count="${2:-40}"
  python3 - <<'PY' "${path}" "${count}"
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
count = int(sys.argv[2])
if not path.exists():
    sys.exit(0)
lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
for line in lines[-count:]:
    print(line)
PY
}

smoke_cleanup_project() {
  local bin="$1"
  local project_dir="$2"
  if [[ ! -d "${project_dir}" ]]; then
    return 0
  fi
  if [[ ! -f "${project_dir}/.yeast/project.json" ]]; then
    return 0
  fi
  (cd "${project_dir}" && "${bin}" destroy >/dev/null 2>&1 || true)
  (cd "${project_dir}" && "${bin}" clean >/dev/null 2>&1 || true)
}

smoke_project_id() {
  local project_dir="$1"
  python3 - <<'PY' "${project_dir}/.yeast/project.json"
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
if not path.exists():
    sys.exit(0)
data = json.loads(path.read_text(encoding="utf-8"))
print(data.get("id", ""))
PY
}

smoke_assert_project_released() {
  local project_dir="$1"
  local project_id
  project_id="$(smoke_project_id "${project_dir}")"
  if [[ -z "${project_id}" ]]; then
    return 0
  fi
  local runtime_dir="${HOME}/.yeast/projects/${project_id}"
  local state_path="${runtime_dir}/state.json"
  if [[ ! -f "${state_path}" ]]; then
    return 0
  fi
  local instance_count
  instance_count="$(python3 - <<'PY' "${state_path}"
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
data = json.loads(path.read_text(encoding='utf-8'))
print(len(data.get('instances', {})))
PY
)"
  if [[ "${instance_count}" != "0" ]]; then
    smoke_die "state still tracks instances after cleanup: ${state_path}"
  fi
}

smoke_json_query() {
  local path="$1"
  local expr="$2"
  python3 - <<'PY' "${path}" "${expr}"
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
expr = sys.argv[2]
data = json.loads(path.read_text(encoding="utf-8"))

parts = [part for part in expr.split('.') if part]
value = data
for part in parts:
    if '[' in part and part.endswith(']'):
        name, index = part[:-1].split('[', 1)
        if name:
            value = value[name]
        value = value[int(index)]
    else:
        value = value[part]

if isinstance(value, (dict, list)):
    print(json.dumps(value))
else:
    print(value)
PY
}
