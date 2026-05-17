# Yeast Installation

Yeast v0.1 targets Linux hosts with KVM.

Install from the v0.1.0 release:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/v0.1.0/install.sh | YEAST_REF=v0.1.0 bash
```

The installer attempts to:

- detect the package manager
- install QEMU/KVM and SSH tooling
- install or bootstrap Go when needed
- create `~/.yeast/cache/images`
- generate an SSH key if missing
- run `yeast doctor`

If the installer adds your user to the KVM group, log out and back in before running `yeast up`.

Check the result:

```bash
yeast version
yeast doctor
```
