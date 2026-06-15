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

## Snapshot Missing

Check the exact instance and snapshot name:

```bash
yeast snapshots web
```

Snapshot names are per instance. A snapshot for `web` is not automatically a snapshot for another VM.

## Restore Did Not Remove A File

Confirm the marker was created after the snapshot:

```bash
yeast up
yeast exec web -- test -e /home/yeast/marker
yeast down
yeast restore web baseline
yeast up
yeast exec web -- test ! -e /home/yeast/marker
```

If the final command fails, inspect the VM and logs before destroying the project.
