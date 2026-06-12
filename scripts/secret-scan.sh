#!/usr/bin/env bash
set -euo pipefail

ARTIFACT_DIR="${1:-artifacts}"
mkdir -p "${ARTIFACT_DIR}"
LOG_FILE="${ARTIFACT_DIR}/secret-scan.log"

PATTERNS=(
  'sk-[A-Za-z0-9]{20,}'
  'ghp_[A-Za-z0-9_]{20,}'
  'github_pat_[A-Za-z0-9_]{20,}'
  'ANTHROPIC[_]API[_]KEY'
  'OC[_]GO[_]CC[_]API[_]KEY'
  'OPENAI[_]API[_]KEY'
)

mapfile -d '' FILES < <(
  git ls-files -z |
    while IFS= read -r -d '' file; do
      [[ -f "${file}" ]] && printf '%s\0' "${file}"
    done
)

if [[ ${#FILES[@]} -eq 0 ]]; then
  printf 'No tracked files found.\n' | tee "${LOG_FILE}"
  exit 0
fi

rg_args=()
for pattern in "${PATTERNS[@]}"; do
  rg_args+=(-e "${pattern}")
done

if rg --color=never --line-number "${rg_args[@]}" "${FILES[@]}" >"${LOG_FILE}"; then
  cat "${LOG_FILE}"
  printf '\nSecret-like values found in tracked files.\n' >&2
  exit 1
fi

printf 'No secret-like values found in tracked files.\n' | tee "${LOG_FILE}"
