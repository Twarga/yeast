# Yeast v0.1.0 Manual Test Tutorial

This is the client-side test for Yeast v0.1.0.

Follow it from a normal Linux machine as if you are a new user installing Yeast from the landing page.

The goal is to prove the v0.1.0 loop:

```text
install -> doctor -> init -> pull -> up -> status -> ssh -> down -> up again -> destroy
```

## 0. What This Test Proves

This manual test proves that Yeast can:

- install on a real Linux host
- detect host requirements
- create a project config
- pull a trusted Ubuntu image
- start a QEMU/KVM VM
- wait for SSH
- show accurate status
- open an SSH session
- stop the VM
- start it again
- destroy project runtime files
- keep the shared image cache

This does not test future features like provisioning, snapshots, private networking, templates, LabsBackery, Yeast MCP, or Twarga Cloud.

## 1. Host Requirements

Use a Linux machine with virtualization enabled.

Required:

- Linux host
- CPU virtualization enabled in BIOS/UEFI
- `/dev/kvm` available
- internet access
- permission to install packages with `sudo`
- enough disk space for Ubuntu cloud image and VM disk

Recommended:

- 4 GB RAM minimum
- 10 GB free disk
- stable internet connection

Check KVM:

```bash
ls -l /dev/kvm
```

If `/dev/kvm` does not exist, enable virtualization in BIOS/UEFI or install your distro's KVM/QEMU packages.

## 2. Install Yeast From The Landing Page Command

Copy this exact command:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/v0.1.0/install.sh | YEAST_REF=v0.1.0 bash
```

Why `YEAST_REF=v0.1.0` is included:

- the script is downloaded from the `v0.1.0` tag
- the source build also checks out the `v0.1.0` tag
- this avoids accidentally testing a newer branch

Expected result:

- installer detects your package manager
- installer installs or verifies QEMU/KVM, SSH, Git, Go, and ISO tooling
- installer creates Yeast directories
- installer builds and installs `yeast`
- installer runs `yeast doctor`

If the installer says you were added to a KVM group, log out and log back in before continuing. A reboot is also fine.

## 3. Verify The Installed Binary

Run:

```bash
which yeast
yeast version
```

Expected:

```text
v0.1.0
```

If the version is not `v0.1.0`, stop and write down the output.

## 4. Run Doctor

Run:

```bash
yeast doctor
```

Expected:

- QEMU check passes
- `qemu-img` check passes
- ISO tool check passes
- SSH check passes
- SSH public key check passes
- KVM check passes or gives clear instructions

If KVM permission fails:

```bash
groups
ls -l /dev/kvm
```

If your user was just added to a group, log out and log back in.

## 5. Create A Clean Test Project

Use a new empty folder:

```bash
mkdir -p ~/yeast-v010-test
cd ~/yeast-v010-test
```

Make sure it is empty:

```bash
ls -la
```

## 6. Initialize The Project

Run:

```bash
yeast init
```

Expected:

- `yeast.yaml` is created
- default instance is named `web`
- default image is `ubuntu-24.04`

Inspect the config:

```bash
cat yeast.yaml
```

Expected shape:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
```

## 7. List Supported Images

Run:

```bash
yeast pull --list
```

Expected:

- `ubuntu-22.04`
- `ubuntu-24.04`

## 8. Pull Ubuntu 24.04

Run:

```bash
yeast pull ubuntu-24.04
```

Expected:

- image downloads if not already cached
- checksum verification succeeds
- image is stored in the Yeast cache

This can take time depending on your internet connection.

If the command fails, save the full output.

## 9. Start The VM

Run:

```bash
yeast up
```

Expected:

- Yeast creates project runtime files
- Yeast creates a VM disk
- Yeast creates cloud-init seed data
- Yeast starts QEMU/KVM
- Yeast waits for SSH
- Yeast reports the VM as ready or running

First boot can take a few minutes because Ubuntu cloud-init needs to finish.

## 10. Check Status

Run:

```bash
yeast status
```

Expected:

- instance `web` appears
- status is running
- SSH address or port is visible

Also test JSON output:

```bash
yeast status --json
```

Expected:

- valid JSON
- no styled terminal text
- instance `web` exists in the JSON result
- status is running

## 11. SSH Into The VM

Run:

```bash
yeast ssh web
```

Inside the VM, run:

```bash
hostname
whoami
uname -a
ip addr
exit
```

Expected:

- SSH opens without manually typing a port
- hostname is related to `web`
- user is the configured cloud-init user
- `exit` returns to your host shell

## 12. Stop The VM

Run:

```bash
yeast down
```

Expected:

- VM stops
- disk is not deleted
- image cache is not deleted

Check status:

```bash
yeast status
```

Expected:

- `web` is stopped or not running

## 13. Start The VM Again

Run:

```bash
yeast up
```

Expected:

- Yeast reuses the existing project disk
- VM starts again
- SSH becomes reachable again

Check:

```bash
yeast status
yeast ssh web
```

Then exit SSH:

```bash
exit
```

## 14. Destroy The Project Runtime

Run:

```bash
yeast destroy
```

Expected:

- project runtime files are removed
- VM process is stopped
- project disk is removed
- shared image cache remains
- `yeast.yaml` remains in your project folder

Check status:

```bash
yeast status
```

Expected:

- no running VM
- no stale running status

## 15. Verify Cache Was Not Destroyed

Check Yeast cache:

```bash
find ~/.yeast/cache -maxdepth 3 -type f | head
```

Expected:

- cached image files still exist

This matters because `destroy` should clean the project runtime, not delete shared base images.

## 16. Test Error Handling

Run this from the project folder:

```bash
yeast ssh missing-vm
```

Expected:

- Yeast gives a clear error
- it does not crash

Run:

```bash
yeast pull does-not-exist
```

Expected:

- Yeast gives a clear unsupported image error

## 17. Clean Up After The Test

If everything passed:

```bash
cd ~
rm -rf ~/yeast-v010-test
```

Do not remove `~/.yeast/cache` unless you want to delete downloaded base images.

Optional full cleanup:

```bash
rm -rf ~/.yeast
```

## 18. Pass Or Fail Report

When finished, write a result like this:

```text
Yeast v0.1.0 manual test

Host:
Distro:
Kernel:
CPU:

Install: pass/fail
Doctor: pass/fail
Init: pass/fail
Pull: pass/fail
Up: pass/fail
Status: pass/fail
Status JSON: pass/fail
SSH: pass/fail
Down: pass/fail
Up again: pass/fail
Destroy: pass/fail
Cache preserved: pass/fail
Error handling: pass/fail

Notes:
```

Useful host info:

```bash
cat /etc/os-release
uname -a
groups
yeast version
```

## 19. If Something Fails

Save:

- the exact command you ran
- the full terminal output
- your distro
- whether `/dev/kvm` exists
- whether your user is in the KVM/libvirt group

Useful debug commands:

```bash
yeast doctor
yeast status --json
ps aux | grep qemu
find ~/.yeast -maxdepth 4 -type f | sort
```

Do not delete the project folder before saving the output if you want to debug the failure.
