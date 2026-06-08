---
title: Troubleshooting
description: Common issues and fixes
---

# Troubleshooting

This page helps you diagnose and fix common issues.

## System Check

First, run the doctor command:

```bash
yeast doctor
```

## Common Issues

### KVM Not Available

**Symptom:** `yeast doctor` reports KVM not available

**Fix:**

```bash
sudo modprobe kvm
sudo modprobe kvm_intel  # or kvm_amd
sudo usermod -aG kvm $USER
```

### QEMU Not Installed

**Symptom:** `yeast doctor` reports QEMU not installed

**Fix:**

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y qemu-kvm qemu-utils

# Fedora
sudo dnf install -y qemu-kvm qemu-img
```

### SSH Key Missing

**Symptom:** `yeast doctor` reports SSH key not found

**Fix:**

```bash
ssh-keygen -t ed25519 -C "your_email@example.com"
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
```

### Port Already in Use

**Symptom:** `yeast up` fails with "port already in use"

**Fix:**

```bash
ss -tlnp | grep 8080
# Kill the process or change the port in yeast.yaml
```

### VM Won't Start

**Debugging:**

```bash
yeast status
yeast logs web
ps aux | grep qemu
```

### SSH Timeout

**Debugging:**

```bash
yeast status
ss -tlnp | grep 2222
ssh -p 2222 yeast@127.0.0.1
```

### Provisioning Fails

**Debugging:**

```bash
yeast logs web
yeast ssh web
dpkg -l | grep nginx
systemctl status nginx
```

### Can't Ping Between VMs

**Debugging:**

```bash
yeast exec web -- ip addr show
yeast exec db -- ip addr show
ps aux | grep qemu | grep mcast
```

## Error Messages

### "port XXXX is already in use"

Another process is using the port. Find and stop it, or change the port.

### "SSH timeout"

The VM didn't become reachable within the timeout. Wait longer or check VM logs.

### "image not found"

Run `yeast pull <image>` to download the base image.

### "project not initialized"

Run `yeast init` in your project directory.

## Getting Help

If you're still stuck:

1. Check the [Known Limitations](./known-limitations) page
2. Search [GitHub Issues](https://github.com/Twarga/yeast/issues)
3. Open a new issue with relevant details

## Next Steps

- [Known Limitations](./known-limitations) - What Yeast doesn't support yet
- [Commands](./commands) - CLI command reference
- [Configuration](./configuration) - yeast.yaml reference
