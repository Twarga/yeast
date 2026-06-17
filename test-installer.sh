#!/usr/bin/env bash
set -euo pipefail

echo "==> Testing install.sh"

pass() {
    printf '✅ %s\n' "$1"
}

fail() {
    printf '❌ %s\n' "$1" >&2
    exit 1
}

echo "Testing script syntax..."
bash -n install.sh || fail "Script has syntax errors"
pass "Script syntax is valid"

echo "Testing source guard..."
bash -c 'source install.sh >/dev/null 2>&1' || fail "install.sh should be safe to source"
pass "install.sh is safe to source"

echo "Testing piped execution guard..."
bash -lc '
    awk '\''{
        if ($0 == "  main \"$@\"") {
            print "  printf \"MAIN_RAN\\n\""
            next
        }
        print
    }'\'' install.sh | bash
' | grep -qx "MAIN_RAN" || fail "install.sh piped execution guard still breaks or does not enter main"
pass "Piped execution guard works"

echo "Testing distro detection..."
bash -c 'source install.sh && detect_distro && [[ -n "${PKG_MANAGER}" ]]' || fail "detect_distro did not set PKG_MANAGER"
pass "Distro detection works"

echo "Testing release artifact naming..."
bash -c 'source install.sh && YEAST_ARCH=amd64 && compute_release_artifact && [[ "${RELEASE_ARTIFACT}" == "yeast_linux_amd64.tar.gz" ]]' \
    || fail "compute_release_artifact produced the wrong amd64 artifact name"
pass "Release artifact naming works"

echo "Testing legacy release archive extraction..."
bash -c '
    source install.sh
    tmp_dir="$(mktemp -d)"
    export WORKDIR="${tmp_dir}"
    export YEAST_ARCH="amd64"
    export RELEASE_ARTIFACT="yeast_linux_amd64.tar.gz"
    export YEAST_INSTALL_DIR="${tmp_dir}/bin"
    export YEAST_BIN_PATH="${YEAST_INSTALL_DIR}/yeast"
    need_root() {
        "$@"
    }
    mkdir -p "${tmp_dir}/package"
    printf "fake-binary" > "${tmp_dir}/package/yeast_linux_amd64"
    tar -czf "${tmp_dir}/${RELEASE_ARTIFACT}" -C "${tmp_dir}/package" yeast_linux_amd64
    extract_and_install_binary
    [[ -f "${YEAST_BIN_PATH}" ]]
' || fail "extract_and_install_binary did not tolerate the legacy release binary name"
pass "Legacy release archive extraction works"

echo "Testing post-install doctor fix flow..."
bash -c '
    source install.sh
    stub_dir="$(mktemp -d)"
    log_file="${stub_dir}/doctor.log"
    cat > "${stub_dir}/yeast" <<EOF
#!/usr/bin/env bash
printf "%s\n" "\$*" >> "${log_file}"
EOF
    chmod +x "${stub_dir}/yeast"
    YEAST_BIN_PATH="${stub_dir}/yeast"
    run_doctor_fixups
    grep -qx "doctor --fix --yes" "${log_file}"
' || fail "run_doctor_fixups did not invoke 'yeast doctor --fix --yes'"
pass "Doctor fix flow runs yeast doctor --fix --yes"

echo "Testing SSH key bootstrap helper..."
bash -c '
    source install.sh
    tmp_home="$(mktemp -d)"
    export TARGET_HOME="${tmp_home}"
    export TARGET_USER="$(id -un)"
    export TARGET_GROUP="$(id -gn)"
    need_root() {
        "$@"
    }
    ssh-keygen() {
        local out=""
        while (($#)); do
            if [[ "$1" == "-f" ]]; then
                out="$2"
                shift 2
                continue
            fi
            shift
        done
        mkdir -p "$(dirname "${out}")"
        : > "${out}"
        : > "${out}.pub"
    }
    ensure_user_ssh_key
    [[ -f "${tmp_home}/.ssh/id_ed25519" ]]
    [[ -f "${tmp_home}/.ssh/id_ed25519.pub" ]]
' || fail "ensure_user_ssh_key did not create a default SSH key pair"
pass "SSH key bootstrap helper works"

echo
echo "==> All installer tests passed"
