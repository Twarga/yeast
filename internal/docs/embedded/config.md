# Yeast Config

Minimal config:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
```

Common fields:

- `version`
- `instances[].name`
- `instances[].image`
- `instances[].memory`
- `instances[].cpus`
- `instances[].disk_size`
- `instances[].ssh_port`
- `provision`
- `networks`

Templates are project starters, not a separate config schema:

```bash
yeast init --list-templates
yeast init --template caddy-single-vm
```

Current supported images:

- `ubuntu-22.04`
- `ubuntu-24.04`
