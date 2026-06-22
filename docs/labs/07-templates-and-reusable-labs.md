# Lab 07: Templates And Reusable Labs

Use built-in templates, inspect what they copy, and turn a normal Yeast project into a reusable local template.

You will learn:

- how to list built-in templates
- how `yeast init --template` works
- why templates are project starters
- how local templates are structured
- how reusable labs stay ordinary and editable

## What You Will Build

```text
yeast-lab-07/
├── generated/
│   └── project from caddy-single-vm
└── local-template/
    ├── template.yaml
    ├── yeast.yaml
    └── README.md
```

## Before You Start

Run:

```bash
yeast doctor
```

## Step 1: List Templates

```bash
yeast init --templates
```

`yeast init --list-templates` is the longer form of the same command.

Built-in templates in v1.1:

| Template | Purpose |
|---|---|
| `ubuntu-basic` | minimal single Ubuntu VM |
| `caddy-single-vm` | Ubuntu VM with Caddy provisioning |
| `two-vm-lab` | two Ubuntu VMs on one private network |

For scripts:

```bash
yeast init --templates --json
```

## Step 2: Create A Project From A Template

```bash
mkdir yeast-lab-07
cd yeast-lab-07
mkdir generated
cd generated
yeast init --template caddy-single-vm
```

## Step 3: Inspect The Copied Files

```bash
ls -la
ls -la site
cat yeast.yaml
```

After a template is copied, the result is just a normal Yeast project. You can edit `yeast.yaml`, `README.md`, or any copied files.

## Step 4: Start And Stop The Generated Project

```bash
yeast up
yeast exec web -- systemctl is-active caddy
yeast down
```

Expected output from the `exec` command:

```text
active
```

## Step 5: Create A Small Local Template

Move back to the lab root:

```bash
cd ..
mkdir local-template
cd local-template
```

Create `template.yaml`:

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

Create `yeast.yaml`:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    user: yeast
    sudo: nopasswd
```

Create `README.md`:

```markdown
# Local Ubuntu Basic

Reusable local Yeast starter.
```

## Step 6: Use The Local Template

```bash
cd ..
mkdir from-local-template
cd from-local-template
yeast init --template ../local-template
ls -la
cat yeast.yaml
```

You should see the files copied from the local template.

## Clean Up

From `yeast-lab-07/from-local-template`:

```bash
yeast destroy
cd ../generated
yeast destroy
```

If either project was already stopped or never started, cleanup may simply report that there is no running VM to stop.

## What You Learned

Templates are not a second kind of Yeast project. They are a repeatable way to copy starter files into a new project folder.

This makes labs reusable without hiding how Yeast works.

## Finish

You finished the public Yeast mini bootcamp.

Good next pages:

- [Commands](../reference/commands.md)
- [yeast.yaml Reference](../reference/yeast-yaml.md)
- [Images](../reference/images.md)
