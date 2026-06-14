# Yeast v1.1.0 Release Smoke Test

Run this on a fresh Linux/KVM host before calling `v1.1.0` shippable.

```bash
export YEAST_SMOKE_VERSION="v1.1.0"
export YEAST_SMOKE_ROOT="$HOME/yeast-${YEAST_SMOKE_VERSION}-smoke"
mkdir -p "$YEAST_SMOKE_ROOT"
cd "$YEAST_SMOKE_ROOT"
```

Host checks:

```bash
uname -m
ls -l /dev/kvm || true
id
```

Artifact check:

```bash
mkdir -p artifact-test
cd artifact-test
curl -fLO "https://github.com/Twarga/yeast/releases/download/${YEAST_SMOKE_VERSION}/yeast_linux_amd64.tar.gz"
curl -fLO "https://github.com/Twarga/yeast/releases/download/${YEAST_SMOKE_VERSION}/SHA256SUMS.txt"
grep "yeast_linux_amd64.tar.gz" SHA256SUMS.txt | sha256sum -c -
tar -tzf yeast_linux_amd64.tar.gz
tar -xzf yeast_linux_amd64.tar.gz
./yeast version
sudo install -m 0755 yeast /usr/local/bin/yeast
hash -r
yeast version
cd "$YEAST_SMOKE_ROOT"
```

Installer check:

```bash
mkdir -p installer-test
cd installer-test
curl -fsSL "https://raw.githubusercontent.com/Twarga/yeast/${YEAST_SMOKE_VERSION}/install.sh" -o install.sh
bash -n install.sh
YEAST_REF="${YEAST_SMOKE_VERSION}" YEAST_EXPECTED_VERSION="${YEAST_SMOKE_VERSION}" bash install.sh
hash -r
yeast version
yeast doctor
cd "$YEAST_SMOKE_ROOT"
```

Updater check:

```bash
mkdir -p update-test
cd update-test
git clone --depth 1 --branch "${YEAST_SMOKE_VERSION}" https://github.com/Twarga/yeast.git source
cd source
GO_BIN="$(command -v go || true)"
if [ -z "$GO_BIN" ] && [ -x /usr/local/lib/yeast/go/go1.25.0/bin/go ]; then
  GO_BIN="/usr/local/lib/yeast/go/go1.25.0/bin/go"
fi
"$GO_BIN" build -trimpath -ldflags "-s -w -X yeast/internal/app.Version=${YEAST_SMOKE_VERSION}-smoke-old" -o /tmp/yeast-smoke-old ./cmd/yeast
sudo install -m 0755 /tmp/yeast-smoke-old /usr/local/bin/yeast
hash -r
yeast version
sudo yeast update --force --version "${YEAST_SMOKE_VERSION}"
hash -r
yeast version
yeast update --check --version "${YEAST_SMOKE_VERSION}"
cd "$YEAST_SMOKE_ROOT"
```

Single-VM check:

```bash
mkdir -p vm-basic
cd vm-basic
yeast init --template ubuntu-basic
yeast pull ubuntu-24.04
yeast up
yeast status
yeast inspect web
yeast exec web -- whoami
yeast exec web -- hostname
printf 'yeast-smoke\n' > artifact.txt
yeast copy web --to-guest ./artifact.txt /home/yeast/artifact.txt
yeast copy web --from-guest /home/yeast/artifact.txt ./artifact-out.txt
diff -u artifact.txt artifact-out.txt
yeast down
yeast snapshot web baseline --description "v1.1.0 smoke baseline"
yeast up
yeast exec web -- touch /home/yeast/broken-marker
yeast down
yeast restore web baseline
yeast up
yeast exec web -- test ! -e /home/yeast/broken-marker
yeast down
yeast clean
yeast destroy
cd "$YEAST_SMOKE_ROOT"
```

Two-VM networking check:

```bash
mkdir -p vm-network
cd vm-network
yeast init --template two-vm-lab
yeast up
yeast status
yeast exec attacker -- ping -c 2 10.10.10.20
yeast exec target -- ping -c 2 10.10.10.10
yeast status --json
yeast inspect attacker --json
yeast down
yeast clean
yeast destroy
```

Pass criteria:

- artifact download and checksum pass
- installed version is `v1.1.0`
- update replaces the smoke-old binary
- `yeast doctor` has no blockers
- Ubuntu image pulls and verifies
- single VM boots, accepts guest control, snapshots, restores, and cleans up
- two VMs can ping over the private lab network
