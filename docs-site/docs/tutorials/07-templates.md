---
title: Tutorial 07 - Templates
description: Built-in and custom templates for quick project setup
---

# Tutorial 07 - Templates

Yeast ships with built-in starters for the common paths.

## Discover built-in templates

```bash
yeast init --list-templates
```

## Try each template in a fresh folder

```bash
mkdir 07-templates
cd 07-templates

mkdir ubuntu-basic
cd ubuntu-basic
yeast init --template ubuntu-basic
cd ..

mkdir caddy-single-vm
cd caddy-single-vm
yeast init --template caddy-single-vm
cd ..

mkdir two-vm-lab
cd two-vm-lab
yeast init --template two-vm-lab
```

Expected result:

- `ubuntu-basic` creates a minimal `yeast.yaml`
- `caddy-single-vm` also creates a `site/` directory
- `two-vm-lab` creates a two-instance networked starter config

## What You Learned

- How to list available templates
- How to initialize projects from templates
- How templates provide starter configurations

## Next Steps

- [Tutorial 08 - JSON And Events](./08-json-automation) - JSON output and event streams
