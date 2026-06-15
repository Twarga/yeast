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
- `management_host`
- `networks`
- `instances[].name`
- `instances[].hostname`
- `instances[].image`
- `instances[].memory`
- `instances[].cpus`
- `instances[].disk_size`
- `instances[].ssh_port`
- `instances[].user`
- `instances[].sudo`
- `instances[].env`
- `instances[].user_data`
- `provision`
- `provision.packages`
- `provision.files`
- `provision.shell`

Templates are project starters, not a separate config schema:

```bash
yeast init --list-templates
yeast init --template caddy-single-vm
```

List current images:

```bash
yeast pull --list
```

Useful image/cache commands:

```bash
yeast pull ubuntu-24.04
yeast pull --cached
yeast images clean ubuntu-24.04 --dry-run
```

Not supported in v1.1 config:

- `ports`
- `host_port`
- `guest_port`
