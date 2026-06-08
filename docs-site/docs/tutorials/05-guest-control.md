---
title: Tutorial 05 - Guest Control
description: Execute commands, copy files, and inspect VMs
---

# Tutorial 05 - Guest Control

This walkthrough focuses on Yeast's command and file-copy surface.

## Create the project

```bash
mkdir 05-guest-control
cd 05-guest-control
yeast init --template ubuntu-basic
printf 'smoke-artifact\n' > artifact.txt
```

## Use the guest control commands

```bash
yeast up --no-provision
yeast exec web -- whoami
yeast copy web --to-guest ./artifact.txt /home/yeast/artifact.txt
yeast copy web --from-guest /home/yeast/artifact.txt ./artifact-out.txt
yeast inspect web
yeast logs web --tail 20
```

Expected result:

- `yeast exec` returns `yeast`
- `artifact-out.txt` matches `artifact.txt`
- `inspect` shows runtime metadata and provisioning state

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to execute commands in VMs with `yeast exec`
- How to copy files between host and guest
- How to inspect VM details
- How to view VM logs

## Next Steps

- [Tutorial 06 - LabsBackery Lab](./06-labsbackery-lab) - Cybersecurity lab environment
