---
title: Tutorial 06 - LabsBackery Lab
description: Cybersecurity lab environment with attacker and target VMs
---

# Tutorial 06 - LabsBackery Lab

This walkthrough demonstrates setting up a cybersecurity lab environment with attacker and target VMs.

## Create the project

```bash
mkdir 06-labsbackery-lab
cd 06-labsbackery-lab
yeast init --template /path/to/yeast/examples/labsbackery-attacker-target-basic
```

## Boot and validate with JSON

```bash
yeast up --json --events
yeast status --json
yeast exec attacker --json -- bash -lc 'echo > /dev/tcp/10.20.30.20/22'
yeast exec target --json -- bash -lc 'test -f /home/yeast/labsbackery-target.txt && grep -q labsbackery-ready /home/yeast/labsbackery-target.txt'
```

## Snapshot and restore

```bash
yeast down --json --events
yeast snapshot attacker clean --description "Clean attacker baseline" --json
yeast snapshot target clean --description "Clean target baseline" --json
yeast restore attacker clean --json --events
yeast restore target clean --json --events
yeast up --json --events
```

## Cleanup

```bash
yeast destroy --json --events
```

Expected result:

- both checks pass before snapshot
- the lab comes back after restore

## What You Learned

- How to use Yeast for cybersecurity labs
- How to work with attacker and target VMs
- How to use JSON output for automation
- How to snapshot and restore lab environments

## Next Steps

- [Tutorial 07 - Templates](./07-templates) - Built-in and custom templates
