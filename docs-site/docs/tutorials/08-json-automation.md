---
title: Tutorial 08 - JSON And Events
description: JSON output and event streams for automation
---

# Tutorial 08 - JSON And Events

This walkthrough focuses on the machine-readable contract.

## Create a project

```bash
mkdir 08-json-automation
cd 08-json-automation
yeast init --template ubuntu-basic
```

## Capture JSON and events

```bash
yeast up --json --events > up.jsonl
yeast status --json > status.json
yeast down --json --events > down.jsonl
yeast destroy --json --events > destroy.jsonl
```

## Inspect the artifacts

```bash
tail -n 5 up.jsonl
cat status.json
tail -n 3 destroy.jsonl
```

Expected result:

- `up.jsonl` contains event records followed by one final success envelope
- `status.json` contains instance metadata including `ssh_port`, `user`, and runtime paths
- `destroy.jsonl` ends with `workflow.completed` and a final success envelope

## What You Learned

- How to use JSON output for automation
- How to capture event streams
- How to parse and use JSON artifacts

## Next Steps

- [Tutorial 09 - Nodi Home Lab](./09-nodi-home-lab) - Complex multi-VM architecture
