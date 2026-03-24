#!/usr/bin/env bash
set -euo pipefail

YEAST_REPO_URL="${YEAST_REPO_URL:-https://github.com/Twarga/yeast.git}"
YEAST_REF="${YEAST_REF:-dev}"
YEAST_INSTALL_DIR="${YEAST_INSTALL_DIR:-/usr/local/bin}"
YEAST_BIN_PATH="${YEAST_INSTALL_DIR}/yeast"

log() {
  printf '[yeast-install] %s\n' "$*"
}

warn() {
  printf '[yeast-install] warning: %s\n' "$*" >&2
}

die() {
  printf '[yeast-install] error: %s\n' "$*" >&2
  exit 1
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
  TARGET_HOME="$(getent passwd "${TARGET_USER}" | cut -d: -f6)"
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

pkg_update() {
  case "${PKG_MANAGER}" in
    apt)
      need_root apt-get update
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
      need_root apt-get install -y "$@"
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
  log "Installing base packages with ${PKG_MANAGER}"
  case "${PKG_MANAGER}" in
    apt)
      pkg_install ca-certificates curl git openssh-client golang-go build-essential
      ;;
    dnf|yum)
      pkg_install ca-certificates curl git openssh-clients golang gcc
      ;;
    pacman)
      pkg_install ca-certificates curl git openssh go base-devel
      ;;
    zypper)
      pkg_install ca-certificates curl git openssh go gcc
      ;;
    apk)
      pkg_install ca-certificates curl git openssh-client go build-base bash
      ;;
  esac
}

install_virtualization_packages() {
  log "Installing QEMU, image tooling, and ISO tooling"
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

ensure_required_tools() {
  local tool
  for tool in git go qemu-system-x86_64 qemu-img genisoimage ssh-keygen; do
    if ! command -v "${tool}" >/dev/null 2>&1; then
      die "required tool ${tool} is still missing after installation"
    fi
  done
}

clone_and_build() {
  WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/yeast-install.XXXXXX")"
  trap 'rm -rf "${WORKDIR}"' EXIT

  log "Cloning ${YEAST_REPO_URL} (${YEAST_REF})"
  git clone --depth 1 --branch "${YEAST_REF}" "${YEAST_REPO_URL}" "${WORKDIR}/src" || \
    die "failed to clone ${YEAST_REPO_URL}; if the repo is private, authenticate first or set YEAST_REPO_URL to an accessible source"

  log "Building Yeast"
  (
    cd "${WORKDIR}/src"
    go build -o "${WORKDIR}/yeast" ./cmd/yeast
  )
}

install_binary() {
  log "Installing binary to ${YEAST_BIN_PATH}"
  need_root install -d "${YEAST_INSTALL_DIR}"
  need_root install -m 0755 "${WORKDIR}/yeast" "${YEAST_BIN_PATH}"
}

ensure_user_paths() {
  log "Creating Yeast directories in ${TARGET_HOME}"
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast"
  need_root install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.yeast/cache"
}

ensure_ssh_key() {
  if [[ -f "${TARGET_HOME}/.ssh/id_ed25519.pub" || -f "${TARGET_HOME}/.ssh/id_rsa.pub" ]]; then
    log "SSH public key already exists for ${TARGET_USER}"
    return
  fi

  log "Generating an SSH key for ${TARGET_USER}"
  need_root install -d -m 0700 -o "${TARGET_USER}" -g "${TARGET_GROUP}" "${TARGET_HOME}/.ssh"
  run_as_target ssh-keygen -t ed25519 -N "" -C "yeast@$(hostname)" -f "${TARGET_HOME}/.ssh/id_ed25519"
}

ensure_kvm_group_membership() {
  if ! getent group kvm >/dev/null 2>&1; then
    warn "kvm group was not found; you may need to install host virtualization packages or configure KVM manually"
    return
  fi

  if id -nG "${TARGET_USER}" | tr ' ' '\n' | grep -qx kvm; then
    log "${TARGET_USER} is already in the kvm group"
    return
  fi

  log "Adding ${TARGET_USER} to the kvm group"
  need_root usermod -aG kvm "${TARGET_USER}"
  KVM_GROUP_UPDATED=1
}

run_post_install_checks() {
  log "Running yeast doctor"
  if ! run_as_target "${YEAST_BIN_PATH}" doctor; then
    warn "yeast doctor reported warnings or blockers"
  fi
}

print_summary() {
  log "Installation complete"
  printf '\n'
  printf 'Installed binary: %s\n' "${YEAST_BIN_PATH}"
  printf 'Repo source: %s (ref: %s)\n' "${YEAST_REPO_URL}" "${YEAST_REF}"
  printf '\n'
  printf 'Next steps:\n'
  printf '  1. yeast pull --list\n'
  printf '  2. mkdir my-project && cd my-project\n'
  printf '  3. yeast init --name web --image ubuntu-22.04 --memory 2048 --cpus 2\n'
  printf '  4. yeast pull ubuntu-22.04\n'
  printf '  5. yeast up\n'
  printf '\n'
  if [[ "${KVM_GROUP_UPDATED:-0}" -eq 1 ]]; then
    warn "${TARGET_USER} was added to the kvm group. Log out and back in before running Yeast so the new group membership is active."
  fi
}

main() {
  [[ "$(uname -s)" == "Linux" ]] || die "Yeast installer currently supports Linux only"
  detect_target_user
  detect_package_manager
  pkg_update
  install_base_packages
  install_virtualization_packages
  ensure_required_tools
  clone_and_build
  install_binary
  ensure_user_paths
  ensure_ssh_key
  ensure_kvm_group_membership
  run_post_install_checks
  print_summary
}

main "$@"
