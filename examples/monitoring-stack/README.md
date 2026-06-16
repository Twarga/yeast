# monitoring-stack

A 4-VM monitoring stack for Yeast.

## What it does

- `monitor` — Prometheus + Grafana via Docker Compose
- `web`, `db`, `cache` — Node Exporter on each VM for metrics
- Prometheus scrapes all 3 nodes every 15 seconds

## Quick start

```bash
mkdir my-monitor-lab && cd my-monitor-lab
yeast init
cp -r /path/to/yeast/examples/monitoring-stack/* ./
yeast up
bash scripts/verify.sh
```

`yeast up` downloads the Ubuntu image automatically if it is not cached yet.

## Inspect

Public host port mappings are not part of Yeast v1.1. Inspect the services from inside the monitor VM:

```bash
yeast exec monitor -- curl -fsS http://localhost:9090
yeast exec monitor -- curl -fsS http://localhost:3000
```

## Note

This is an advanced example, not part of the beginner docs path yet.
