# monitoring-stack

A 4-VM monitoring stack for Yeast `v1.0`.

## What it does

- `monitor` — Prometheus + Grafana via Docker Compose
- `web`, `db`, `cache` — Node Exporter on each VM for metrics
- Prometheus scrapes all 3 nodes every 15 seconds

## Quick start

```bash
mkdir my-monitor-lab && cd my-monitor-lab
yeast init
cp -r /path/to/yeast/examples/monitoring-stack/* ./
yeast pull ubuntu-24.04
yeast up
bash scripts/verify.sh
```

## Browse

- Prometheus: http://127.0.0.1:9090
- Grafana: http://127.0.0.1:3030 (admin/admin)

## Full tutorial

See [Tutorial 12: Monitoring Stack](../../tutorials/12-monitoring-stack.md).
