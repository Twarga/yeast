# Yeast Quickstart

Yeast v0.1 is Linux-first and QEMU/KVM-first.

Core flow:

```bash
yeast doctor
mkdir yeast-ubuntu-basic
cd yeast-ubuntu-basic
yeast init
yeast pull ubuntu-24.04
yeast up
yeast status
yeast ssh web
yeast down
yeast destroy
```

What Yeast handles:

- project metadata
- trusted image cache
- cloud-init seed files
- qcow2 overlay disk
- QEMU/KVM startup
- SSH readiness
- project state tracking

Use `yeast status --json` for automation.
