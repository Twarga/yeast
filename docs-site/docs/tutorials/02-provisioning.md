---
title: Tutorial 02 - Provisioning
description: Install packages, copy files, and configure VMs automatically
---

# Tutorial 02 - Provisioning

This walkthrough demonstrates provisioning with Caddy web server.

## Create the project

```bash
mkdir 02-provisioning
cd 02-provisioning
yeast init
cp /path/to/yeast/examples/caddy-single-vm/yeast.yaml ./yeast.yaml
mkdir -p site
cp /path/to/yeast/examples/caddy-single-vm/site/index.html ./site/index.html
cp /path/to/yeast/examples/caddy-single-vm/site/Caddyfile ./site/Caddyfile
```

## Boot and validate

```bash
yeast up
yeast exec web -- curl -fsS http://127.0.0.1
```

Expected result:

- provisioning installs `caddy`
- the copied site is served from inside the guest
- the HTML returned by `curl` matches `site/index.html`

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to use the `provision` section in yeast.yaml
- How to install packages automatically
- How to copy files from host to guest
- How to run shell commands during provisioning

## Next Steps

- [Tutorial 03 - Snapshots](./03-snapshots) - Save and restore VM state
