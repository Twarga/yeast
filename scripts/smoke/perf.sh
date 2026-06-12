#!/usr/bin/env bash
# scripts/smoke/perf.sh — Performance benchmark suite
# Usage: bash scripts/smoke/perf.sh [path-to-yeast-binary]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib.sh"

BIN="${1:-$(smoke_resolve_binary)}"
WORK_DIR="$(smoke_prepare_root)"
RESULTS=()
BASELINE_FILE="${SMOKE_REPO_ROOT}/artifacts/perf-baseline.txt"

bench() {
  local label="$1"; shift
  local start end elapsed
  start=$(smoke_now)
  "$@" >/dev/null 2>&1
  end=$(smoke_now)
  elapsed=$((end - start))
  RESULTS+=("${label}: ${elapsed}s")
  printf '  \033[36m●\033[0m %-40s %ds\n' "${label}" "${elapsed}"
}

section() { printf '\n\033[1;36m═══ %s ═══\033[0m\n' "$*"; }

# --- Setup ---
section "Setup"
BENCH_DIR="${WORK_DIR}/bench"
mkdir -p "${BENCH_DIR}"
(cd "${BENCH_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${BENCH_DIR}/yeast.yaml" <<'YAML'
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 512
    disk: 8
    ssh_port: 2295
provision:
  shell:
    - echo perf-provision-ready
YAML

# --- Benchmarks ---
section "Performance Benchmarks"

bench "status_latency" bash -c "cd '${BENCH_DIR}' && '${BIN}' status"
bench "version_latency" "${BIN}" version
bench "doctor_latency" "${BIN}" doctor
bench "pull_list_latency" "${BIN}" pull --list

# Cold boot (first up with provision)
bench "cold_boot_provision" bash -c "cd '${BENCH_DIR}' && '${BIN}' up"

# Warm boot (second up, provision skipped)
bench "warm_boot_cached" bash -c "cd '${BENCH_DIR}' && '${BIN}' up"

# Snapshot/restore require a stopped instance.
(cd "${BENCH_DIR}" && "${BIN}" down >/dev/null 2>&1)
bench "snapshot_create" bash -c "cd '${BENCH_DIR}' && '${BIN}' snapshot web bench-snap --description 'benchmark'"

bench "snapshot_restore" bash -c "cd '${BENCH_DIR}' && '${BIN}' restore web bench-snap"

# Down + Destroy
bench "down" bash -c "cd '${BENCH_DIR}' && '${BIN}' down"
bench "destroy" bash -c "cd '${BENCH_DIR}' && '${BIN}' destroy"

# --- Multi-VM Benchmark ---
section "Multi-VM Benchmark"
MULTI_DIR="${WORK_DIR}/bench-multi"
mkdir -p "${MULTI_DIR}"
(cd "${MULTI_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${MULTI_DIR}/yeast.yaml" <<'YAML'
version: 1
instances:
  - name: vm1
    image: ubuntu-24.04
    memory: 512
    disk: 8
    ssh_port: 2296
  - name: vm2
    image: ubuntu-24.04
    memory: 512
    disk: 8
    ssh_port: 2297
YAML
bench "multi_vm_boot_2" bash -c "cd '${MULTI_DIR}' && '${BIN}' up"
bench "multi_vm_down" bash -c "cd '${MULTI_DIR}' && '${BIN}' down"
bench "multi_vm_destroy" bash -c "cd '${MULTI_DIR}' && '${BIN}' destroy"

# --- Image Pull Benchmark ---
section "Image Pull Benchmark"
PULL_DIR="${WORK_DIR}/bench-pull"
mkdir -p "${PULL_DIR}"
(cd "${PULL_DIR}" && "${BIN}" init >/dev/null 2>&1)
# Only benchmark if image not cached
if [[ ! -f "${HOME}/.yeast/cache/images/ubuntu-24.04/image.qcow2" ]]; then
  bench "image_pull_ubuntu_24_04" bash -c "cd '${PULL_DIR}' && '${BIN}' pull ubuntu-24.04"
else
  printf '  \033[33m●\033[0m %-40s %s\n' "image_pull_ubuntu_24_04" "skipped (cached)"
fi

# --- Memory Overhead ---
section "Memory Overhead"
MEM_DIR="${WORK_DIR}/bench-mem"
mkdir -p "${MEM_DIR}"
(cd "${MEM_DIR}" && "${BIN}" init >/dev/null 2>&1)
cat > "${MEM_DIR}/yeast.yaml" <<'YAML'
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    disk: 8
    ssh_port: 2298
YAML
(cd "${MEM_DIR}" && "${BIN}" up >/dev/null 2>&1)
QEMU_RSS=$(ps aux | grep qemu | grep -v grep | awk '{sum+=$6} END {print sum/1024 "MB"}')
RESULTS+=("qemu_rss_memory: ${QEMU_RSS}")
printf '  \033[36m●\033[0m %-40s %s\n' "qemu_rss_memory" "${QEMU_RSS}"
(cd "${MEM_DIR}" && "${BIN}" down >/dev/null 2>&1)
(cd "${MEM_DIR}" && "${BIN}" destroy >/dev/null 2>&1)

# --- Cleanup ---
smoke_cleanup_project "${BIN}" "${BENCH_DIR}" 2>/dev/null || true
smoke_cleanup_project "${BIN}" "${MULTI_DIR}" 2>/dev/null || true
smoke_cleanup_project "${BIN}" "${PULL_DIR}" 2>/dev/null || true
smoke_cleanup_project "${BIN}" "${MEM_DIR}" 2>/dev/null || true
rm -rf "${WORK_DIR}"

# --- Save Baseline ---
mkdir -p "$(dirname "${BASELINE_FILE}")"
{
  echo "PERFORMANCE BASELINES ($(date -u +"%Y-%m-%d"))"
  echo "─────────────────────────────────────────────"
  printf '%s\n' "${RESULTS[@]}"
  echo "─────────────────────────────────────────────"
} > "${BASELINE_FILE}"

# --- Regression Detection ---
if [[ -f "${BASELINE_FILE}" ]]; then
  section "Regression Check"
  while IFS= read -r line; do
    label=$(echo "${line}" | cut -d: -f1 | xargs)
    current=$(echo "${line}" | grep -o '[0-9]*s$' | tr -d 's' || true)
    if [[ -n "${current}" ]]; then
      # Check against previous baseline if exists
      prev_line=$(grep "^${label}:" "${BASELINE_FILE}" 2>/dev/null | head -1 || true)
      if [[ -n "${prev_line}" ]]; then
        prev=$(echo "${prev_line}" | grep -o '[0-9]*s$' | tr -d 's' || true)
        if [[ -n "${prev}" && "${prev}" -gt 0 ]]; then
          ratio=$((current * 100 / prev))
          if [[ ${ratio} -gt 200 ]]; then
            printf '  \033[31m⚠\033[0m %-40s %ds (was %ds, %d%% slower)\n' "${label}" "${current}" "${prev}" "${ratio}"
          fi
        fi
      fi
    fi
  done <<< "$(printf '%s\n' "${RESULTS[@]}")"
fi

# --- Summary ---
section "Performance Baselines"
printf '%s\n' "${RESULTS[@]}"
printf '\nBaseline saved to: %s\n' "${BASELINE_FILE}"
printf '\n'
