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

Yeast starts QEMU with `-enable-kvm` and has no software-emulation fallback —
without a readable/writable `/dev/kvm`, `yeast up` fails before the VM boots.
