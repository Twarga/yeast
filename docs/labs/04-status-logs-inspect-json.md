# Lab 04: Status, Logs, Inspect, JSON

In this lab, you will observe a running Yeast project and use JSON output for automation.

You will learn:

- `yeast status`
- `yeast logs`
- `yeast inspect`
- `--json`
- `--events`

## Create And Start

```bash
mkdir yeast-lab-04
cd yeast-lab-04
yeast init --template ubuntu-basic
yeast up
```

## Human Output

```bash
yeast status
yeast inspect web
yeast logs web --tail 50
```

Use human output when you are working in a terminal.

## JSON Output

```bash
yeast status --json
yeast inspect web --json
```

Use JSON output when writing scripts.

## Event Stream

```bash
yeast down
yeast up --json --events
```

Events are JSON Lines. They are useful for tools that want progress updates.

## Clean Up

```bash
yeast down
yeast destroy
```

Next: [Snapshots And Restore](05-snapshots-and-restore.md).
