# Troubleshooting

Start with:

```bash
yeast doctor
```

Then gather:

```bash
yeast status
yeast logs <instance> --tail 100
yeast inspect <instance>
```

For automation:

```bash
yeast status --json
yeast inspect <instance> --json
```

Use the topic pages for specific failures.

## Fast Triage

Ask three questions:

1. Did the host pass `yeast doctor`?
2. Did the project config validate?
3. Did the VM boot but fail later?

Useful commands:

```bash
yeast doctor
yeast status --json
yeast inspect <instance> --json
yeast logs <instance> --tail 120
```

## Keep Evidence Before Cleanup

Before `yeast destroy`, capture:

```bash
yeast status
yeast inspect <instance>
yeast logs <instance> --tail 200
```

If you are reporting a bug, include the Yeast version:

```bash
yeast version
```
