#!/usr/bin/env bash
# Yeast Smart Installer
# Detects distro, checks prerequisites, installs only what's missing,
# handles WSL, and provides actionable recovery hints.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
#   bash install.sh

set -euo pipefail

# ─── Configuration ──────────────────────────────────────────────────────
YEAST_REPO_URL="${YEAST_REPO_URL:-https://github.com/Twarga/yeast.git}"
YEAST_REF="${YEAST_REF:-v1.0.1}"
YEAST_INSTALL_DIR="${YEAST_INSTALL_DIR:-/usr/local/bin}"
YEAST_BIN_PATH="${YEAST_INSTALL_DIR}/yeast"
YEAST_EXPECTED_VERSION="${YEAST_EXPECTED_VERSION:-}"
YEAST_INSTALL_VERBOSE="${YEAST_INSTALL_VERBOSE:-0}"
YEAST_KEEP_LOGS="${YEAST_KEEP_LOGS:-0}"
YEAST_MIN_GO_VERSION="${YEAST_MIN_GO_VERSION:-1.25.0}"
YEAST_GO_VERSION="${YEAST_GO_VERSION:-1.26.3}"
YEAST_GO_INSTALL_ROOT="${YEAST_GO_INSTALL_ROOT:-/usr/local/lib/yeast/go}"
YEAST_GO_TARBALL_SHA256="${YEAST_GO_TARBALL_SHA256:-}"
GO_BIN=""

# ─── Colors ───────────────────────────────────────────────────────────
if [[ -t 1 && -z "${NO_COLOR:-}" && "${TERM:-}" != "dumb" ]]; then
  C_RESET=$'\033[0m'; C_BOLD=$'\033[1m'; C_DIM=$'\033[2m'
  C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'
  C_BLUE=$'\033[34m'; C_CYAN=$'\033[36m'; C_MAGENTA=$'\033[35m'
else
  C_RESET=""; C_BOLD=""; C_DIM=""; C_RED=""; C_GREEN=""
  C_YELLOW=""; C_BLUE=""; C_CYAN=""; C_MAGENTA=""
fi

paint() { local code="$1"; shift; printf '%b%s%b' "${code}" "$*" "${C_RESET}"; }
bold()  { paint "${C_BOLD}" "$*"; }
dim()   { paint "${C_DIM}" "$*"; }
blue()  { paint "${C_BOLD}${C_BLUE}" "$*"; }
cyan()  { paint "${C_BOLD}${C_CYAN}" "$*"; }
green() { paint "${C_BOLD}${C_GREEN}" "$*"; }
yellow(){ paint "${C_BOLD}${C_YELLOW}" "$*"; }
red()   { paint "${C_BOLD}${C_RED}" "$*"; }
mg()    { paint "${C_BOLD}${C_MAGENTA}" "$*"; }

section()  { printf '\n%s %s\n' "$(blue '==>')" "$(bold "$*")"; }
info()     { printf '%s %s\n' "$(blue '[info]')" "$*"; }
success()  { printf '%s %s\n' "$(green '[ok]')" "$*"; }
warn()     { printf '%s %s\n' "$(yellow '[warn]')" "$*" >&2; }
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
key_value() { printf '    %s %s\n' "$(dim "$1:")" "$2"; }

# ─── Spinner ────────────────────────────────────────────────────────
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

step_log_path() {
  local label="$1" slug
  slug="$(printf '%s' "${label}" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-')"
  printf '%s/%02d-%s.log' "${LOG_DIR}" "${STEP_INDEX}" "${slug}"
}

run_step() {
  local label="$1"; shift
  local log_file rc=0
  STEP_INDEX=$((STEP_INDEX + 1))
  log_file="$(step_log_path "${label}")"
  LAST_LOG_FILE="${log_file}"
  if [[ "${YEAST_INSTALL_VERBOSE}" == "1" || ! -t 1 ]]; then
    info "${label}"
    if "$@" >"${log_file}" 2>&1; then success "${label}"; return 0; fi
    rc=$?; warn "Step failed: ${label}"; return "${rc}"
  fi
  ("$@" >"${log_file}" 2>&1) & local pid=$!
  spinner "${pid}" "${label}"
  wait "${pid}" || rc=$?
  if [[ "${rc}" -eq 0 ]]; then success "${label}"; return 0; fi
  warn "Step failed: ${label}"; return "${rc}"
}

run_required() {
  local label="$1"; shift
  if ! run_step "${label}" "$@"; then die "unable to continue"; fi
}

run_optional() {
  local label="$1"; shift
  if run_step "${label}" "$@"; then return 0; fi
  YEAST_KEEP_LOGS=1
  warn "${label} completed with warnings"
  if [[ -n "${LAST_LOG_FILE:-}" && -s "${LAST_LOG_FILE}" ]]; then
    printf '%s %s\n' "$(dim 'log:')" "${LAST_LOG_FILE}" >&2
  fi
  return 0
}

# ─── Helpers ────────────────────────────────────────────────────────
need_root() {
  if [[ "$(id -u)" -eq 0 ]]; then "$@"; return; fi
  if ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required for package installation and writing to ${YEAST_INSTALL_DIR}"
  fi
  sudo "$@"
}

run_as_target() {
  if [[ "$(id -un)" == "${TARGET_USER}" ]]; then "$@"; return; fi
  if ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required to run setup steps as ${TARGET_USER}"
  fi
  sudo -u "${TARGET_USER}" env HOME="${TARGET_HOME}" "$@"
}

cleanup() {
  [[ -n "${SUDO_KEEPALIVE_PID:-}" ]] && kill "${SUDO_KEEPALIVE_PID}" >/dev/null 2>&1 || true
  if [[ "${YEAST_KEEP_LOGS}" != "1" && -n "${LOG_DIR:-}" ]]; then
    rm -rf "${LOG_DIR}" >/dev/null 2>&1 || true
  fi
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

# ─── Smart System Detection ─────────────────────────────────────────
detect_environment() {
  IS_WSL=0; WSL_VERSION=""
  if [[ -f /proc/version ]] && grep -qi microsoft /proc/version; then
    if [[ -f /proc/sys/kernel/osrelease ]] && grep -q "WSL2" /proc/sys/kernel/osrelease; then
      IS_WSL=1; WSL_VERSION="2"
    elif [[ -f /proc/sys/kernel/osrelease ]] && grep -q "WSL" /proc/sys/kernel/osrelease; then
      IS_WSL=1; WSL_VERSION="1"
    else
      IS_WSL=1; WSL_VERSION="unknown"
    fi
  fi

  IS_CONTAINER=0
  if [[ -f /.dockerenv ]] || grep -q docker /proc/self/cgroup 2>/dev/null; then
    IS_CONTAINER=1
  fi

  KERNEL="$(uname -r)"
  ARCH="$(uname -m)"
  case "${ARCH}" in
    x86_64|amd64) YEAST_ARCH="amd64" ;;
    aarch64|arm64) YEAST_ARCH="arm64" ;;
    *) die "unsupported CPU architecture: ${ARCH}" ;;
  esac
}

detect_distro() {
  DISTRO=""; DISTRO_VERSION=""; DISTRO_ID=""
  PKG_MANAGER=""

  if [[ -f /etc/os-release ]]; then
    DISTRO_ID="$(source /etc/os-release && printf '%s' "${ID:-}")"
    DISTRO_VERSION="$(source /etc/os-release && printf '%s' "${VERSION_ID:-}")"
    DISTRO="$(source /etc/os-release && printf '%s' "${NAME:-}")"
  fi

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

# ─── Smart Prerequisite Checker ─────────────────────────────────────

# Check if a command exists and report its version
check_tool() {
  local tool="$1" version_flag="${2:---version}"
  if command -v "${tool}" >/dev/null 2>&1; then
    local ver; ver="$(${tool} ${version_flag} 2>/dev/null | head -1 || true)"
    printf 'found\t%s' "${ver}"
    return 0
  fi
  printf 'missing'
  return 1
}

# Check if a package is installed (distro-aware)
is_pkg_installed() {
  local pkg="$1"
  case "${PKG_MANAGER}" in
    apt)
      dpkg -l "${pkg}" 2>/dev/null | grep -q "^ii" && return 0
      ;;
    dnf|yum)
      rpm -q "${pkg}" >/dev/null 2>&1 && return 0
      ;;
    pacman)
      pacman -Q "${pkg}" >/dev/null 2>&1 && return 0
      ;;
    zypper)
      rpm -q "${pkg}" >/dev/null 2>&1 && return 0
      ;;
    apk)
      apk info -e "${pkg}" >/dev/null 2>&1 && return 0
      ;;
  esac
  return 1
}

check_prerequisites() {
  section "Checking System"
  key_value "OS" "${DISTRO:-Linux} ${DISTRO_VERSION:-}"
  key_value "Kernel" "${KERNEL}"
  key_value "Architecture" "${YEAST_ARCH}"
  key_value "Package manager" "${PKG_MANAGER}"

  if [[ "${IS_WSL}" -eq 1 ]]; then
    key_value "Environment" "WSL${WSL_VERSION}"
    if [[ "${WSL_VERSION}" == "1" ]]; then
      die "WSL1 is not supported. Upgrade to WSL2: https://aka.ms/wsl2-install"
    fi
    if [[ ! -e /dev/kvm ]]; then
      warn "WSL2 detected without KVM. VMs will work in TCG (software emulation) mode, which is ~10x slower."
      info "To enable KVM in WSL2, enable nested virtualization in Windows. See docs for details."
    fi
  elif [[ "${IS_CONTAINER}" -eq 1 ]]; then
    key_value "Environment" "Container"
    warn "Running inside a container. KVM acceleration is unlikely to work. VMs will use TCG (slow)."
  fi

  section "Checking Prerequisites"
  local missing=() present=()

  # Check each prerequisite
  local git_status; git_status="$(check_tool git)"
  if [[ "${git_status}" == found* ]]; then
    present+=("git")
    key_value "git" "$(echo "${git_status}" | cut -f2)"
  else
    missing+=("git")
    key_value "git" "$(red 'missing')"
  fi

  local curl_status; curl_status="$(check_tool curl)"
  if [[ "${curl_status}" == found* ]]; then
    present+=("curl")
    key_value "curl" "$(echo "${curl_status}" | cut -f2)"
  else
    missing+=("curl")
    key_value "curl" "$(red 'missing')"
  fi

  local tar_status; tar_status="$(check_tool tar)"
  if [[ "${tar_status}" == found* ]]; then
    present+=("tar")
    key_value "tar" "$(echo "${tar_status}" | cut -f2)"
  else
    missing+=("tar")
    key_value "tar" "$(red 'missing')"
  fi

  local ssh_status; ssh_status="$(check_tool ssh)"
  if [[ "${ssh_status}" == found* ]]; then
    present+=("ssh")
    key_value "ssh" "$(echo "${ssh_status}" | cut -f2)"
  else
    missing+=("ssh")
    key_value "ssh" "$(red 'missing')"
  fi

  # QEMU checks
  local qemu_status qemu_img_status
  qemu_status="$(check_tool qemu-system-x86_64)"
  qemu_img_status="$(check_tool qemu-img)"
  if [[ "${qemu_status}" == found* && "${qemu_img_status}" == found* ]]; then
    present+=("qemu-system-x86_64 qemu-img")
    key_value "QEMU" "$(echo "${qemu_status}" | cut -f2)"
  else
    missing+=("qemu")
    if [[ "${qemu_status}" == found* ]]; then
      key_value "qemu-system-x86_64" "$(echo "${qemu_status}" | cut -f2)"
    else
      key_value "qemu-system-x86_64" "$(red 'missing')"
    fi
    if [[ "${qemu_img_status}" == found* ]]; then
      key_value "qemu-img" "$(echo "${qemu_img_status}" | cut -f2)"
    else
      key_value "qemu-img" "$(red 'missing')"
    fi
  fi

  # ISO builder check
  local iso_status="missing"
  if command -v genisoimage >/dev/null 2>&1; then
    iso_status="genisoimage"
    present+=("genisoimage")
  elif command -v mkisofs >/dev/null 2>&1; then
    iso_status="mkisofs"
    present+=("mkisofs")
  elif command -v xorriso >/dev/null 2>&1; then
    iso_status="xorriso"
    present+=("xorriso")
  else
    missing+=("genisoimage/mkisofs/xorriso")
  fi
  key_value "ISO builder" "${iso_status}"

  # KVM check
  local kvm_status=""
  if [[ -e /dev/kvm ]]; then
    if run_as_target test -r /dev/kvm -a -w /dev/kvm 2>/dev/null; then
      kvm_status="$(green 'available')"
      present+=("kvm")
    else
      kvm_status="$(yellow 'present but not accessible')"
      # Not in missing — we'll try to fix permissions
    fi
  else
    if [[ "${IS_WSL}" -eq 1 ]]; then
      kvm_status="$(yellow 'not available in WSL2 (TCG fallback)')"
    elif [[ "${IS_CONTAINER}" -eq 1 ]]; then
      kvm_status="$(yellow 'not available in container (TCG fallback)')"
    else
      kvm_status="$(red 'missing — check BIOS/UEFI virtualization settings')"
    fi
  fi
  key_value "KVM" "${kvm_status}"

  # Go check
  local go_status; go_status="$(resolve_go_bin)"
  if [[ -n "${go_status}" ]]; then
    local go_ver; go_ver="$(go_version_value "${go_status}")"
    if version_at_least "${go_ver}" "${YEAST_MIN_GO_VERSION}"; then
      present+=("go")
      key_value "Go" "${go_ver} (>= ${YEAST_MIN_GO_VERSION})"
    else
      key_value "Go" "${go_ver} $(yellow "(needs >= ${YEAST_MIN_GO_VERSION})")"
      missing+=("go (upgrade)")
    fi
  else
    key_value "Go" "$(red 'missing')"
    missing+=("go")
  fi

  # Summary
  printf '\n'
  if [[ ${#missing[@]} -eq 0 ]]; then
    success "All prerequisites satisfied"
  else
    info "Missing prerequisites: ${missing[*]}"
  fi

  PREREQ_MISSING="${missing[*]}"
  PREREQ_PRESENT="${present[*]}"
}

# ─── Smart Package Installation ───────────────────────────────────────

pkg_update() {
  case "${PKG_MANAGER}" in
    apt) need_root env DEBIAN_FRONTEND=noninteractive apt-get update ;;
    pacman) need_root pacman -Sy --noconfirm ;;
    apk) need_root apk update ;;
  esac
}

pkg_install() {
  case "${PKG_MANAGER}" in
    apt) need_root env DEBIAN_FRONTEND=noninteractive apt-get install -y "$@" ;;
    dnf) need_root dnf install -y "$@" ;;
    yum) need_root yum install -y "$@" ;;
    pacman) need_root pacman -S --noconfirm --needed "$@" ;;
    zypper) need_root zypper --non-interactive install --no-recommends "$@" ;;
    apk) need_root apk add --no-cache "$@" ;;
    *) die "unsupported package manager ${PKG_MANAGER}" ;;
  esac
}

# Install a package only if it's not already installed
ensure_pkg() {
  local pkg="$1" description="${2:-$1}"
  if is_pkg_installed "${pkg}"; then
    info "${description} already installed, skipping"
    return 0
  fi
  run_step "Installing ${description}" pkg_install "${pkg}"
}

# Try multiple package names, stop at first success
try_pkgs() {
  local description="$1"; shift
  local pkg
  for pkg in "$@"; do
    if is_pkg_installed "${pkg}"; then
      info "${description} (${pkg}) already installed"
      return 0
    fi
    if pkg_install "${pkg}"; then
      success "Installed ${description} (${pkg})"
      return 0
    fi
  done
  die "failed to install ${description} with ${PKG_MANAGER}"
}

install_base_packages() {
  section "Installing Base Packages"
  case "${PKG_MANAGER}" in
    apt)
      ensure_pkg "ca-certificates" "CA certificates"
      ensure_pkg "curl" "curl"
      ensure_pkg "git" "Git"
      ensure_pkg "openssh-client" "SSH client"
      ensure_pkg "tar" "tar"
      ;;
    dnf|yum)
      ensure_pkg "ca-certificates" "CA certificates"
      ensure_pkg "curl" "curl"
      ensure_pkg "git" "Git"
      ensure_pkg "openssh-clients" "SSH client"
      ensure_pkg "tar" "tar"
      ;;
    pacman)
      ensure_pkg "ca-certificates" "CA certificates"
      ensure_pkg "curl" "curl"
      ensure_pkg "git" "Git"
      ensure_pkg "openssh" "SSH"
      ensure_pkg "tar" "tar"
      ;;
    zypper)
      ensure_pkg "ca-certificates" "CA certificates"
      ensure_pkg "curl" "curl"
      ensure_pkg "git" "Git"
      ensure_pkg "openssh" "SSH"
      ensure_pkg "tar" "tar"
      ;;
    apk)
      ensure_pkg "ca-certificates" "CA certificates"
      ensure_pkg "curl" "curl"
      ensure_pkg "git" "Git"
      ensure_pkg "openssh-client" "SSH client"
      ensure_pkg "tar" "tar"
      ensure_pkg "bash" "bash"
      ;;
  esac
}

install_virtualization_packages() {
  section "Installing Virtualization Packages"
  case "${PKG_MANAGER}" in
    apt)
      try_pkgs "QEMU" "qemu-system-x86" "qemu-system-x86-64" "qemu-kvm"
      try_pkgs "qemu-img" "qemu-utils" "qemu-img"
      try_pkgs "ISO builder" "genisoimage" "cdrkit-genisoimage" "xorriso"
      ;;
    dnf|yum)
      try_pkgs "QEMU" "qemu-system-x86" "qemu-kvm" "qemu-system-x86-core"
      try_pkgs "qemu-img" "qemu-img" "qemu-utils"
      try_pkgs "ISO builder" "genisoimage" "mkisofs" "xorriso"
      ;;
    pacman)
      try_pkgs "QEMU" "qemu-base" "qemu-desktop" "qemu-full"
      try_pkgs "ISO builder" "cdrtools" "cdrkit" "xorriso"
      ;;
    zypper)
      try_pkgs "QEMU" "qemu-x86" "qemu-kvm" "qemu"
      try_pkgs "qemu-tools" "qemu-tools" "qemu-img"
      try_pkgs "ISO builder" "genisoimage" "mkisofs" "xorriso"
      ;;
    apk)
      try_pkgs "QEMU" "qemu-system-x86_64" "qemu-system-x86" "qemu"
      try_pkgs "qemu-img" "qemu-img"
      try_pkgs "ISO builder" "cdrkit" "xorriso" "mkisofs"
      ;;
  esac
}

# ─── KVM Setup ────────────────────────────────────────────────────────
fix_kvm_permissions() {
  if [[ ! -e /dev/kvm ]]; then return 0; fi
  if run_as_target test -r /dev/kvm -a -w /dev/kvm 2>/dev/null; then return 0; fi

  section "Fixing KVM Permissions"

  # Check if kvm group exists
  if ! getent group kvm >/dev/null 2>&1 && ! grep -q '^kvm:' /etc/group; then
    warn "kvm group not found; cannot add user"
    return 0
  fi

  if ! id -nG "${TARGET_USER}" | tr ' ' '\n' | grep -qx kvm; then
    run_step "Adding ${TARGET_USER} to kvm group" need_root usermod -aG kvm "${TARGET_USER}"
    KVM_GROUP_UPDATED=1
  fi

  # Fix device permissions
  if [[ "$(stat -c '%a' /dev/kvm 2>/dev/null || true)" != "660" ]]; then
    run_step "Fixing /dev/kvm permissions" need_root chmod 660 /dev/kvm
  fi

  if [[ "$(stat -c '%G' /dev/kvm 2>/dev/null || true)" != "kvm" ]]; then
    run_step "Setting /dev/kvm group" need_root chown root:kvm /dev/kvm
  fi

  # Reload udev rules if needed
  if command -v udevadm >/dev/null 2>&1; then
    run_optional "Reloading udev rules" need_root udevadm control --reload-rules
  fi
}

load_kvm_modules() {
  if lsmod | grep -q "^kvm "; then
    info "KVM module already loaded"
    return 0
  fi
  if [[ ! -d /sys/module/kvm ]]; then
    run_step "Loading KVM module" need_root modprobe kvm
  fi
  if grep -q "vmx\|svm" /proc/cpuinfo 2>/dev/null; then
    local module="kvm_intel"
    grep -q "AuthenticAMD" /proc/cpuinfo && module="kvm_amd"
    if [[ ! -d "/sys/module/${module}" ]]; then
      run_optional "Loading ${module} module" need_root modprobe "${module}"
    fi
  fi
}

# ─── Go Toolchain ───────────────────────────────────────────────────
download_official_go() {
  local archive url install_dir
  archive="${WORKDIR}/go${YEAST_GO_VERSION}.linux-${YEAST_ARCH}.tar.gz"
  url="https://go.dev/dl/go${YEAST_GO_VERSION}.linux-${YEAST_ARCH}.tar.gz"
  install_dir="${YEAST_GO_INSTALL_ROOT}/go${YEAST_GO_VERSION}"

  info "Downloading Go ${YEAST_GO_VERSION}..."
  curl -fL --retry 3 --retry-delay 2 -o "${archive}" "${url}"

  if [[ -n "${YEAST_GO_TARBALL_SHA256}" ]]; then
    printf '%s  %s\n' "${YEAST_GO_TARBALL_SHA256}" "${archive}" | sha256sum -c -
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
  local installed; installed="$(go_version_value "${GO_BIN}")"
  if [[ -z "${installed}" ]] || ! version_at_least "${installed}" "${YEAST_MIN_GO_VERSION}"; then
    die "installed Go ${installed:-unknown}, need ${YEAST_MIN_GO_VERSION}+"
  fi
}

# ─── genisoimage Compatibility ──────────────────────────────────────
ensure_genisoimage_compat() {
  if command -v genisoimage >/dev/null 2>&1; then return 0; fi
  if command -v mkisofs >/dev/null 2>&1; then return 0; fi
  if ! command -v xorriso >/dev/null 2>&1; then return 0; fi

  need_root install -d "${YEAST_INSTALL_DIR}"
  cat <<'EOF' | need_root tee "${YEAST_INSTALL_DIR}/genisoimage" >/dev/null
#!/usr/bin/env bash
exec xorriso -as mkisofs "$@"
EOF
  need_root chmod 0755 "${YEAST_INSTALL_DIR}/genisoimage"
  info "Created genisoimage wrapper using xorriso"
}

# ─── Build Yeast ──────────────────────────────────────────────────────
clone_source() {
  git clone --depth 1 --branch "${YEAST_REF}" "${YEAST_REPO_URL}" "${SRC_DIR}" || \
    die "failed to clone ${YEAST_REPO_URL}"
}

build_source() {
  local go_bin version
  ensure_go_toolchain
  go_bin="$(resolve_go_bin || true)"
  [[ -n "${go_bin}" ]] || die "failed to resolve Go toolchain"
  version="$(expected_version || true)"
  [[ -n "${version}" ]] || version="0.0.0-dev"
  (
    cd "${SRC_DIR}"
    "${go_bin}" build \
      -trimpath \
      -ldflags "-s -w -X yeast/internal/app.Version=${version}" \
      -o "${WORKDIR}/yeast" \
      ./cmd/yeast
  )
}

expected_version() {
  if [[ -n "${YEAST_EXPECTED_VERSION}" ]]; then
    printf '%s' "${YEAST_EXPECTED_VERSION}"; return 0
  fi
  if [[ "${YEAST_REF}" =~ ^v[0-9]+[.][0-9]+[.][0-9]+([-+][A-Za-z0-9._-]+)?$ ]]; then
    printf '%s' "${YEAST_REF}"; return 0
  fi
  return 1
}

install_binary() {
  need_root install -d "${YEAST_INSTALL_DIR}"
  need_root install -m 0755 "${WORKDIR}/yeast" "${YEAST_BIN_PATH}"
}

verify_binary() {
  local actual expected
  [[ -x "${YEAST_BIN_PATH}" ]] || die "installed binary not executable"
  actual="$("${YEAST_BIN_PATH}" version 2>/dev/null || true)"
  [[ -n "${actual}" ]] || die "installed binary did not print version"
  expected="$(expected_version || true)"
  if [[ -n "${expected}" && "${actual}" != "${expected}" ]]; then
    die "version mismatch: expected ${expected}, got ${actual}"
  fi
  info "Installed Yeast: ${actual}"
}

# ─── User Environment ─────────────────────────────────────────────────
detect_target_user() {
  TARGET_USER="${SUDO_USER:-${USER:-$(id -un)}}"
  if command -v getent >/dev/null 2>&1; then
    TARGET_HOME="$(getent passwd "${TARGET_USER}" | cut -d: -f6)"
  else
    TARGET_HOME="$(awk -F: -v user="${TARGET_USER}" '$1 == user { print $6 }' /etc/passwd)"
  fi
  [[ -n "${TARGET_HOME}" ]] || die "failed to resolve home for ${TARGET_USER}"
  TARGET_GROUP="$(id -gn "${TARGET_USER}")"
}

ensure_user_paths() {
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast"
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast/cache"
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast/cache/images"
}

ensure_ssh_key() {
  if [[ -f "${TARGET_HOME}/.ssh/id_ed25519.pub" || -f "${TARGET_HOME}/.ssh/id_rsa.pub" ]]; then
    info "SSH key already exists"
    return
  fi
  need_root install -d -m 0700 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.ssh"
  run_as_target ssh-keygen -t ed25519 -N "" -C "yeast@$(hostname)" -f "${TARGET_HOME}/.ssh/id_ed25519"
}

# ─── Main ─────────────────────────────────────────────────────────────
show_banner() {
  printf '%s\n' "$(bold 'Yeast Installer')"
  printf '%s\n' "$(dim 'Smart distro-aware installer for Yeast — Linux VM orchestration')"
}

require_sudo() {
  if [[ "$(id -u)" -eq 0 ]]; then return; fi
  if ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required for package installation"
  fi
  info "Requesting sudo access"
  sudo -v
  (while true; do sudo -n true >/dev/null 2>&1 || exit; sleep 30; done) &
  SUDO_KEEPALIVE_PID=$!
}

print_summary() {
  local go_bin; go_bin="$(resolve_go_bin || true)"
  section "Installation Complete"
  key_value "Binary" "${YEAST_BIN_PATH}"
  key_value "Version" "$("${YEAST_BIN_PATH}" version 2>/dev/null || echo 'unknown')"
  key_value "Target user" "${TARGET_USER}"
  key_value "Package manager" "${PKG_MANAGER}"
  key_value "Go" "$(${go_bin:-false} version 2>/dev/null || echo 'unknown')"
  printf '\n'
  info "Next steps"
  key_value "1" "yeast doctor"
  key_value "2" "yeast pull --list"
  key_value "3" "mkdir my-project && cd my-project"
  key_value "4" "yeast init"
  key_value "5" "yeast pull ubuntu-24.04"
  key_value "6" "yeast up"
  if [[ "${KVM_GROUP_UPDATED:-0}" -eq 1 ]]; then
    printf '\n'
    yellow "${TARGET_USER} was added to the kvm group. Log out and back in for KVM access."
  fi
  if [[ "${IS_WSL}" -eq 1 && ! -e /dev/kvm ]]; then
    printf '\n'
    yellow "WSL2 detected without KVM. VMs will use TCG (software emulation) mode."
    dim "To enable KVM:"
    dim "  1. Open PowerShell as Administrator"
    dim "  2. Run: wsl --shutdown"
    dim "  3. Create %USERPROFILE%\\.wslconfig with:"
    dim "     [wsl2]"
    dim "     nestedVirtualization=true"
    dim "  4. Restart WSL"
  fi
}

main() {
  [[ "$(uname -s)" == "Linux" ]] || die "Yeast installer supports Linux only"

  LOG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/yeast-install-logs.XXXXXX")"
  WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/yeast-install.XXXXXX")"
  SRC_DIR="${WORKDIR}/src"
  trap cleanup EXIT
  STEP_INDEX=0
  KVM_GROUP_UPDATED=0

  show_banner
  detect_environment
  detect_distro
  detect_target_user
  require_sudo

  section "Install Plan"
  key_value "Target user" "${TARGET_USER}"
  key_value "Home" "${TARGET_HOME}"
  key_value "Distro" "${DISTRO:-unknown} ${DISTRO_VERSION:-}"
  key_value "Package manager" "${PKG_MANAGER}"
  key_value "Architecture" "${YEAST_ARCH}"
  key_value "Repo" "${YEAST_REPO_URL}"
  key_value "Ref" "${YEAST_REF}"
  key_value "Install path" "${YEAST_BIN_PATH}"

  # Phase 1: Smart prerequisite check
  check_prerequisites

  # Phase 2: Install what's missing
  if [[ -n "${PREREQ_MISSING:-}" ]]; then
    section "Installing Missing Dependencies"
    run_required "Refreshing package metadata" pkg_update
    install_base_packages
    install_virtualization_packages
    run_required "Preparing genisoimage compatibility" ensure_genisoimage_compat
  else
    section "All Dependencies Satisfied"
    success "No packages to install"
  fi

  # Phase 3: Fix KVM if available
  if [[ -e /dev/kvm ]]; then
    load_kvm_modules
    fix_kvm_permissions
  fi

  # Phase 4: Build and install
  section "Building Yeast"
  run_required "Cloning repository" clone_source
  run_required "Building CLI binary" build_source
  run_required "Installing yeast binary" install_binary
  run_required "Verifying installed binary" verify_binary

  # Phase 5: Configure user environment
  section "Configuring User Environment"
  run_required "Creating Yeast directories" ensure_user_paths
  run_required "Ensuring SSH key" ensure_ssh_key

  # Phase 6: Validation
  section "Running Validation"
  run_optional "Running yeast doctor" run_as_target "${YEAST_BIN_PATH}" doctor

  print_summary
}

main "$@"
