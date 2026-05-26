# LabsBakery Lab Package Convention

Status: Draft for Yeast v0.9.0

This document defines the first file/folder convention for a Yeast-backed LabsBakery lab. It is intentionally small. The goal is to let LabsBakery bake a visual lab into a normal Yeast project without adding LabsBakery product state to Yeast.

## Purpose

A LabsBakery lab package is a portable source package. A LabsBakery lab session is a copied, initialized working directory created from that package.

Yeast should only consume:

- `yeast.yaml`
- provision source files referenced by `yeast.yaml`
- normal local template metadata when the package is used with `yeast init --template`

LabsBakery should consume:

- lab metadata
- topology coordinates
- instructions
- checks
- scoring
- visual notes
- import/export metadata

## Minimum Package Shape

Recommended source package:

```text
my-lab/
  template.yaml
  yeast.yaml
  lab.yaml
  README.md
  files/
    target/
      index.html
  scenario/
    instructions.md
    checks.yaml
  assets/
    diagram.png
```

Minimum runnable source package:

```text
my-lab/
  template.yaml
  yeast.yaml
  lab.yaml
```

Minimum Yeast-only template:

```text
my-template/
  template.yaml
  yeast.yaml
```

The Yeast-only template shape remains valid. LabsBakery fields are additive and should not be required by Yeast.

## `template.yaml`

`template.yaml` is the Yeast starter metadata. Yeast already understands this file for local templates.

Example:

```yaml
name: attacker-target-basic
title: Attacker Target Basic
description: Two Ubuntu machines on one private lab network.
category: lab
version: "1"
files:
  - yeast.yaml
  - lab.yaml
  - README.md
  - files/target/index.html
  - scenario/instructions.md
  - scenario/checks.yaml
```

Rules:

- Must include every file that `yeast init --template <path>` should copy into the session directory.
- Must keep paths inside the package.
- Must not include generated runtime files.
- Must not include `.yeast/`, disk images, snapshots, logs, or state files.

## `yeast.yaml`

`yeast.yaml` is the engine configuration. It describes machines, networks, provisioning, and guest access.

Example:

```yaml
version: 1
networks:
  - name: lab
    cidr: 10.10.10.0/24
instances:
  - name: attacker
    hostname: attacker-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    user: yeast
    sudo: nopasswd
    networks:
      - name: lab
        ipv4: 10.10.10.10
  - name: target
    hostname: target-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    user: yeast
    sudo: nopasswd
    networks:
      - name: lab
        ipv4: 10.10.10.20
    provision:
      files:
        - source: ./files/target/index.html
          destination: /home/yeast/index.html
          permissions: "0644"
```

Rules:

- LabsBakery may generate or edit `yeast.yaml` during bake.
- Yeast is the authority for fields it supports.
- LabsBakery should not put product-only metadata in `yeast.yaml`.
- Relative provision paths should stay inside the package/session directory.

## `lab.yaml`

`lab.yaml` is LabsBakery-owned metadata. Yeast should ignore it.

Example:

```yaml
schema: labsbakery.lab.v1
id: attacker-target-basic
title: Attacker Target Basic
summary: Practice connecting from an attacker machine to a target service.
version: "1"
author: TwargaOps
tags:
  - linux
  - networking
  - beginner
engine:
  type: yeast
  min_version: v0.9.0
  schema_version: yeast.v1
topology:
  nodes:
    - id: attacker
      instance: attacker
      role: attacker
      x: 120
      y: 160
    - id: target
      instance: target
      role: target
      x: 520
      y: 160
  networks:
    - id: lab
      name: lab
      cidr: 10.10.10.0/24
      nodes:
        - attacker
        - target
scenario:
  instructions: scenario/instructions.md
  checks: scenario/checks.yaml
baseline:
  name: clean
  description: Clean starting point
```

Required first fields:

- `schema`
- `id`
- `title`
- `version`
- `engine.type`
- `engine.min_version`
- `engine.schema_version`

Recommended first fields:

- `summary`
- `author`
- `tags`
- `topology.nodes`
- `topology.networks`
- `scenario.instructions`
- `scenario.checks`
- `baseline.name`

## `scenario/instructions.md`

Markdown instructions for the learner.

Example:

````markdown
# Connect To The Target

Open the attacker terminal and verify that the target is reachable:

```bash
ping -c 3 10.10.10.20
```
````

Yeast should not parse this file.

## `scenario/checks.yaml`

LabsBakery-owned check definitions. Yeast should not parse this file directly.

Example:

```yaml
schema: labsbakery.checks.v1
checks:
  - id: target-reachable
    title: Target SSH port is reachable
    instance: attacker
    command: bash -lc 'echo > /dev/tcp/10.10.10.20/22'
    expected_exit_code: 0
```

LabsBakery may execute checks by calling:

```bash
yeast exec <instance> --json -- <command...>
```

Rules:

- Commands are lab-authored and should be treated as trusted local lab content.
- LabsBakery owns scoring and check result storage.
- Yeast only runs the requested guest command and returns structured output.

## Session Shape

When a user starts a lab, LabsBakery should create a session directory by materializing or copying the package.

Example:

```text
~/.labsbakery/sessions/<session-id>/
  template.yaml
  yeast.yaml
  lab.yaml
  README.md
  files/
  scenario/
  assets/
  .yeast/
    project.json
```

Then LabsBakery should run Yeast commands from that session directory.

Session runtime files managed by Yeast remain under:

```text
~/.yeast/projects/<project-id>/
```

LabsBakery may keep its own session database record pointing to:

- session id
- session directory
- Yeast project id
- lab package id/version
- learner progress
- baseline status

## Baseline Convention

For v0.9/v1 local labs, the first baseline name should be:

```text
clean
```

Recommended baseline creation flow:

```bash
yeast up --json --events
yeast down --json --events
yeast snapshot <instance> clean --description "Clean baseline" --json
```

For multi-VM labs, create one `clean` snapshot per instance while the full lab is stopped.

Recommended reset flow:

```bash
yeast down --json --events
yeast restore <instance> clean --json --events
yeast up --json --events
```

For multi-VM labs, restore every instance snapshot before starting the lab again.

## Import/Export Boundary

LabsBakery export packages may later use:

```text
<lab-id>.lbz
```

For now, `.lbz` should be treated as a LabsBakery concern. Yeast should not read or write `.lbz` archives in v0.9.

If export is needed early, LabsBakery can zip the source package files and exclude:

- `.yeast/`
- `*.qcow2`
- `*.iso`
- `*.log`
- runtime state
- downloaded base images

## What Yeast Should Not Do In v0.9

Do not add these to Yeast for this convention:

- LabsBakery lab registry
- `.lbz` archive import/export
- visual topology fields in `yeast.yaml`
- learner progress
- scoring state
- course state
- user accounts
- browser terminal sessions
- marketplace metadata

## First Example Target

The first package that should be built from this convention is:

```text
examples/labsbackery-attacker-target-basic/
  template.yaml
  yeast.yaml
  lab.yaml
  README.md
  files/target/index.html
  scenario/instructions.md
  scenario/checks.yaml
```

It should prove:

- two VM instances
- one private lab network
- static lab IPs
- terminal connection metadata through `status --json`
- one target-side file or service
- one check executed through `yeast exec`
- `clean` baseline snapshot
- reset through restore
