# Yeast Troubleshooting

Common checks:

```bash
yeast doctor
yeast status --json
ps aux | grep qemu
ls -l /dev/kvm
```

Common problems:

- `/dev/kvm` missing
- user not in KVM group
- `qemu-system-x86_64` not installed
- `genisoimage` or `mkisofs` missing
- no SSH public key at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`

If a VM does not start cleanly, save the exact command and full output before cleanup.
