# KVM Troubleshooting

Check for KVM:

```bash
ls -la /dev/kvm
lsmod | grep kvm
```

If your user cannot access `/dev/kvm`:

```bash
sudo usermod -aG kvm "$USER"
```

Log out and back in.

Without KVM, QEMU may fall back to slower software emulation.
