#!/usr/bin/env bash
set -euo pipefail

YEAST_REPO_URL="${YEAST_REPO_URL:-https://github.com/Twarga/yeast.git}"
YEAST_REF="${YEAST_REF:-main}"
YEAST_INSTALL_DIR="${YEAST_INSTALL_DIR:-/usr/local/bin}"
YEAST_BIN_PATH="${YEAST_INSTALL_DIR}/yeast"
YEAST_INSTALL_VERBOSE="${YEAST_INSTALL_VERBOSE:-0}"
YEAST_KEEP_LOGS="${YEAST_KEEP_LOGS:-0}"
YEAST_MIN_GO_VERSION="${YEAST_MIN_GO_VERSION:-1.25.0}"
YEAST_GO_VERSION="${YEAST_GO_VERSION:-1.26.3}"
YEAST_GO_INSTALL_ROOT="${YEAST_GO_INSTALL_ROOT:-/usr/local/lib/yeast/go}"
YEAST_GO_TARBALL_SHA256="${YEAST_GO_TARBALL_SHA256:-}"
GO_BIN=""

if [[ -t 1 && -z "${NO_COLOR:-}" && "${TERM:-}" != "dumb" ]]; then
  C_RESET=$'\033[0m'
  C_BOLD=$'\033[1m'
  C_DIM=$'\033[2m'
  C_RED=$'\033[31m'
  C_GREEN=$'\033[32m'
  C_YELLOW=$'\033[33m'
  C_BLUE=$'\033[34m'
  C_CYAN=$'\033[36m'
else
  C_RESET=""
  C_BOLD=""
  C_DIM=""
  C_RED=""
  C_GREEN=""
  C_YELLOW=""
  C_BLUE=""
  C_CYAN=""
fi

paint() {
  local code="$1"
  shift
  printf '%b%s%b' "${code}" "$*" "${C_RESET}"
}

bold() {
  paint "${C_BOLD}" "$*"
}

dim() {
  paint "${C_DIM}" "$*"
}

blue() {
  paint "${C_BOLD}${C_BLUE}" "$*"
}

cyan() {
  paint "${C_BOLD}${C_CYAN}" "$*"
}

green() {
  paint "${C_BOLD}${C_GREEN}" "$*"
}

yellow() {
  paint "${C_BOLD}${C_YELLOW}" "$*"
}

red() {
  paint "${C_BOLD}${C_RED}" "$*"
}

section() {
  printf '\n%s %s\n' "$(blue '==>')" "$(bold "$*")"
}

info() {
  printf '%s %s\n' "$(blue '[info]')" "$*"
}

success() {
  printf '%s %s\n' "$(green '[ok]')" "$*"
}

warn() {
  printf '%s %s\n' "$(yellow '[warn]')" "$*" >&2
}

die() {
  YEAST_KEEP_LOGS=1
  printf '%s %s\n' "$(red '[fail]')" "$*" >&2
  if [[ -n "${LAST_LOG_FILE:-}" && -s "${LAST_LOG_FILE}" ]]; then
    printf '%s %s\n' "$(dim 'log:')" "${LAST_LOG_FILE}" >&2
    printf '%s\n' "$(dim 'last output:')" >&2
    tail -n 20 "${LAST_LOG_FILE}" >&2 || true
  fi
  exit 1
}

key_value() {
  printf '    %s %s\n' "$(dim "$1:")" "$2"
}

clear_step_line() {
  if [[ -t 1 ]]; then
    printf '\r\033[2K'
  fi
}

spinner() {
  local pid="$1"
  local label="$2"
  local frames=('|' '/' '-' $'\\')
  local i=0

  while kill -0 "${pid}" 2>/dev/null; do
    printf '\r%s %s %s' "$(cyan "[${frames[$i]}]")" "${label}" "$(dim 'working...')"
    i=$(((i + 1) % ${#frames[@]}))
    sleep 0.12
  done
  clear_step_line
}

step_log_path() {
  local label="$1"
  local slug
  slug="$(printf '%s' "${label}" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-')"
  printf '%s/%02d-%s.log' "${LOG_DIR}" "${STEP_INDEX}" "${slug}"
}

show_banner() {
  printf '%s\n' "$(bold 'Yeast Installer')"
  printf '%s\n' "$(dim 'Install dependencies, build Yeast, and prepare your Linux host for local VMs.')"
}

cleanup() {
  if [[ -n "${SUDO_KEEPALIVE_PID:-}" ]]; then
    kill "${SUDO_KEEPALIVE_PID}" >/dev/null 2>&1 || true
  fi
  if [[ "${YEAST_KEEP_LOGS}" != "1" && -n "${LOG_DIR:-}" ]]; then
    rm -rf "${LOG_DIR}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${WORKDIR:-}" ]]; then
    rm -rf "${WORKDIR}" >/dev/null 2>&1 || true
  fi
}

prepare_workspace() {
  LOG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/yeast-install-logs.XXXXXX")"
  WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/yeast-install.XXXXXX")"
  SRC_DIR="${WORKDIR}/src"
  trap cleanup EXIT
}

need_root() {
  if [[ "$(id -u)" -eq 0 ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required for package installation and writing to ${YEAST_INSTALL_DIR}"
  fi
  sudo "$@"
}

run_as_target() {
  if [[ "$(id -un)" == "${TARGET_USER}" ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required to run setup steps as ${TARGET_USER}"
  fi
  sudo -u "${TARGET_USER}" env HOME="${TARGET_HOME}" "$@"
}

detect_target_user() {
  TARGET_USER="${SUDO_USER:-${USER:-$(id -un)}}"
  if command -v getent >/dev/null 2>&1; then
    TARGET_HOME="$(getent passwd "${TARGET_USER}" | cut -d: -f6)"
  else
    TARGET_HOME="$(awk -F: -v user="${TARGET_USER}" '$1 == user { print $6 }' /etc/passwd)"
  fi
  [[ -n "${TARGET_HOME}" ]] || die "failed to resolve home directory for ${TARGET_USER}"
  TARGET_GROUP="$(id -gn "${TARGET_USER}")"
}

detect_package_manager() {
  if command -v apt-get >/dev/null 2>&1; then
    PKG_MANAGER="apt"
  elif command -v dnf >/dev/null 2>&1; then
    PKG_MANAGER="dnf"
  elif command -v yum >/dev/null 2>&1; then
    PKG_MANAGER="yum"
  elif command -v pacman >/dev/null 2>&1; then
    PKG_MANAGER="pacman"
  elif command -v zypper >/dev/null 2>&1; then
    PKG_MANAGER="zypper"
  elif command -v apk >/dev/null 2>&1; then
    PKG_MANAGER="apk"
  else
    die "unsupported Linux distribution: no known package manager found"
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      YEAST_ARCH="amd64"
      ;;
    aarch64|arm64)
      YEAST_ARCH="arm64"
      ;;
    *)
      die "unsupported CPU architecture: $(uname -m)"
      ;;
  esac
}

require_sudo_session() {
  if [[ "$(id -u)" -eq 0 ]]; then
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required for package installation and writing to ${YEAST_INSTALL_DIR}"
  fi

  info "Requesting sudo access"
  sudo -v
  (
    while true; do
      sudo -n true >/dev/null 2>&1 || exit
      sleep 30
    done
  ) &
  SUDO_KEEPALIVE_PID=$!
}

run_step() {
  local label="$1"
  shift
  local log_file
  local rc=0

  STEP_INDEX=$((STEP_INDEX + 1))
  log_file="$(step_log_path "${label}")"
  LAST_LOG_FILE="${log_file}"

  if [[ "${YEAST_INSTALL_VERBOSE}" == "1" || ! -t 1 ]]; then
    info "${label}"
    if "$@" >"${log_file}" 2>&1; then
      success "${label}"
      return 0
    fi
    rc=$?
    warn "Step failed: ${label}"
    return "${rc}"
  fi

  ("$@" >"${log_file}" 2>&1) &
  local pid=$!
  spinner "${pid}" "${label}"
  wait "${pid}" || rc=$?
  if [[ "${rc}" -eq 0 ]]; then
    success "${label}"
    return 0
  fi

  warn "Step failed: ${label}"
  return "${rc}"
}

run_required_step() {
  local label="$1"
  shift
  if ! run_step "${label}" "$@"; then
    die "unable to continue"
  fi
}

run_optional_step() {
  local label="$1"
  shift
  if run_step "${label}" "$@"; then
    return 0
  fi
  YEAST_KEEP_LOGS=1
  warn "${label} completed with warnings"
  if [[ -n "${LAST_LOG_FILE:-}" && -s "${LAST_LOG_FILE}" ]]; then
    printf '%s %s\n' "$(dim 'log:')" "${LAST_LOG_FILE}" >&2
  fi
  return 0
}

pkg_update() {
  case "${PKG_MANAGER}" in
    apt)
      need_root env DEBIAN_FRONTEND=noninteractive apt-get update
      ;;
    pacman)
      need_root pacman -Sy --noconfirm
      ;;
    apk)
      need_root apk update
      ;;
  esac
}

pkg_install() {
  case "${PKG_MANAGER}" in
    apt)
      need_root env DEBIAN_FRONTEND=noninteractive apt-get install -y "$@"
      ;;
    dnf)
      need_root dnf install -y "$@"
      ;;
    yum)
      need_root yum install -y "$@"
      ;;
    pacman)
      need_root pacman -S --noconfirm --needed "$@"
      ;;
    zypper)
      need_root zypper --non-interactive install --no-recommends "$@"
      ;;
    apk)
      need_root apk add --no-cache "$@"
      ;;
    *)
      die "unsupported package manager ${PKG_MANAGER}"
      ;;
  esac
}

install_one_of() {
  local description="$1"
  shift
  local candidate
  for candidate in "$@"; do
    # shellcheck disable=SC2086
    if pkg_install ${candidate}; then
      return 0
    fi
  done
  die "failed to install ${description} with ${PKG_MANAGER}"
}

install_base_packages() {
  case "${PKG_MANAGER}" in
    apt)
      pkg_install ca-certificates curl git openssh-client golang-go build-essential tar
      ;;
    dnf|yum)
      pkg_install ca-certificates curl git openssh-clients golang gcc tar
      ;;
    pacman)
      pkg_install ca-certificates curl git openssh go base-devel tar
      ;;
    zypper)
      pkg_install ca-certificates curl git openssh go gcc tar
      ;;
    apk)
      pkg_install ca-certificates curl git openssh-client go build-base bash tar
      ;;
  esac
}

install_virtualization_packages() {
  case "${PKG_MANAGER}" in
    apt)
      pkg_install qemu-system-x86 qemu-utils genisoimage
      ;;
    dnf|yum)
      install_one_of "QEMU virtualization packages" \
        "qemu-system-x86 qemu-img genisoimage" \
        "qemu-kvm qemu-img genisoimage" \
        "qemu-system-x86-core qemu-img genisoimage"
      ;;
    pacman)
      install_one_of "QEMU virtualization packages" \
        "qemu-base cdrtools" \
        "qemu-desktop cdrtools"
      ;;
    zypper)
      install_one_of "QEMU virtualization packages" \
        "qemu-x86 qemu-tools genisoimage" \
        "qemu-kvm qemu-tools genisoimage" \
        "qemu qemu-tools genisoimage"
      ;;
    apk)
      install_one_of "QEMU virtualization packages" \
        "qemu-system-x86_64 qemu-img cdrkit" \
        "qemu-system-x86_64 qemu-img xorriso"
      ;;
  esac
}

ensure_genisoimage_compat() {
  if command -v genisoimage >/dev/null 2>&1; then
    return
  fi
  if ! command -v xorriso >/dev/null 2>&1; then
    return
  fi

  need_root install -d "${YEAST_INSTALL_DIR}"
  cat <<'EOF' | need_root tee "${YEAST_INSTALL_DIR}/genisoimage" >/dev/null
#!/usr/bin/env bash
exec xorriso -as mkisofs "$@"
EOF
  need_root chmod 0755 "${YEAST_INSTALL_DIR}/genisoimage"
}

version_at_least() {
  local current="$1"
  local minimum="$2"
  local current_major current_minor current_patch
  local minimum_major minimum_minor minimum_patch

  IFS=. read -r current_major current_minor current_patch <<<"${current}"
  IFS=. read -r minimum_major minimum_minor minimum_patch <<<"${minimum}"

  current_major="${current_major:-0}"
  current_minor="${current_minor:-0}"
  current_patch="${current_patch:-0}"
  minimum_major="${minimum_major:-0}"
  minimum_minor="${minimum_minor:-0}"
  minimum_patch="${minimum_patch:-0}"

  ((current_major > minimum_major)) && return 0
  ((current_major < minimum_major)) && return 1
  ((current_minor > minimum_minor)) && return 0
  ((current_minor < minimum_minor)) && return 1
  ((current_patch >= minimum_patch))
}

go_version_value() {
  local output
  output="$("$1" version 2>/dev/null || true)"
  output="${output#go version go}"
  printf '%s' "${output%% *}"
}

official_go_bin_path() {
  printf '%s/go%s/bin/go' "${YEAST_GO_INSTALL_ROOT}" "${YEAST_GO_VERSION}"
}

resolve_go_bin() {
  local candidate

  candidate="$(command -v go || true)"
  if [[ -n "${candidate}" ]]; then
    printf '%s' "${candidate}"
    return 0
  fi

  candidate="$(official_go_bin_path)"
  if [[ -x "${candidate}" ]]; then
    printf '%s' "${candidate}"
    return 0
  fi

  return 1
}

download_official_go() {
  local archive
  local url
  local install_dir

  archive="${WORKDIR}/go${YEAST_GO_VERSION}.linux-${YEAST_ARCH}.tar.gz"
  url="https://go.dev/dl/go${YEAST_GO_VERSION}.linux-${YEAST_ARCH}.tar.gz"
  install_dir="${YEAST_GO_INSTALL_ROOT}/go${YEAST_GO_VERSION}"

  info "Installing Go ${YEAST_GO_VERSION} from ${url}"
  curl -fL --retry 3 --retry-delay 2 -o "${archive}" "${url}"

  if [[ -n "${YEAST_GO_TARBALL_SHA256}" ]]; then
    printf '%s  %s\n' "${YEAST_GO_TARBALL_SHA256}" "${archive}" | sha256sum -c -
  else
    warn "YEAST_GO_TARBALL_SHA256 is not set; relying on HTTPS transport for Go toolchain download"
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
    local current
    current="$(go_version_value "${GO_BIN}")"
    if [[ -n "${current}" ]] && version_at_least "${current}" "${YEAST_MIN_GO_VERSION}"; then
      info "Using Go ${current} at ${GO_BIN}"
      return
    fi
    warn "Found Go ${current:-unknown}, but Yeast requires Go ${YEAST_MIN_GO_VERSION}+"
  fi

  download_official_go

  local installed
  installed="$(go_version_value "${GO_BIN}")"
  if [[ -z "${installed}" ]] || ! version_at_least "${installed}" "${YEAST_MIN_GO_VERSION}"; then
    die "installed Go ${installed:-unknown}, but Yeast requires Go ${YEAST_MIN_GO_VERSION}+"
  fi
}

ensure_iso_builder() {
  if command -v genisoimage >/dev/null 2>&1 || command -v mkisofs >/dev/null 2>&1; then
    return
  fi
  die "required ISO builder is missing: install genisoimage or mkisofs"
}

ensure_required_tools() {
  local tool
  for tool in git curl tar qemu-system-x86_64 qemu-img ssh ssh-keygen; do
    if ! command -v "${tool}" >/dev/null 2>&1; then
      die "required tool ${tool} is still missing after installation"
    fi
  done
  ensure_iso_builder
  ensure_go_toolchain
}

clone_source() {
  git clone --depth 1 --branch "${YEAST_REF}" "${YEAST_REPO_URL}" "${SRC_DIR}" || \
    die "failed to clone ${YEAST_REPO_URL}; if the repo is private, authenticate first or set YEAST_REPO_URL to an accessible source"
}

build_source() {
  local go_bin
  ensure_go_toolchain
  go_bin="$(resolve_go_bin || true)"
  [[ -n "${go_bin}" ]] || die "failed to resolve Go toolchain after installation"
  (
    cd "${SRC_DIR}"
    "${go_bin}" build -o "${WORKDIR}/yeast" ./cmd/yeast
  )
}

install_binary() {
  need_root install -d "${YEAST_INSTALL_DIR}"
  need_root install -m 0755 "${WORKDIR}/yeast" "${YEAST_BIN_PATH}"
}

ensure_user_paths() {
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast"
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast/cache"
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast/cache/images"
}

ensure_ssh_key() {
  if [[ -f "${TARGET_HOME}/.ssh/id_ed25519.pub" || -f "${TARGET_HOME}/.ssh/id_rsa.pub" ]]; then
    info "SSH public key already exists for ${TARGET_USER}"
    return
  fi

  need_root install -d -m 0700 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.ssh"
  run_as_target ssh-keygen -t ed25519 -N "" -C "yeast@$(hostname)" -f "${TARGET_HOME}/.ssh/id_ed25519"
}

ensure_kvm_group_membership() {
  if command -v getent >/dev/null 2>&1; then
    if ! getent group kvm >/dev/null 2>&1; then
      warn "kvm group was not found; you may need to configure KVM manually"
      return
    fi
  elif ! grep -q '^kvm:' /etc/group; then
    warn "kvm group was not found; you may need to configure KVM manually"
    return
  fi

  if id -nG "${TARGET_USER}" | tr ' ' '\n' | grep -qx kvm; then
    info "${TARGET_USER} is already in the kvm group"
    return
  fi

  need_root usermod -aG kvm "${TARGET_USER}"
  KVM_GROUP_UPDATED=1
}

check_kvm_access() {
  if [[ ! -e /dev/kvm ]]; then
    warn "/dev/kvm does not exist. Enable virtualization in BIOS/UEFI and load KVM modules before running VMs."
    return
  fi
  if [[ ! -c /dev/kvm ]]; then
    warn "/dev/kvm exists but is not a character device"
    return
  fi
  if run_as_target test -r /dev/kvm -a -w /dev/kvm; then
    info "${TARGET_USER} can access /dev/kvm"
    return
  fi
  warn "${TARGET_USER} cannot access /dev/kvm in the current session"
}

run_post_install_checks() {
  run_as_target "${YEAST_BIN_PATH}" doctor
}

print_summary() {
  local go_bin
  go_bin="$(resolve_go_bin || true)"

  section "Installation Complete"
  key_value "Binary" "${YEAST_BIN_PATH}"
  key_value "Repo" "${YEAST_REPO_URL}"
  key_value "Ref" "${YEAST_REF}"
  key_value "Target user" "${TARGET_USER}"
  key_value "Package manager" "${PKG_MANAGER}"
  key_value "Go" "$("${go_bin:-false}" version 2>/dev/null || printf 'unknown')"
  printf '\n'
  info "Next steps"
  key_value "1" "yeast pull --list"
  key_value "2" "mkdir my-project && cd my-project"
  key_value "3" "yeast init"
  key_value "4" "yeast pull ubuntu-24.04"
  key_value "5" "yeast up"
  if [[ "${KVM_GROUP_UPDATED:-0}" -eq 1 ]]; then
    printf '\n'
    warn "${TARGET_USER} was added to the kvm group. Log out and back in before starting Yeast so the new group membership is active."
  fi
}

main() {
  [[ "$(uname -s)" == "Linux" ]] || die "Yeast installer currently supports Linux only"

  prepare_workspace
  show_banner
  detect_arch
  detect_target_user
  detect_package_manager
  require_sudo_session

  section "Install Plan"
  key_value "Target user" "${TARGET_USER}"
  key_value "Home" "${TARGET_HOME}"
  key_value "Package manager" "${PKG_MANAGER}"
  key_value "Architecture" "${YEAST_ARCH}"
  key_value "Repo" "${YEAST_REPO_URL}"
  key_value "Ref" "${YEAST_REF}"
  key_value "Install path" "${YEAST_BIN_PATH}"

  section "Installing Dependencies"
  run_required_step "Refreshing package metadata" pkg_update
  run_required_step "Installing base packages" install_base_packages
  run_required_step "Installing virtualization packages" install_virtualization_packages
  run_required_step "Preparing genisoimage compatibility" ensure_genisoimage_compat
  run_required_step "Verifying required tools" ensure_required_tools

  section "Building Yeast"
  run_required_step "Cloning repository" clone_source
  run_required_step "Building CLI binary" build_source
  run_required_step "Installing yeast binary" install_binary

  section "Configuring User Environment"
  run_required_step "Creating Yeast directories" ensure_user_paths
  run_required_step "Ensuring SSH key" ensure_ssh_key
  run_required_step "Ensuring kvm group membership" ensure_kvm_group_membership
  run_optional_step "Checking KVM access" check_kvm_access

  section "Running Validation"
  run_optional_step "Running yeast doctor" run_post_install_checks

  print_summary
}

STEP_INDEX=0
main "$@"
