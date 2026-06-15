# Projects

A Yeast project is a folder with a `yeast.yaml` file and a `.yeast/project.json` identity file.

Create one:

```bash
mkdir my-lab
cd my-lab
yeast init
```

## Desired State

`yeast.yaml` describes what you want:

- instances
- images
- CPU and memory
- SSH ports
- private networks
- provisioning

This file should stay human-editable. It should not contain runtime facts like PIDs, QMP socket paths, or generated disk paths.

## Runtime State

Yeast stores runtime state separately. Runtime state records what actually exists, including assigned SSH ports, VM status, disk paths, logs, and snapshots.

Runtime state belongs to Yeast. You normally inspect it through commands:

```bash
yeast status
yeast inspect web
yeast snapshots web
```

## Important Habit

Run Yeast commands from the project folder.

```bash
yeast status
yeast up
yeast down
```

If you are not sure where you are, run:

```bash
pwd
ls yeast.yaml .yeast/project.json
```

## Project Safety

Use one folder per lab or experiment.

That keeps:

- VM disks separate
- snapshots separate
- state locks separate
- cleanup safer

When you are finished with a project:

```bash
yeast down
yeast destroy
```
