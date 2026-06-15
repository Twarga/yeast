---
title: Tutorial 03 - Snapshots And Reset
description: Take and restore snapshots to save VM state
---

# Tutorial 03 - Snapshots And Reset

This walkthrough builds on [Tutorial 02](./02-provisioning) and the same Caddy example.

## Create a clean baseline

```bash
yeast down
yeast snapshot web clean --description "Provisioned Caddy baseline"
yeast snapshots web
```

## Break the guest

```bash
yeast up
yeast exec web -- sudo rm -f /var/www/html/index.html
yeast down
```

## Restore it

```bash
yeast restore web clean
yeast up
yeast exec web -- curl -fsS http://127.0.0.1
```

## Cleanup

```bash
yeast down
yeast delete-snapshot web clean
yeast destroy
```

Expected result:

- `yeast snapshots web` lists `clean`
- after restore, the site responds again

## What You Learned

- How to take snapshots of stopped VMs
- How to restore VMs from snapshots
- How to use snapshots for rollback and testing

## Next Steps

- [Tutorial 04 - Multi-VM Lab](./04-multi-vm-lab) - Multiple VMs with private networking
