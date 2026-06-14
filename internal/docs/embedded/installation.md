# Yeast Installation

Yeast v1.0 targets Linux hosts with KVM.

Install the latest stable release:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

Install the explicit v1.0.0 release:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | YEAST_REF=v1.0.0 bash
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
