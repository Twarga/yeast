# Yeast v0.2.0 Manual Test Tutorial

This is the real host manual test for the current `v0.2.0` candidate.

It is written for your current situation:

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
./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

That script will:

- run the happy-path lifecycle test
- run a negative-path contract test suite
- assert JSON error codes for common v0.2.0 failure cases

The rest of this document is the same flow, but broken into individual manual steps.

### Smoke Script Modes

The script supports three modes through `TEST_MODE`:

```fish
TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=positive ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

- `full`: happy path plus negative-path suite
- `positive`: only the real VM lifecycle
- `negative`: only error-path contract checks, no VM boot

## 0. What This Test Proves

This test proves that the current `v0.2.0` candidate can:

- run on a real Linux host
- detect host requirements with `doctor`
- initialize a project
- pull a trusted Ubuntu image
- start a real QEMU/KVM VM
- wait for SSH readiness
- report status correctly
- SSH into the guest
- honor explicit `hostname`
- honor explicit `ssh_port`
- stop and restart the VM cleanly
- destroy runtime state cleanly
- classify common invalid-input and bad-state failures with stable JSON error codes

This does not test:

- provisioning
- snapshots
- restore
- multi-VM networking
- guest exec/copy/logs
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

For this full test, use the new built binary only.

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
v0.2.0-test
```

If you do not see `v0.2.0-test`, stop and check which binary you are running.

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

Run:

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

Use a clean folder:

```fish
mkdir -p /tmp/yeast-v020-test
cd /tmp/yeast-v020-test
rm -rf .yeast yeast.yaml
```

## 6. Initialize The Project

Run:

```fish
$BIN init
```

Expected:

- `yeast.yaml` created
- `.yeast/project.json` created

Check:

```fish
ls -la
cat yeast.yaml
```

## 7. Replace The Config With A Real v0.2.0 Test Case

Write this exact config:

```fish
printf '%s\n' \
'version: 1' \
'instances:' \
'  - name: web' \
'    hostname: web-lab' \
'    image: ubuntu-24.04' \
'    memory: 1024' \
'    cpus: 1' \
'    disk_size: 20G' \
'    ssh_port: 2205' \
'    user: yeast' \
'    sudo: none' > yeast.yaml
```

Verify it:

```fish
cat yeast.yaml
```

This config specifically tests:

- `disk_size`
- `hostname`
- `ssh_port`

## 8. List Supported Images

Run:

```fish
$BIN pull --list
```

Expected:

- `ubuntu-22.04`
- `ubuntu-24.04`

## 9. Pull The Ubuntu Image

Run:

```fish
$BIN pull ubuntu-24.04
```

Expected:

- image is downloaded or confirmed from cache
- no checksum failure
- command completes successfully

## 10. Start The VM

Run:

```fish
$BIN up
```

Expected:

- VM starts successfully
- Yeast prints something like:

```text
Started web (127.0.0.1:2205)
```

Important checks here:

- the host SSH port must be `2205`
- startup must not silently choose `2222`

## 11. Check Status

Run:

```fish
$BIN status
```

Expected:

- instance `web`
- status `running`
- host address `127.0.0.1:2205`

Then run JSON mode:

```fish
$BIN status --json
```

Expected:

- valid JSON
- `SSHPort` or equivalent host-side port is `2205`
- no terminal formatting noise

## 12. SSH Into The VM

Run:

```fish
$BIN ssh web
```

Inside the VM, run:

```bash
hostname
whoami
```

Expected:

- `hostname` returns `web-lab`
- `whoami` returns `yeast`

This is the most important `v0.2.0` check.

If hostname is still `web`, then the new hostname feature is not working on a real guest.

## 13. Exit The Guest

Inside the guest:

```bash
exit
```

## 14. Stop The VM

Run:

```fish
$BIN down
```

Then:

```fish
$BIN status
```

Expected:

- instance still exists
- status is stopped
- no stale running PID/port state

## 15. Start It Again

Run:

```fish
$BIN up
```

Then:

```fish
$BIN status
```

Expected:

- starts successfully again
- still uses `2205`
- no unexpected port drift

## 16. SSH Again And Recheck

Run:

```fish
$BIN ssh web
```

Inside the guest:

```bash
hostname
whoami
exit
```

Expected:

- hostname still `web-lab`
- user still `yeast`

## 17. Destroy The Project

Run:

```fish
$BIN destroy
```

Then:

```fish
$BIN status
```

Expected:

- no running instances
- project runtime state cleaned up

## 18. Optional Cache Check

Destroy should not remove the shared image cache.

Check:

```fish
ls -la ~/.yeast/cache/images/ubuntu-24.04
```

Expected:

- image cache still exists

## 19. Pass Criteria

Call this manual test a pass only if all of these are true:

- `doctor` shows no blocker
- `pull ubuntu-24.04` works
- `up` works on a real VM
- reported SSH port is `2205`
- `ssh web` works
- guest `hostname` is `web-lab`
- guest user is `yeast`
- `down` works
- restart works
- `destroy` works
- `status --json` works cleanly

## 20. Failure Notes Template

If anything fails, capture:

- command you ran
- exact output
- whether failure is before boot, during boot, during SSH, or after restart

Use this format:

```text
Command:
$BIN up

Observed:
<paste output>

Expected:
Started web (127.0.0.1:2205)

Notes:
<anything unusual>
```

## 21. Final Release Decision

If this full manual test passes on your laptop, then `v0.2.0` is in good shape to release.

If it fails on:

- boot
- SSH
- hostname
- ssh_port
- restart

then do not release yet. Fix the failure first.
