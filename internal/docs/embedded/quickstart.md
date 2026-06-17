# Yeast Quickstart

Yeast is Linux-first and QEMU/KVM-first.

Core flow:

```bash
yeast doctor
mkdir my-lab
cd my-lab
yeast init --template ubuntu-basic
yeast up
yeast status
yeast ssh web
yeast down
yeast destroy
```

`yeast up` downloads the trusted image automatically if it is not cached yet.

What Yeast handles:

- project metadata
- project templates
- trusted image cache
- cloud-init seed files
- qcow2 overlay disk
- QEMU/KVM startup
- SSH readiness
- project state tracking

Use `yeast status --json` for automation.

Use lifecycle events when another tool needs progress:

```bash
yeast up --json --events
```

Template commands:

```bash
yeast init --list-templates
yeast init --template caddy-single-vm
yeast init --template two-vm-lab
```

Useful image/cache commands:

```bash
yeast pull --list
yeast pull --cached
```

List offline terminal docs:

```bash
yeast docs --list
```
