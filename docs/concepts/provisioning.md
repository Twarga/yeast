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
