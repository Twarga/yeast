# Yeast Config

Default v0.1 config:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
```

Current v0.1 fields:

- `version`
- `instances[].name`
- `instances[].image`
- `instances[].memory`
- `instances[].cpus`

Current supported images:

- `ubuntu-22.04`
- `ubuntu-24.04`
