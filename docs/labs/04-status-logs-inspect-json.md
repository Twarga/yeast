# Lab 04: Status, Logs, Inspect, JSON

Observe a Yeast project as both a human and an automation tool.

You will learn:

- when to use `yeast status`
- when to use `yeast logs`
- when to use `yeast inspect`
- how `--json` changes output
- how `--json --events` streams lifecycle events

## What You Will Build

```text
yeast-lab-04/
└── web VM
    ├── human status output
    ├── detailed inspect output
    ├── runtime logs
    └── JSON output for scripts
```

## Before You Start

Run:

```bash
yeast doctor
```

## Step 1: Create And Start

```bash
mkdir yeast-lab-04
cd yeast-lab-04
yeast init --template ubuntu-basic
yeast up
```

## Step 2: Use Human Output

```bash
yeast status
yeast inspect web
yeast logs web --tail 50
```

Human output is for terminals. It is allowed to be friendly, formatted, and easier to read.

## Step 3: Use JSON Output

```bash
yeast status --json
yeast inspect web --json
```

JSON output is for scripts and integrations. It uses the `yeast.v1` schema version.

## Step 4: See The Error Shape

Run this outside a Yeast project or from a temporary folder:

```bash
mkdir ../not-a-yeast-project
cd ../not-a-yeast-project
yeast status --json
cd ../yeast-lab-04
```

The output should still be JSON, with `ok: false` and a structured error.

## Step 5: Stream Events

```bash
yeast down
yeast up --json --events
```

Events are JSON Lines. Each line is one event object. Tools can read progress as Yeast works instead of waiting for the final result.

## Verification

Check that these commands work:

```bash
yeast status --json
yeast inspect web --json
yeast logs web --tail 20
```

## Clean Up

```bash
yeast down
yeast destroy
cd ..
rm -rf not-a-yeast-project
```

## What You Learned

Use human output when you are driving Yeast yourself.

Use JSON and events when another tool is driving Yeast.

## Next Lab

Continue with [Snapshots And Restore](05-snapshots-and-restore.md).
