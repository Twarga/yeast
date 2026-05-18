# Yeast v0.3.0 Manual Test Tutorial

This is the real host manual test for the current `v0.3.0` candidate.

It assumes:

- you already have an older `yeast` installed in `/usr/local/bin`
- you are using `fish`
- you want to test the new binary without replacing the old one yet

This guide uses the built binary directly from the repo:

```fish
~/Projects/yeast/dist/yeast-linux-amd64
```

## Fast Path

If you want the full loop in one command, use the smoke-test script from the repo root:

```fish
cd ~/Projects/yeast
TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

That script now proves:

- the lifecycle path
- `disk_size`, `hostname`, and `ssh_port`
- provisioning during `yeast up`
- `yeast provision` reruns
- Caddy serving content inside the guest
- negative-path JSON contracts for key config and state failures

### Smoke Script Modes

```fish
TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=positive ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

- `full`: lifecycle + provisioning + negative-path suite
- `positive`: only the real VM and provisioning path
- `negative`: only error-path contract checks, no VM boot

The rest of this document is the same flow, but broken into explicit manual steps.

## 0. What This Test Proves

This test proves that the current `v0.3.0` candidate can:

- run on a real Linux host
- detect host requirements with `doctor`
- initialize a project
- pull a trusted Ubuntu image
- start a real QEMU/KVM VM
- wait for SSH readiness
- apply post-boot provisioning automatically
- report `provisioned` status in JSON
- SSH into the guest
- honor explicit `hostname`
- honor explicit `ssh_port`
- install and run Caddy inside the guest
- rerun provisioning with `yeast provision`
- stop and restart the VM cleanly
- destroy runtime state cleanly
- classify common invalid-input and bad-state failures with stable JSON error codes

This does not test:

- snapshots
- restore
- private networking
- guest exec/copy/logs beyond provisioning
- templates
- LabsBackery
- Yeast MCP
- Twarga Cloud
- installer upgrade behavior across every Linux distro
- every internal helper-failure branch that only unit tests can force

## 1. Important Rule For This Test

Do not run plain `yeast`.

Your shell currently resolves:

```fish
which yeast
```

to:

```text
/usr/local/bin/yeast
```

That is the old installed binary.

For this test, use the new built binary only.

## 2. Set The Binary Path In Fish

From the repo root:

```fish
cd ~/Projects/yeast
set BIN ./dist/yeast-linux-amd64
```

Confirm it:

```fish
$BIN version
```

Expected:

```text
v0.3.0-test
```

If you do not see `v0.3.0-test`, stop and check which binary you are running.

## 3. Host Requirements

You need:

- Linux
- `/dev/kvm`
- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage` or `mkisofs`
- `ssh`
- a valid public key in `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`

Optional but recommended:

- at least 4 GB RAM free
- at least 10 GB disk free
- stable internet for first image pull

## 4. Run Doctor

```fish
$BIN doctor
```

Expected:

- `qemu-system-x86_64` ok
- `qemu-img` ok
- `iso-builder` ok
- `ssh` ok
- `/dev/kvm` ok
- `ssh-public-key` ok

If any blocker appears, fix that before going further.

## 5. Create A Fresh Test Project

```fish
mkdir -p /tmp/yeast-v030-test
cd /tmp/yeast-v030-test
rm -rf .yeast yeast.yaml site
mkdir -p site
```

## 6. Initialize The Project

```fish
$BIN init
```

Expected:

- `yeast.yaml` created
- `.yeast/project.json` created

## 7. Write The Caddy Example Files

Create the site file:

```fish
printf '%s\n' \
'<!doctype html>' \
'<html lang="en">' \
'  <body>' \
'    <h1>Yeast v0.3 provisioning works.</h1>' \
'  </body>' \
'</html>' > site/index.html
```

Create the Caddyfile:

```fish
printf '%s\n' \
':80 {' \
'  root * /var/www/html' \
'  file_server' \
'}' > site/Caddyfile
```

## 8. Replace The Config With A Real v0.3.0 Test Case

```fish
printf '%s\n' \
'version: 1' \
'provision:' \
'  packages:' \
'    - caddy' \
'  files:' \
'    - source: ./site/index.html' \
'      destination: /var/www/html/index.html' \
'      permissions: "0644"' \
'    - source: ./site/Caddyfile' \
'      destination: /etc/caddy/Caddyfile' \
'      permissions: "0644"' \
'  shell:' \
'    - sudo systemctl enable caddy' \
'    - sudo systemctl restart caddy' \
'instances:' \
'  - name: web' \
'    hostname: caddy-lab' \
'    image: ubuntu-24.04' \
'    memory: 1024' \
'    cpus: 1' \
'    disk_size: 20G' \
'    ssh_port: 2205' \
'    user: yeast' \
'    sudo: nopasswd' > yeast.yaml
```

Verify it:

```fish
cat yeast.yaml
```

This config specifically tests:

- `disk_size`
- `hostname`
- `ssh_port`
- `provision.packages`
- `provision.files`
- `provision.shell`

## 9. Pull The Ubuntu Image

```fish
$BIN pull ubuntu-24.04
```

Expected:

- image is downloaded or confirmed from cache
- no checksum failure

## 10. Start The VM

```fish
$BIN up
```

Expected:

- VM starts successfully
- provisioning runs automatically
- command completes without shell or package failure

## 11. Check Status

```fish
$BIN status
$BIN status --json
```

Expected:

- instance `web`
- status `running`
- host address `127.0.0.1:2205`
- JSON includes `ProvisioningStatus` set to `provisioned`

## 12. SSH Into The VM And Verify The Guest

```fish
$BIN ssh web
```

Inside the guest, run:

```bash
hostname
whoami
sudo systemctl is-active caddy
curl -fsS http://127.0.0.1
```

Expected:

- `hostname` returns `caddy-lab`
- `whoami` returns `yeast`
- `systemctl is-active caddy` returns `active`
- `curl` output contains `Yeast v0.3 provisioning works.`

Exit the guest:

```bash
exit
```

## 13. Rerun Provisioning

Edit the site content on the host:

```fish
printf '%s\n' \
'<!doctype html>' \
'<html lang="en">' \
'  <body>' \
'    <h1>Yeast reprovisioned content.</h1>' \
'  </body>' \
'</html>' > site/index.html
```

Rerun provisioning:

```fish
$BIN provision web
```

Then SSH in again:

```fish
$BIN ssh web
```

Inside the guest:

```bash
curl -fsS http://127.0.0.1
exit
```

Expected:

- curl output now contains `Yeast reprovisioned content.`

## 14. Stop The VM

```fish
$BIN down
$BIN status
```

Expected:

- instance still exists
- status is `stopped`

## 15. Start It Again

```fish
$BIN up
$BIN status
$BIN status --json
```

Expected:

- starts successfully again
- still uses `2205`
- JSON still shows `ProvisioningStatus` as `provisioned`

## 16. Recheck The Served Content After Restart

```fish
$BIN ssh web
```

Inside the guest:

```bash
curl -fsS http://127.0.0.1
exit
```

Expected:

- the page still contains `Yeast reprovisioned content.`

## 17. Destroy The Project

```fish
$BIN destroy
$BIN status --json
```

Expected:

- no tracked instances remain
- project runtime state is cleaned up

## 18. Negative Contract Checks

The full smoke script also verifies:

- uninitialized `status` -> `failed_precondition`
- repeated `init` -> `conflict`
- unsupported `pull` -> `invalid_argument`
- corrupt metadata -> `internal`
- state project mismatch -> `internal`
- missing config -> `failed_precondition`
- invalid `disk_size` -> `invalid_argument`
- invalid `hostname` -> `invalid_argument`
- invalid `ssh_port` -> `invalid_argument`
- duplicate `ssh_port` -> `invalid_argument`
- missing provision source file -> `invalid_argument`

Run that with:

```fish
TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

## 19. Pass Criteria

Call this manual test a pass only if all of these are true:

- `doctor` shows no blocker
- `pull ubuntu-24.04` works
- `up` works on a real VM
- provisioning completes during `up`
- reported SSH port is `2205`
- guest `hostname` is `caddy-lab`
- guest user is `yeast`
- Caddy is active
- guest HTTP content matches the provisioned file
- `yeast provision web` updates guest content
- `down` works
- restart works
- `destroy` works
- `status --json` reports `provisioned`

## 20. Failure Notes Template

If anything fails, capture:

- command you ran
- exact output
- whether failure is before boot, during boot, during provisioning, during SSH, or after restart

Use this format:

```text
Command:
$BIN up

Observed:
<paste output>

Expected:
VM starts and provisioning completes

Notes:
<anything unusual>
```

## 21. Final Release Decision

If this full manual test passes on your Linux/KVM host, then `v0.3.0` is in good shape to release.

If it fails on:

- boot
- provisioning
- SSH
- served content
- rerun provisioning
- restart

then do not release yet. Fix the failure first.
