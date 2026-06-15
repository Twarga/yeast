# Provisioning

Provisioning runs after a VM is reachable over SSH.

Yeast supports:

- `packages`
- `files`
- `shell`

## Example

```yaml
version: 1
provision:
  packages:
    - curl
  shell:
    - echo "project setup"
instances:
  - name: web
    image: ubuntu-24.04
    provision:
      packages:
        - caddy
      shell:
        - sudo systemctl enable --now caddy
```

## Provisioning Order

Yeast applies provisioning in this order:

1. install packages
2. copy files
3. run shell commands

Top-level provisioning is shared by instances. Instance-level provisioning adds work for that instance.

## File Copy

```yaml
provision:
  files:
    - source: ./site/index.html
      destination: /home/yeast/site/index.html
      permissions: "0644"
```

`source` is on the host. `destination` is inside the guest.

Quote `permissions` values so YAML keeps them as strings.

## Shell Commands

Shell commands run in the guest.

Use `sudo` when the command needs root:

```yaml
provision:
  shell:
    - sudo systemctl restart caddy
```

Write commands so they can run more than once. That makes `yeast provision` safer.

## Rerun Provisioning

```bash
yeast provision web
```

Or during `up`:

```bash
yeast up --reprovision
```

Skip provisioning:

```bash
yeast up --no-provision
```

## Common Mistakes

- package name is valid on one distro but not another
- file `source` path is wrong relative to the project folder
- shell command assumes a directory exists
- command needs `sudo`
- command is not idempotent and fails on rerun
