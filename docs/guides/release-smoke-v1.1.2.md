# Release Smoke Test v1.1.2

This guide is for maintainers validating `v1.1.2` on a real Linux/KVM host.

Run it on a fresh machine when possible. The goal is to prove fresh install, update, artifact download, VM lifecycle, guest control, snapshots, private networking, and the daily update notice cache.

## Prepare

```bash
export YEAST_SMOKE_VERSION="v1.1.2"
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

## Artifact Check

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

The tarball must contain a binary named `yeast`.

## Installer Check

```bash
mkdir -p installer-test
cd installer-test
curl -fsSL "https://raw.githubusercontent.com/Twarga/yeast/${YEAST_SMOKE_VERSION}/install.sh" -o install.sh
bash -n install.sh
YEAST_VERSION="${YEAST_SMOKE_VERSION}" bash install.sh
hash -r
yeast version
yeast doctor
cd "$YEAST_SMOKE_ROOT"
```

## Updater Check

```bash
yeast update --check --version "${YEAST_SMOKE_VERSION}"
sudo yeast update --force --version "${YEAST_SMOKE_VERSION}"
hash -r
yeast version
```

## Single VM Check

```bash
mkdir -p vm-basic
cd vm-basic
yeast init --template ubuntu-basic
yeast up
test -f "$HOME/.yeast/cache/update-check.json"
yeast status
yeast inspect web
yeast exec web -- whoami
yeast exec web -- hostname
printf 'yeast-smoke\n' > artifact.txt
yeast copy web --to-guest ./artifact.txt /home/yeast/artifact.txt
yeast copy web --from-guest /home/yeast/artifact.txt ./artifact-out.txt
diff -u artifact.txt artifact-out.txt
yeast down
yeast snapshot web baseline --description "v1.1.2 smoke baseline"
yeast up
yeast exec web -- touch /home/yeast/broken-marker
yeast down
yeast restore web baseline
yeast up
yeast exec web -- test ! -e /home/yeast/broken-marker
yeast down
yeast destroy
cd "$YEAST_SMOKE_ROOT"
```

## Two VM Networking Check

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
yeast destroy
```

## Pass Criteria

- artifact download and checksum pass
- tarball extracts `yeast`
- installed version is `v1.1.2`
- update path completes
- `yeast doctor` has no blockers
- `yeast up` creates the daily update-check cache
- Ubuntu image pulls and verifies
- single VM boots and accepts guest control
- copy to/from guest works
- snapshot/restore removes the marker file
- two VMs can ping over the private lab network

The embedded terminal version is available with:

```bash
yeast docs release-smoke
```
