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

## Runtime State

Yeast stores runtime state separately. Runtime state records what actually exists, including assigned SSH ports, VM status, disk paths, logs, and snapshots.

## Important Habit

Run Yeast commands from the project folder.

```bash
yeast status
yeast up
yeast down
```
