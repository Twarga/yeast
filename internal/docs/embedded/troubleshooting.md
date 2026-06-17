# Yeast Troubleshooting

Common checks:

```bash
yeast doctor
yeast status --json
yeast logs web --tail 100
yeast inspect web
ps aux | grep qemu
ls -l /dev/kvm
```

Common problems:

- `/dev/kvm` missing
- user not in KVM group
- `qemu-system-x86_64` not installed
- `genisoimage` or `mkisofs` missing
- no SSH public key at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`
- image cache missing or corrupted
- SSH readiness timeout after first boot
- provisioning command failed inside the guest

Useful recovery commands:

```bash
yeast down
yeast up --reprovision
yeast images clean ubuntu-24.04 --dry-run
```

If a VM does not start cleanly, save the exact command, `yeast logs <instance> --tail 100`, and `yeast inspect <instance> --json` before cleanup.
