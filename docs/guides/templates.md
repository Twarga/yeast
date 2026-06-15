# Templates

Templates are project starters.

They copy files into your current folder. After that, the folder is a normal Yeast project that you can edit.

## List Built-In Templates

```bash
yeast init --list-templates
```

For scripts:

```bash
yeast init --list-templates --json
```

Built-in templates in v1.1:

| Template | Category | Purpose |
|---|---|---|
| `ubuntu-basic` | `vm` | minimal Ubuntu VM starter |
| `caddy-single-vm` | `app` | one Ubuntu VM with Caddy provisioning |
| `two-vm-lab` | `lab` | two Ubuntu VMs on one private network |

## Create From A Built-In Template

```bash
mkdir my-yeast-project
cd my-yeast-project
yeast init --template ubuntu-basic
```

Inspect the result:

```bash
find . -maxdepth 3 -type f | sort
sed -n '1,180p' yeast.yaml
```

## Create From A Local Template Directory

A local template is a folder with `template.yaml` plus the files listed inside it.

Example `template.yaml`:

```yaml
name: local-ubuntu-basic
title: Local Ubuntu Basic
description: Local reusable Ubuntu starter for Yeast.
category: vm
version: "1"
files:
  - yeast.yaml
  - README.md
```

Use it:

```bash
mkdir project-from-local-template
cd project-from-local-template
yeast init --template ../local-ubuntu-basic
```

## Template Rules

- Template files are copied into the project folder.
- A template does not create a special project type.
- After `yeast init`, edit `yeast.yaml` normally.
- Local templates are useful for repeating your own lab layout.

## Verify A Template Project

```bash
yeast up
yeast status
yeast down
yeast destroy
```

## Good Next Step

Try [Templates And Reusable Labs](../labs/07-templates-and-reusable-labs.md) for a full walkthrough.
