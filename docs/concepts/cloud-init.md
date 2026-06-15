# Cloud-Init

Yeast uses cloud-init to prepare Linux cloud images on first boot.

Cloud-init handles:

- hostname
- user creation
- SSH authorized key
- sudo policy
- first-boot guest setup

## Example

```yaml
version: 1
instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    user: yeast
    sudo: nopasswd
```

## Provisioning Comes After

Cloud-init prepares the guest so Yeast can SSH in.

Then Yeast provisioning can install packages, copy files, and run shell commands.

## Manual Images

Some manual images do not support cloud-init. Those images may not support automatic user setup or normal provisioning behavior.
