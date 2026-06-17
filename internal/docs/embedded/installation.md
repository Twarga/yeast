# Yeast Installation

Yeast targets Linux hosts with QEMU/KVM.

Install the latest stable release:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

Install the explicit v1.1.0 release:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | YEAST_VERSION=v1.1.0 bash
```

The default installer:

- downloads the release binary
- verifies the checksum
- creates `~/.yeast/cache/images`
- creates a default SSH key if needed
- runs `yeast doctor --fix --yes`

It does not build from source or install Go in the normal path.

If you prefer to install host packages yourself instead of letting `yeast doctor --fix --yes` do it, use:

```bash
sudo apt update
sudo apt install -y qemu-kvm qemu-utils genisoimage openssh-client
```

If `/dev/kvm` exists but your user cannot access it:

```bash
sudo usermod -aG kvm "$USER"
```

Then log out and back in before running `yeast up`.

Source-build mode is still available for contributors:

```bash
YEAST_INSTALL_MODE=source bash install.sh
```

Check the result:

```bash
yeast version
yeast doctor
```
