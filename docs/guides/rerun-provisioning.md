# Rerun Provisioning

Use `yeast provision` when a VM is already running and you want to reapply provisioning.

```bash
yeast provision web
```

Use `--reprovision` when starting:

```bash
yeast up --reprovision
```

Use `--no-provision` when you want to boot without running provision steps:

```bash
yeast up --no-provision
```

Write provisioning steps so they are safe to run again.
