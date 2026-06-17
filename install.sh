#!/usr/bin/env bash
# Yeast Installer
#
# Default mode: download and install the pre-built release binary.
# Source-build mode (contributors / unsupported platforms): YEAST_INSTALL_MODE=source
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
#   bash install.sh
#   YEAST_INSTALL_MODE=source bash install.sh

set -euo pipefail

# ─── Configuration ─────────────────────────────────────────���────────────────
YEAST_VERSION="${YEAST_VERSION:-v1.1.1}"
YEAST_INSTALL_DIR="${YEAST_INSTALL_DIR:-/usr/local/bin}"
YEAST_BIN_PATH="${YEAST_INSTALL_DIR}/yeast"
YEAST_INSTALL_MODE="${YEAST_INSTALL_MODE:-binary}"
YEAST_INSTALL_VERBOSE="${YEAST_INSTALL_VERBOSE:-0}"
YEAST_SKIP_DOCTOR="${YEAST_SKIP_DOCTOR:-0}"

# Source-mode only (advanced / contributor path)
YEAST_REPO_URL="${YEAST_REPO_URL:-https://github.com/Twarga/yeast.git}"
YEAST_REF="${YEAST_REF:-${YEAST_VERSION}}"
YEAST_MIN_GO_VERSION="${YEAST_MIN_GO_VERSION:-1.25.0}"
YEAST_GO_VERSION="${YEAST_GO_VERSION:-1.25.0}"
YEAST_GO_INSTALL_ROOT="${YEAST_GO_INSTALL_ROOT:-/usr/local/lib/yeast/go}"

# ─── Colors ─────────────────────────────────────────────────────────────────
if [[ -t 1 && -z "${NO_COLOR:-}" && "${TERM:-}" != "dumb" ]]; then
  C_RESET=$'\033[0m'; C_BOLD=$'\033[1m'; C_DIM=$'\033[2m'
  C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'
  C_BLUE=$'\033[34m'; C_CYAN=$'\033[36m'
else
  C_RESET=""; C_BOLD=""; C_DIM=""; C_RED=""; C_GREEN=""
  C_YELLOW=""; C_BLUE=""; C_CYAN=""
fi

paint()   { local code="$1"; shift; printf '%b%s%b' "${code}" "$*" "${C_RESET}"; }
bold()    { paint "${C_BOLD}" "$*"; }
dim()     { paint "${C_DIM}" "$*"; }
blue()    { paint "${C_BOLD}${C_BLUE}" "$*"; }
cyan()    { paint "${C_BOLD}${C_CYAN}" "$*"; }
green()   { paint "${C_BOLD}${C_GREEN}" "$*"; }
yellow()  { paint "${C_BOLD}${C_YELLOW}" "$*"; }
red()     { paint "${C_BOLD}${C_RED}" "$*"; }

section()  { printf '\n%s %s\n' "$(blue '==>')" "$(bold "$*")"; }
info()     { printf '%s %s\n' "$(blue '[info]')" "$*"; }
success()  { printf '%s %s\n' "$(green '[ok]')" "$*"; }
warn()     { printf '%s %s\n' "$(yellow '[warn]')" "$*" >&2; }
die() {
  printf '%s %s\n' "$(red '[fail]')" "$*" >&2
  exit 1
}
key_value() { printf '    %s %s\n' "$(dim "$1:")" "$2"; }

# ─── Spinner ────────────────────────────────────────────────────────────────
clear_line() { [[ -t 1 ]] && printf '\r\033[2K'; }
spinner() {
  local pid="$1" label="$2"
  local frames=('|' '/' '-' $'\\')
  local i=0
  while kill -0 "${pid}" 2>/dev/null; do
    printf '\r%s %s %s' "$(cyan "[${frames[$i]}]")" "${label}" "$(dim 'working...')"
    i=$(((i + 1) % ${#frames[@]}))
    sleep 0.12
  done
  clear_line
}

run_step() {
  local label="$1"; shift
  local log_file
  log_file="$(mktemp)"
  if [[ "${YEAST_INSTALL_VERBOSE}" == "1" || ! -t 1 ]]; then
    info "${label}"
    "$@" >"${log_file}" 2>&1 && { success "${label}"; rm -f "${log_file}"; return 0; }
    local rc=$?
    warn "Step failed: ${label}"
    cat "${log_file}" >&2
    rm -f "${log_file}"
    return "${rc}"
  fi
  ("$@" >"${log_file}" 2>&1) & local pid=$!
  spinner "${pid}" "${label}"
  if wait "${pid}"; then
    success "${label}"
    rm -f "${log_file}"
    return 0
  fi
  warn "Step failed: ${label}"
  cat "${log_file}" >&2
  rm -f "${log_file}"
  return 1
}

run_required() {
  local label="$1"; shift
  run_step "${label}" "$@" || die "unable to continue: ${label}"
}

run_optional() {
  local label="$1"; shift
  run_step "${label}" "$@" || warn "${label} completed with warnings (non-fatal)"
}

# ─── Helpers ────────────────────────────────────────────────────────────────
need_root() {
  if [[ "$(id -u)" -eq 0 ]]; then "$@"; return; fi
  command -v sudo >/dev/null 2>&1 || die "sudo is required to write to ${YEAST_INSTALL_DIR}"
  sudo "$@"
}

require_sudo() {
  [[ "$(id -u)" -eq 0 ]] && return
  command -v sudo >/dev/null 2>&1 || die "sudo is required for installation"
  info "Requesting sudo access"
  sudo -v
  (while true; do sudo -n true >/dev/null 2>&1 || exit; sleep 30; done) &
  SUDO_KEEPALIVE_PID=$!
}

cleanup() {
  [[ -n "${SUDO_KEEPALIVE_PID:-}" ]] && kill "${SUDO_KEEPALIVE_PID}" >/dev/null 2>&1 || true
  [[ -n "${WORKDIR:-}" ]] && rm -rf "${WORKDIR}" >/dev/null 2>&1 || true
}

version_at_least() {
  local current="$1" minimum="$2"
  local cm cn cp mm mn mp
  IFS=. read -r cm cn cp <<<"${current}"
  IFS=. read -r mm mn mp <<<"${minimum}"
  cm="${cm:-0}"; cn="${cn:-0}"; cp="${cp:-0}"
  mm="${mm:-0}"; mn="${mn:-0}"; mp="${mp:-0}"
  ((cm > mm)) && return 0; ((cm < mm)) && return 1
  ((cn > mn)) && return 0; ((cn < mn)) && return 1
  ((cp >= mp))
}

# ─── Platform Detection ─────────────────────────────────────────────────────
detect_platform() {
  ARCH="$(uname -m)"
  case "${ARCH}" in
    x86_64|amd64) YEAST_ARCH="amd64" ;;
    aarch64|arm64) YEAST_ARCH="arm64" ;;
    *) die "unsupported CPU architecture: ${ARCH}" ;;
  esac

  IS_WSL=0; WSL_VERSION=""
  if [[ -f /proc/version ]] && grep -qi microsoft /proc/version; then
    IS_WSL=1
    if [[ -f /proc/sys/kernel/osrelease ]] && grep -q "WSL2" /proc/sys/kernel/osrelease; then
      WSL_VERSION="2"
    else
      WSL_VERSION="1"
    fi
  fi

  IS_CONTAINER=0
  if [[ -f /.dockerenv ]] || grep -q docker /proc/self/cgroup 2>/dev/null; then
    IS_CONTAINER=1
  fi
}

# ─── Fetch Helpers ──────────────────────────────────────────────────────────
http_get() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 --retry-delay 2 -o "${dest}" "${url}"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "${dest}" "${url}"
  else
    die "curl or wget is required to download Yeast"
  fi
}

# ─── Binary Install (default) ───────────────────────────────────────────────
compute_release_artifact() {
  RELEASE_ARTIFACT="yeast_linux_${YEAST_ARCH}.tar.gz"
  RELEASE_URL="https://github.com/Twarga/yeast/releases/download/${YEAST_VERSION}/${RELEASE_ARTIFACT}"
  CHECKSUMS_URL="https://github.com/Twarga/yeast/releases/download/${YEAST_VERSION}/SHA256SUMS.txt"

  if [[ "${YEAST_ARCH}" != "amd64" ]]; then
    warn "Architecture ${YEAST_ARCH}: only amd64 has a first-class release artifact."
    warn "If no artifact exists for your arch, use YEAST_INSTALL_MODE=source."
  fi
}

download_release_artifact() {
  info "Downloading ${RELEASE_ARTIFACT} from ${RELEASE_URL}"
  http_get "${RELEASE_URL}" "${WORKDIR}/${RELEASE_ARTIFACT}"
  http_get "${CHECKSUMS_URL}" "${WORKDIR}/SHA256SUMS.txt"
}

verify_release_artifact() {
  local expected_line
  expected_line="$(grep "${RELEASE_ARTIFACT}" "${WORKDIR}/SHA256SUMS.txt" || true)"
  if [[ -z "${expected_line}" ]]; then
    die "checksum not found for ${RELEASE_ARTIFACT} in SHA256SUMS.txt"
  fi
  (cd "${WORKDIR}" && printf '%s\n' "${expected_line}" | sha256sum -c -) \
    || die "checksum mismatch for ${RELEASE_ARTIFACT}"
}

extract_and_install_binary() {
  local extracted_binary=""
  tar -xzf "${WORKDIR}/${RELEASE_ARTIFACT}" -C "${WORKDIR}"
  for candidate in \
    "${WORKDIR}/yeast" \
    "${WORKDIR}/${RELEASE_ARTIFACT%.tar.gz}" \
    "${WORKDIR}/yeast_linux_${YEAST_ARCH}"; do
    if [[ -f "${candidate}" ]]; then
      extracted_binary="${candidate}"
      break
    fi
  done
  [[ -n "${extracted_binary}" ]] || die "binary not found after extracting ${RELEASE_ARTIFACT}"
  need_root install -d "${YEAST_INSTALL_DIR}"
  need_root install -m 0755 "${extracted_binary}" "${YEAST_BIN_PATH}"
}

install_from_release() {
  section "Installing Release Binary"
  compute_release_artifact
  key_value "version" "${YEAST_VERSION}"
  key_value "artifact" "${RELEASE_ARTIFACT}"
  key_value "arch" "${YEAST_ARCH}"

  run_required "Downloading ${RELEASE_ARTIFACT}" download_release_artifact
  run_required "Verifying checksum" verify_release_artifact
  run_required "Installing binary" extract_and_install_binary
}

# ─── Source Build (advanced / contributor) ──────────────────────────────────
go_version_value() {
  local output; output="$("$1" version 2>/dev/null || true)"
  output="${output#go version go}"; printf '%s' "${output%% *}"
}

official_go_bin_path() {
  printf '%s/go%s/bin/go' "${YEAST_GO_INSTALL_ROOT}" "${YEAST_GO_VERSION}"
}

resolve_go_bin() {
  local candidate; candidate="$(command -v go || true)"
  if [[ -n "${candidate}" ]]; then printf '%s' "${candidate}"; return 0; fi
  candidate="$(official_go_bin_path)"
  if [[ -x "${candidate}" ]]; then printf '%s' "${candidate}"; return 0; fi
  return 1
}

download_official_go() {
  local archive url install_dir
  archive="${WORKDIR}/go${YEAST_GO_VERSION}.linux-${YEAST_ARCH}.tar.gz"
  url="https://go.dev/dl/go${YEAST_GO_VERSION}.linux-${YEAST_ARCH}.tar.gz"
  install_dir="${YEAST_GO_INSTALL_ROOT}/go${YEAST_GO_VERSION}"

  info "Downloading Go ${YEAST_GO_VERSION}..."
  http_get "${url}" "${archive}"
  local checksum_url="${url}.sha256"
  local checksum; checksum="$(curl -fsSL --retry 3 "${checksum_url}" | awk 'NR==1{print $1;exit}' || true)"
  if [[ -n "${checksum}" ]]; then
    printf '%s  %s\n' "${checksum}" "${archive}" | sha256sum -c -
  fi

  need_root rm -rf "${install_dir}"
  need_root install -d "$(dirname "${install_dir}")"
  need_root tar -C "$(dirname "${install_dir}")" -xzf "${archive}"
  need_root mv "$(dirname "${install_dir}")/go" "${install_dir}"
  GO_BIN="${install_dir}/bin/go"
}

ensure_go_toolchain() {
  GO_BIN="$(resolve_go_bin || true)"
  if [[ -n "${GO_BIN}" ]]; then
    local current; current="$(go_version_value "${GO_BIN}")"
    if [[ -n "${current}" ]] && version_at_least "${current}" "${YEAST_MIN_GO_VERSION}"; then
      info "Using Go ${current} at ${GO_BIN}"
      return
    fi
    warn "Found Go ${current:-unknown}, need ${YEAST_MIN_GO_VERSION}+"
  fi
  run_step "Installing Go ${YEAST_GO_VERSION}" download_official_go
}

detect_distro() {
  DISTRO=""; DISTRO_ID=""; DISTRO_VERSION=""; PKG_MANAGER=""
  if [[ -f /etc/os-release ]]; then
    DISTRO_ID="$(source /etc/os-release && printf '%s' "${ID:-}")"
    DISTRO_VERSION="$(source /etc/os-release && printf '%s' "${VERSION_ID:-}")"
    DISTRO="$(source /etc/os-release && printf '%s' "${NAME:-}")"
  fi
  if command -v apt-get >/dev/null 2>&1; then PKG_MANAGER="apt"
  elif command -v dnf >/dev/null 2>&1; then PKG_MANAGER="dnf"
  elif command -v yum >/dev/null 2>&1; then PKG_MANAGER="yum"
  elif command -v pacman >/dev/null 2>&1; then PKG_MANAGER="pacman"
  else die "no supported package manager found (apt, dnf, yum, pacman)"; fi
}

pkg_install() {
  case "${PKG_MANAGER}" in
    apt) need_root env DEBIAN_FRONTEND=noninteractive apt-get install -y "$@" ;;
    dnf) need_root dnf install -y "$@" ;;
    yum) need_root yum install -y "$@" ;;
    pacman) need_root pacman -S --noconfirm --needed "$@" ;;
    *) die "unsupported package manager: ${PKG_MANAGER}" ;;
  esac
}

install_source_dependencies() {
  section "Installing Source Build Dependencies"
  case "${PKG_MANAGER}" in
    apt)
      need_root env DEBIAN_FRONTEND=noninteractive apt-get update -qq
      pkg_install git curl tar ca-certificates build-essential openssh-client
      ;;
    dnf|yum)
      pkg_install git curl tar ca-certificates gcc make openssh-clients
      ;;
    pacman)
      need_root pacman -Sy --noconfirm
      pkg_install git curl tar ca-certificates base-devel openssh
      ;;
  esac
}

ensure_genisoimage_compat() {
  if command -v genisoimage >/dev/null 2>&1 || command -v mkisofs >/dev/null 2>&1; then return 0; fi
  if ! command -v xorriso >/dev/null 2>&1; then return 0; fi
  need_root install -d "${YEAST_INSTALL_DIR}"
  cat <<'EOF' | need_root tee "${YEAST_INSTALL_DIR}/genisoimage" >/dev/null
#!/usr/bin/env bash
exec xorriso -as mkisofs "$@"
EOF
  need_root chmod 0755 "${YEAST_INSTALL_DIR}/genisoimage"
  info "Created genisoimage wrapper using xorriso"
}

install_from_source() {
  section "Source Build (advanced mode)"
  warn "YEAST_INSTALL_MODE=source: this path installs Go and builds from source."
  warn "For most users the release binary path is faster and more reliable."

  detect_distro
  key_value "distro" "${DISTRO:-unknown} ${DISTRO_VERSION:-}"
  key_value "package manager" "${PKG_MANAGER}"

  install_source_dependencies

  ensure_go_toolchain
  local go_bin; go_bin="$(resolve_go_bin || true)"
  [[ -n "${go_bin}" ]] || die "failed to resolve Go toolchain"

  section "Building Yeast"
  local SRC_DIR="${WORKDIR}/src"
  run_required "Cloning repository" git clone --depth 1 --branch "${YEAST_REF}" "${YEAST_REPO_URL}" "${SRC_DIR}"

  local version="${YEAST_REF}"
  run_required "Building CLI binary" bash -c "
    cd '${SRC_DIR}'
    '${go_bin}' build -trimpath -ldflags '-s -w -X yeast/internal/app.Version=${version}' -o '${WORKDIR}/yeast' ./cmd/yeast
  "

  need_root install -d "${YEAST_INSTALL_DIR}"
  run_required "Installing binary" need_root install -m 0755 "${WORKDIR}/yeast" "${YEAST_BIN_PATH}"

  section "Virtualization Tools"
  detect_distro
  case "${PKG_MANAGER}" in
    apt)
      run_optional "Installing QEMU" pkg_install qemu-system-x86 qemu-utils
      run_optional "Installing ISO builder" pkg_install genisoimage
      ;;
    dnf|yum)
      run_optional "Installing QEMU" pkg_install qemu-system-x86 qemu-img
      run_optional "Installing ISO builder" pkg_install genisoimage
      ;;
    pacman)
      run_optional "Installing QEMU" pkg_install qemu-base
      run_optional "Installing ISO builder" pkg_install cdrtools
      ;;
  esac
  run_optional "Preparing genisoimage compat" ensure_genisoimage_compat
}

# ─── Verify Binary ──────────────────────────────────────────────────────────
verify_binary() {
  [[ -x "${YEAST_BIN_PATH}" ]] || die "installed binary not executable: ${YEAST_BIN_PATH}"
  local actual; actual="$("${YEAST_BIN_PATH}" version 2>/dev/null || true)"
  [[ -n "${actual}" ]] || die "installed binary did not print a version"
  info "Installed Yeast: ${actual}"
}

# ─── User Environment ───────────────────────────────────────────────────────
detect_target_user() {
  TARGET_USER="${SUDO_USER:-${USER:-$(id -un)}}"
  if command -v getent >/dev/null 2>&1; then
    TARGET_HOME="$(getent passwd "${TARGET_USER}" | cut -d: -f6)"
  else
    TARGET_HOME="$(awk -F: -v user="${TARGET_USER}" '$1==user{print $6}' /etc/passwd)"
  fi
  [[ -n "${TARGET_HOME}" ]] || die "failed to resolve home directory for ${TARGET_USER}"
  TARGET_GROUP="$(id -gn "${TARGET_USER}")"
}

ensure_user_paths() {
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast"
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast/cache"
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast/cache/images"
}

run_as_target_user() {
  if [[ "$(id -un)" == "${TARGET_USER}" ]]; then
    "$@"
    return
  fi
  if command -v sudo >/dev/null 2>&1; then
    sudo -u "${TARGET_USER}" "$@"
    return
  fi
  if command -v runuser >/dev/null 2>&1; then
    runuser -u "${TARGET_USER}" -- "$@"
    return
  fi
  die "need sudo or runuser to execute commands as ${TARGET_USER}"
}

ensure_user_ssh_key() {
  local ssh_dir="${TARGET_HOME}/.ssh"
  local private_key="${ssh_dir}/id_ed25519"
  local public_key="${private_key}.pub"

  if [[ -f "${public_key}" ]]; then
    info "Found existing SSH public key at ${public_key}"
    return 0
  fi

  command -v ssh-keygen >/dev/null 2>&1 || die "ssh-keygen is required to create a default SSH key"
  need_root install -d -m 0700 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${ssh_dir}"

  if [[ -f "${private_key}" ]]; then
    run_as_target_user bash -lc "ssh-keygen -y -f '${private_key}' > '${public_key}'"
    return 0
  fi

  run_as_target_user ssh-keygen -t ed25519 -f "${private_key}" -N ""
}

run_doctor_fixups() {
  "${YEAST_BIN_PATH}" doctor --fix --yes
}

# ─── Summary ────────────────────────────────────────────────────────────────
print_summary() {
  section "Installation Complete"
  key_value "binary" "${YEAST_BIN_PATH}"
  key_value "version" "$("${YEAST_BIN_PATH}" version 2>/dev/null || echo 'unknown')"
  key_value "user" "${TARGET_USER}"
  key_value "mode" "${YEAST_INSTALL_MODE}"
  key_value "arch" "${YEAST_ARCH}"

  if [[ "${IS_WSL}" -eq 1 ]]; then
    printf '\n'
    yellow "WSL${WSL_VERSION} detected: Yeast is installed, but KVM acceleration is not guaranteed."
    dim "  - VMs may fall back to TCG (software emulation) which is ~10x slower."
    dim "  - Run 'yeast doctor' to check your WSL setup."
    dim "  - See docs/getting-started/installation-windows-wsl.md for details."
  fi

  printf '\n'
  info "Next steps"
  key_value "1" "yeast doctor"
  key_value "2" "mkdir my-lab && cd my-lab"
  key_value "3" "yeast init"
  key_value "4" "yeast up"
  key_value "5" "yeast ssh <instance-name>"
  printf '\n'
  dim "  Tip: 'yeast up' downloads trusted images automatically."
  dim "  Use 'yeast pull --list' to browse available images."
}

# ─── Main ───────────────────────────────────────────────────────────────────
show_banner() {
  printf '%s\n' "$(bold 'Yeast Installer')"
  printf '%s\n' "$(dim 'Linux VM orchestration for QEMU/KVM — https://github.com/Twarga/yeast')"
}

main() {
  [[ "$(uname -s)" == "Linux" ]] || die "Yeast installer supports Linux only"

  WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/yeast-install.XXXXXX")"
  trap cleanup EXIT
  KVM_GROUP_UPDATED=0

  show_banner
  detect_platform

  section "Platform"
  key_value "arch" "${YEAST_ARCH}"
  key_value "environment" "$(
    if [[ "${IS_WSL}" -eq 1 ]]; then echo "WSL${WSL_VERSION} (beta — KVM not guaranteed)";
    elif [[ "${IS_CONTAINER}" -eq 1 ]]; then echo "container (KVM not available)";
    else echo "native Linux"; fi
  )"
  key_value "mode" "${YEAST_INSTALL_MODE}"

  if [[ "${IS_WSL}" -eq 1 && "${WSL_VERSION}" == "1" ]]; then
    die "WSL1 is not supported. Please upgrade to WSL2."
  fi

  detect_target_user
  require_sudo

  case "${YEAST_INSTALL_MODE}" in
    binary) install_from_release ;;
    source) install_from_source ;;
    *) die "unknown YEAST_INSTALL_MODE: ${YEAST_INSTALL_MODE} (use 'binary' or 'source')" ;;
  esac

  run_required "Verifying installed binary" verify_binary

  section "User Environment"
  run_required "Creating Yeast directories" ensure_user_paths

  if [[ "${YEAST_SKIP_DOCTOR}" != "1" ]]; then
    section "Host Readiness Check"
    run_optional "Running yeast doctor --fix --yes" run_doctor_fixups
  fi

  run_optional "Ensuring SSH key" ensure_user_ssh_key

  print_summary
}

if [[ -z "${BASH_SOURCE[0]-}" || "${BASH_SOURCE[0]-}" == "$0" ]]; then
  main "$@"
fi
