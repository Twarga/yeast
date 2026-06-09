---
title: Troubleshooting
description: Common issues, error messages, and how to fix them
---

# Troubleshooting

This page helps you diagnose and fix common issues with Yeast.

## Quick Diagnostic

Always start with `yeast doctor`:

```bash
yeast doctor
```

This checks:
- KVM support
- Required packages (QEMU, genisoimage, SSH)
- SSH key availability
- System resources

## System Check Issues

### KVM Not Available

**Symptom:** `yeast doctor` reports KVM not available

**Symptoms:**
- `/dev/kvm` does not exist
- `kvm` module not loaded
- User not in `kvm` group

**Fix:**

```bash
# Load KVM modules
sudo modprobe kvm
sudo modprobe kvm_intel  # or kvm_amd

# Add user to kvm group
sudo usermod -aG kvm $USER

# Log out and back in (or reboot)
```

**For VMs to work without KVM:**

Yeast will fall back to TCG (software emulation), which is much slower but functional. To force this mode for testing:

```bash
# Not recommended for production use
# VMs will be very slow
```

### QEMU Not Installed

**Symptom:** `yeast doctor` reports QEMU not found

**Fix:**

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y qemu-system-x86 qemu-utils

# Fedora/RHEL
sudo dnf install -y qemu-system-x86 qemu-img

# Arch Linux
sudo pacman -S qemu-base
```

### genisoimage Not Found

**Symptom:** `yeast doctor` reports missing `genisoimage`

**Fix:**

```bash
# Ubuntu/Debian
sudo apt install -y genisoimage

# Fedora/RHEL
sudo dnf install -y genisoimage

# Arch Linux
sudo pacman -S cdrtools
```

### SSH Key Missing

**Symptom:** `yeast doctor` reports SSH key not found

**Fix:**

```bash
# Generate a new SSH key
ssh-keygen -t ed25519 -C "your_email@example.com"

# Start the SSH agent
eval "$(ssh-agent -s)"

# Add your key
ssh-add ~/.ssh/id_ed25519
```

## Startup Issues

### Port Already in Use

**Symptom:** `yeast up` fails with "port already in use"

**Error:**
```
Error: port 2222 is already in use
```

**Diagnosis:**

```bash
# Find what's using the port
ss -tlnp | grep 2222
lsof -i :2222
```

**Fix:**

1. **Change the port in `yeast.yaml`:**
   ```yaml
   instances:
     - name: web
       ssh_port: 2223  # Use a different port
   ```

2. **Or stop the process using the port:**
   ```bash
   # Find the PID
   ss -tlnp | grep 2222
   
   # Kill it (use with caution)
   kill <PID>
   ```

3. **Or clean up orphaned QEMU processes:**
   ```bash
   yeast clean
   ```

### VM Won't Start

**Symptom:** `yeast up` fails without clear error

**Diagnosis checklist:**

```bash
# 1. Check if the project is initialized
ls .yeast/project.json

# 2. Check yeast.yaml syntax
yeast doctor

# 3. Check if image is pulled
ls ~/.yeast/cache/images/ubuntu-24.04/

# 4. Check for disk space
df -h ~/

# 5. Check VM logs
yeast logs web

# 6. Check if QEMU process exists
ps aux | grep qemu | grep -v grep

# 7. Try running QEMU manually to see errors
# (Advanced: check the QEMU command in vm.log)
```

### SSH Timeout

**Symptom:** `yeast up` hangs at "Waiting for SSH"

**Diagnosis:**

```bash
# 1. Check if VM is actually running
yeast status

# 2. Check if QEMU process exists
ps aux | grep qemu

# 3. Check VM logs for boot errors
yeast logs web

# 4. Try manual SSH with verbose output
ssh -p 2222 -v yeast@127.0.0.1

# 5. Check if port is listening
ss -tlnp | grep 2222
```

**Common causes:**

| Cause | Fix |
|---|---|
| Slow first boot | Wait longer (up to 2 minutes) |
| Cloud-init failing | Check `yeast logs` for cloud-init errors |
| Wrong SSH key | Verify `~/.ssh/id_ed25519.pub` exists |
| Port not forwarded | Check `yeast.yaml` ssh_port setting |
| QEMU crash | Check `yeast logs` and `yeast doctor` |

### Image Not Found

**Symptom:** `yeast up` fails with "image not found"

**Fix:**

```bash
# Pull the image first
yeast pull ubuntu-24.04

# List available images
yeast pull --list
```

## Runtime Issues

### Provisioning Fails

**Symptom:** `yeast up` shows provisioning errors

**Diagnosis:**

```bash
# Check provisioning logs
yeast logs web

# SSH into VM and check manually
yeast ssh web

# Check package installation
dpkg -l | grep <package>

# Check file permissions
ls -la /path/to/file

# Check service status
systemctl status <service>
```

**Common fixes:**

| Issue | Fix |
|---|---|
| Package not found | Use correct package name for Ubuntu/Debian |
| Permission denied | Use `sudo` in shell commands or fix permissions |
| File not found | Check `source` path is relative to project root |
| Service not starting | Check service logs: `journalctl -u <service>` |

### Can't Copy Files

**Symptom:** `yeast copy` fails

**Diagnosis:**

```bash
# Check source file exists
ls -la ./source-file

# Check destination path is absolute
# Bad:  files/destination.txt
# Good: /home/yeast/destination.txt

# Check permissions on destination directory
yeast exec web -- ls -la /path/to/
```

### VMs Can't Ping Each Other

**Symptom:** VMs on lab network can't communicate

**Diagnosis:**

```bash
# 1. Check both VMs have lab IPs
yeast status

# 2. Check interfaces inside VMs
yeast exec web -- ip addr show yeastlab0
yeast exec db -- ip addr show yeastlab0

# 3. Check IP addresses match yeast.yaml
# 4. Check multicast address is the same
ps aux | grep qemu | grep mcast

# 5. Try restarting both VMs
yeast down && yeast up
```

## State Issues

### Project Not Initialized

**Symptom:** `yeast up` fails with "project not initialized"

**Fix:**

```bash
# Initialize the project
yeast init

# Or verify you're in the right directory
pwd
ls yeast.yaml
```

### Orphaned QEMU Processes

**Symptom:** VMs show as running in `ps` but not in `yeast status`

**Fix:**

```bash
# Clean up orphaned processes
yeast clean

# Or manually kill QEMU processes
ps aux | grep qemu
kill <PID>
```

### State File Corrupted

**Symptom:** Strange errors about state

**Fix:**

```bash
# Remove state and let Yeast rebuild
rm .yeast/state.json
# Or: yeast clean
```

**Warning:** `yeast clean` removes runtime state but keeps disks. `yeast destroy` removes everything.

## Error Messages Reference

### "port XXXX is already in use"

Another process is using the port. Find and stop it, or change the port in `yeast.yaml`.

### "SSH timeout"

The VM didn't become reachable within the timeout. Check VM logs and wait longer.

### "image not found"

Run `yeast pull <image>` to download the base image.

### "project not initialized"

Run `yeast init` in your project directory.

### "KVM not available"

Enable KVM or accept slower TCG mode. See [Installation](./installation).

### "qemu-system-x86_64 not found"

Install QEMU. See [Installation](./installation).

### "permission denied on /dev/kvm"

Add your user to the `kvm` group and log out/back in.

## Getting More Help

If you're still stuck:

1. **Run `yeast doctor`** — It checks most common issues
2. **Check the logs** — `yeast logs <instance>` shows VM console output
3. **Run with debug** — `YEAST_DEBUG=1 yeast up` for verbose output
4. **Check GitHub Issues** — [github.com/Twarga/yeast/issues](https://github.com/Twarga/yeast/issues)
5. **Open a new issue** — Include:
   - `yeast doctor` output
   - `yeast.yaml` (sanitized if needed)
   - Error message
   - `yeast logs` output

## Prevention Checklist

Before running Yeast:

- [ ] `yeast doctor` passes all checks
- [ ] Image is pulled (`yeast pull --list`)
- [ ] `ssh_port` values are unique
- [ ] `host_port` values are unique
- [ ] Lab network IPs are within CIDR range
- [ ] Provision sources exist
- [ ] Sufficient disk space (~2 GB per VM)
- [ ] User is in `kvm` group (for performance)

## Next Steps

- [Installation](./installation) — Detailed installation guide
- [Architecture](./architecture) — How Yeast works under the hood
- [Commands](./commands) — Complete CLI reference
- [GitHub Issues](https://github.com/Twarga/yeast/issues) — Report bugs
