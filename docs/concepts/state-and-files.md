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

## Do Not Edit Runtime State By Hand

Prefer commands:

```bash
yeast status
yeast inspect web
yeast logs web --tail 100
yeast snapshots web
```

Manual edits to runtime files can make state disagree with real QEMU processes or disks.

## Locking

Yeast uses project state locking to avoid two operations changing the same project at the same time.

If a command says the project is locked, check whether another Yeast command is still running before deleting anything.

## Cleanup

Stop VMs:

```bash
yeast down
```

Remove tracked runtime files for the project:

```bash
yeast destroy
```

Clean cached images separately:

```bash
yeast images clean ubuntu-24.04 --dry-run
```
