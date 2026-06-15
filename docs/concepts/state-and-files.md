# State And Files

Yeast separates project configuration from runtime state.

## Project Files

```text
project/
├── yeast.yaml
└── .yeast/project.json
```

## Shared Yeast Home

By default, Yeast uses:

```text
~/.yeast/
```

Common contents:

```text
~/.yeast/cache/images/      cached base images
~/.yeast/projects/          project runtime files
```

Runtime files include VM disks, cloud-init data, logs, QMP sockets, and snapshots.
