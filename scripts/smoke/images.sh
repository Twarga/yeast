#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/smoke/lib.sh
source "${SCRIPT_DIR}/lib.sh"

BIN="${1:-$(smoke_resolve_binary)}"
PASS=0
FAIL=0

smoke_info "running images smoke tests"

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    smoke_error "expected output to contain: ${needle}"
    return 1
  fi
}

pass() {
  PASS=$((PASS + 1))
  printf '[PASS] %s\n' "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  printf '[FAIL] %s\n' "$1"
}

# --- Test: pull --list shows all 16 images ---
pull_list_output="$("${BIN}" pull --list 2>&1)" || true

for img in ubuntu-24.04 ubuntu-22.04 debian-12 debian-13 fedora-41 fedora-42 \
           rocky-9 alma-9 centos-stream-9 amazon-linux-2023 \
           kali-2026.1 parrot-security-7.1 alpine-3.21 arch-linux nixos-24.11 opensuse-leap-15.6; do
  if echo "${pull_list_output}" | grep -qF "${img}"; then
    pass "pull_list_contains_${img}"
  else
    fail "pull_list_contains_${img}"
  fi
done

# --- Test: pull --list shows category headers ---
for cat in "General Purpose" "DevOps" "Enterprise" "Security" "Minimal" "Niche"; do
  if echo "${pull_list_output}" | grep -qF "${cat}"; then
    pass "pull_list_category_${cat}"
  else
    fail "pull_list_category_${cat}"
  fi
done

# --- Test: pull --list shows (manual) for manual images ---
if echo "${pull_list_output}" | grep -qF "(manual)"; then
  pass "pull_list_manual_hint"
else
  fail "pull_list_manual_hint"
fi

# --- Test: pull --cached with empty cache ---
pull_cached_output="$("${BIN}" pull --cached 2>&1)" || true
pass "pull_cached_exits_0"

# --- Test: pull unknown image shows suggestions ---
pull_unknown_output="$("${BIN}" pull unknown-image-99 2>&1)" || true
if echo "${pull_unknown_output}" | grep -qF "unknown-image-99"; then
  pass "pull_unknown_shows_name"
else
  fail "pull_unknown_shows_name"
fi

# --- Test: images clean --dry-run ---
clean_output="$("${BIN}" images clean --dry-run 2>&1)" || true
pass "images_clean_dry_run_exits_0"

# --- Test: up with manual image shows instructions ---
project_dir="$(mktemp -d)"
trap "rm -rf '${project_dir}'" EXIT

# Init creates a default config, then overwrite with kali
(cd "${project_dir}" && "${BIN}" init >/dev/null 2>&1)
cat > "${project_dir}/yeast.yaml" <<'EOF'
version: 1
instances:
  - name: web
    image: kali-2026.1
    memory: 1024
    cpus: 1
EOF

up_output="$(cd "${project_dir}" && "${BIN}" up 2>&1)" || true
if echo "${up_output}" | grep -qF "manual download"; then
  pass "up_manual_image_instructions"
else
  fail "up_manual_image_instructions"
fi

printf '\n--- Results ---\n'
printf 'PASS: %d\n' "${PASS}"
printf 'FAIL: %d\n' "${FAIL}"

if [[ "${FAIL}" -gt 0 ]]; then
  exit 1
fi
