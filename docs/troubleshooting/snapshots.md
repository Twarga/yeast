# Snapshot Troubleshooting

Snapshots require stopped VMs.

```bash
yeast down
yeast snapshot web baseline
```

Restore also requires a stopped VM:

```bash
yeast down
yeast restore web baseline
```

If restore behaves unexpectedly, list snapshots and inspect state:

```bash
yeast snapshots web
yeast inspect web
```
