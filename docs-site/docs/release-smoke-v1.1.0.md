---
title: Release Smoke Test v1.1.0
description: Fresh Linux/KVM validation checklist for Yeast v1.1.0
---

# Release Smoke Test v1.1.0

Run this on a fresh Linux machine with KVM support before calling `v1.1.0` shippable.

Use a disposable machine if possible. The test installs Yeast into `/usr/local/bin`, downloads a cloud image, boots real QEMU/KVM VMs, and removes the test projects at the end.

## Pass Criteria

The release is shippable when all of these pass:

- the GitHub release tarball downloads
- `SHA256SUMS.txt` verifies the tarball
- the tarball contains a binary named `yeast`
- manual artifact install prints `v1.1.0`
- install script install prints `v1.1.0`
- `yeast update --force --version v1.1.0` replaces an older-looking binary
- `yeast doctor` has no blockers
- `yeast pull ubuntu-24.04` verifies the image
- single-VM lifecycle works
- guest control works: `exec`, `copy`, `logs`, `inspect`
- stopped-VM snapshot and restore work
- two-VM private networking works
- `yeast down`, `yeast clean`, and `yeast destroy` finish cleanly

## 1. Prepare the Host

Set the version and workspace:

```bash
export YEAST_SMOKE_VERSION="v1.1.0"
export YEAST_SMOKE_ROOT="$HOME/yeast-${YEAST_SMOKE_VERSION}-smoke"
mkdir -p "$YEAST_SMOKE_ROOT"
cd "$YEAST_SMOKE_ROOT"
```

Check the host:

```bash
uname -a
uname -m
id
ls -l /dev/kvm || true
```

Expected:

- architecture is `x86_64`
- `/dev/kvm` exists on a native KVM host
- your user is in the `kvm` group, or you can log out and back in after adding it

Install host packages if needed:

```bash
# Ubuntu / Debian
sudo apt update
sudo apt install -y curl git tar gzip openssh-client qemu-kvm qemu-utils genisoimage

# Fedora / RHEL
sudo dnf install -y curl git tar gzip openssh-clients qemu-kvm qemu-img genisoimage

# Arch Linux
sudo pacman -S --needed curl git tar gzip openssh qemu-full cdrtools
```

If your user cannot access KVM:

```bash
sudo usermod -aG kvm "$USER"
```

Then log out and back in before continuing.

## 2. Verify the Release Artifact

Download the release files:

```bash
mkdir -p artifact-test
cd artifact-test

curl -fLO "https://github.com/Twarga/yeast/releases/download/${YEAST_SMOKE_VERSION}/yeast_linux_amd64.tar.gz"
curl -fLO "https://github.com/Twarga/yeast/releases/download/${YEAST_SMOKE_VERSION}/SHA256SUMS.txt"
```

Verify checksum:

```bash
grep "yeast_linux_amd64.tar.gz" SHA256SUMS.txt | sha256sum -c -
```

Expected:

```text
yeast_linux_amd64.tar.gz: OK
```

Verify archive layout:

```bash
tar -tzf yeast_linux_amd64.tar.gz
```

Expected:

```text
yeast
```

Install the artifact manually:

```bash
tar -xzf yeast_linux_amd64.tar.gz
./yeast version
sudo install -m 0755 yeast /usr/local/bin/yeast
hash -r
yeast version
```

Expected:

```text
v1.1.0
```

Return to the smoke root:

```bash
cd "$YEAST_SMOKE_ROOT"
```

## 3. Verify the Install Script

Download the installer from the release tag:

```bash
mkdir -p installer-test
cd installer-test

curl -fsSL "https://raw.githubusercontent.com/Twarga/yeast/${YEAST_SMOKE_VERSION}/install.sh" -o install.sh
bash -n install.sh
```

Run the installer against the release tag:

```bash
YEAST_REF="${YEAST_SMOKE_VERSION}" \
YEAST_EXPECTED_VERSION="${YEAST_SMOKE_VERSION}" \
bash install.sh
```

Verify:

```bash
hash -r
yeast version
yeast doctor
```

Expected:

- `yeast version` prints `v1.1.0`
- `yeast doctor` shows no blockers

Return to the smoke root:

```bash
cd "$YEAST_SMOKE_ROOT"
```

## 4. Verify `yeast update`

Build a current-code binary that pretends to be older. This tests the updater without relying on an old release's updater behavior.

```bash
mkdir -p update-test
cd update-test

git clone --depth 1 --branch "${YEAST_SMOKE_VERSION}" https://github.com/Twarga/yeast.git source
cd source

GO_BIN="$(command -v go || true)"
if [ -z "$GO_BIN" ] && [ -x /usr/local/lib/yeast/go/go1.25.0/bin/go ]; then
  GO_BIN="/usr/local/lib/yeast/go/go1.25.0/bin/go"
fi

"$GO_BIN" build -trimpath \
  -ldflags "-s -w -X yeast/internal/app.Version=${YEAST_SMOKE_VERSION}-smoke-old" \
  -o /tmp/yeast-smoke-old ./cmd/yeast

sudo install -m 0755 /tmp/yeast-smoke-old /usr/local/bin/yeast
hash -r
yeast version
```

Expected:

```text
v1.1.0-smoke-old
```

Run update:

```bash
sudo yeast update --force --version "${YEAST_SMOKE_VERSION}"
hash -r
yeast version
```

Expected:

```text
v1.1.0
```

Check-only mode should also work:

```bash
yeast update --check --version "${YEAST_SMOKE_VERSION}"
```

Return to the smoke root:

```bash
cd "$YEAST_SMOKE_ROOT"
```

## 5. Single-VM Lifecycle Smoke

Create a clean project:

```bash
mkdir -p vm-basic
cd vm-basic
yeast init --template ubuntu-basic
```

Pull the image:

```bash
yeast pull ubuntu-24.04
```

Start the VM:

```bash
yeast up
yeast status
yeast inspect web
```

Expected:

- `web` is `running`
- SSH address is shown
- `inspect` shows a runtime directory and provisioning status

Run guest commands:

```bash
yeast exec web -- whoami
yeast exec web -- hostname
yeast exec web -- cloud-init status --wait
```

Expected:

- `whoami` prints `yeast`
- `cloud-init status --wait` exits successfully

Test copy:

```bash
printf 'yeast-smoke\n' > artifact.txt
yeast copy web --to-guest ./artifact.txt /home/yeast/artifact.txt
yeast copy web --from-guest /home/yeast/artifact.txt ./artifact-out.txt
diff -u artifact.txt artifact-out.txt
```

Test logs:

```bash
yeast logs web | head -n 40
```

Stop and restart:

```bash
yeast down
yeast up
yeast exec web -- echo restarted-ok
```

Snapshot and restore:

```bash
yeast down
yeast snapshot web baseline --description "v1.1.0 smoke baseline"
yeast snapshots web
yeast up
yeast exec web -- touch /home/yeast/broken-marker
yeast down
yeast restore web baseline
yeast up
yeast exec web -- test ! -e /home/yeast/broken-marker
```

Clean up:

```bash
yeast down
yeast clean
yeast destroy
```

Return to the smoke root:

```bash
cd "$YEAST_SMOKE_ROOT"
```

## 6. Two-VM Networking Smoke

Create a two-VM project:

```bash
mkdir -p vm-network
cd vm-network
yeast init --template two-vm-lab
```

Start both VMs:

```bash
yeast up
yeast status
```

Expected:

- `attacker` is running
- `target` is running
- both have lab IPs

Verify private network from both sides:

```bash
yeast exec attacker -- ping -c 2 10.10.10.20
yeast exec target -- ping -c 2 10.10.10.10
```

Verify JSON output:

```bash
yeast status --json
yeast inspect attacker --json
```

Clean up:

```bash
yeast down
yeast clean
yeast destroy
```

## 7. Final Report

Create a short local report:

```bash
cat > "$YEAST_SMOKE_ROOT/RESULT.md" <<EOF
# Yeast ${YEAST_SMOKE_VERSION} Smoke Result

Host: $(hostname)
Date: $(date -Is)
Kernel: $(uname -r)
Arch: $(uname -m)

- Artifact download: PASS / FAIL
- Artifact checksum: PASS / FAIL
- Manual artifact install: PASS / FAIL
- Install script: PASS / FAIL
- Update command: PASS / FAIL
- Doctor: PASS / FAIL
- Image pull: PASS / FAIL
- Single VM lifecycle: PASS / FAIL
- Guest control: PASS / FAIL
- Snapshot restore: PASS / FAIL
- Two-VM networking: PASS / FAIL
- Cleanup: PASS / FAIL

Notes:
- 
EOF

cat "$YEAST_SMOKE_ROOT/RESULT.md"
```

If every line is `PASS`, `v1.1.0` is shippable from the fresh-machine point of view.

## Failure Notes

If artifact download fails, check that the GitHub release exists and includes:

- `yeast_linux_amd64.tar.gz`
- `SHA256SUMS.txt`

If update fails during extraction, inspect the archive:

```bash
tar -tzf yeast_linux_amd64.tar.gz
```

The archive must contain `yeast`.

If VM boot fails, collect:

```bash
yeast doctor
yeast status --json
yeast logs web
```

If cleanup leaves processes behind:

```bash
ps -ef | grep qemu
yeast clean
```
