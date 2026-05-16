#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-v0.1.0}"
DIST_DIR="${DIST_DIR:-dist}"
TARGET_OS="${TARGET_OS:-linux}"
TARGET_ARCH="${TARGET_ARCH:-amd64}"
PACKAGE_NAME="yeast-${TARGET_OS}-${TARGET_ARCH}"
BIN_PATH="${DIST_DIR}/${PACKAGE_NAME}"

mkdir -p "${DIST_DIR}"
rm -f "${BIN_PATH}" "${BIN_PATH}.sha256"

echo "==> building ${PACKAGE_NAME} (${VERSION})"
CGO_ENABLED=0 GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" go build \
  -trimpath \
  -ldflags "-s -w -X yeast/internal/app.Version=${VERSION}" \
  -o "${BIN_PATH}" \
  ./cmd/yeast

chmod 0755 "${BIN_PATH}"

echo "==> writing checksum"
(
  cd "${DIST_DIR}"
  sha256sum "${PACKAGE_NAME}" >"${PACKAGE_NAME}.sha256"
)

echo "built ${BIN_PATH}"
echo "checksum ${BIN_PATH}.sha256"
