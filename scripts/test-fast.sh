#!/usr/bin/env bash
set -euo pipefail

PACKAGES=(
  ./internal/config
  ./internal/project
  ./internal/state
  ./internal/images
  ./internal/provision/cloudinit
  ./internal/output
)

echo "==> fast unit test suite"
go test -count=1 "${PACKAGES[@]}"
