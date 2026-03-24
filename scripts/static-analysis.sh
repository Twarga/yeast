#!/usr/bin/env bash
set -euo pipefail

ARTIFACT_DIR="${1:-artifacts}"
mkdir -p "${ARTIFACT_DIR}"

echo "==> go vet"
go vet ./... 2>&1 | tee "${ARTIFACT_DIR}/go-vet.log"

echo "==> golangci-lint"
golangci-lint run --config .golangci.yml --timeout 5m 2>&1 | tee "${ARTIFACT_DIR}/golangci-lint.log"

echo "==> gosec"
gosec \
  -include=G101,G103,G107,G201,G202,G204,G401,G402 \
  -exclude-dir=test \
  -severity medium \
  -confidence medium \
  ./... 2>&1 | tee "${ARTIFACT_DIR}/gosec.log"

echo "Static analysis completed."
