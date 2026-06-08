---
title: Tutorial 12 - Monitoring Stack
description: Prometheus + Grafana observability stack
---

# Tutorial 12 - Monitoring Stack

This walkthrough demonstrates setting up an observability stack with Prometheus and Grafana.

## Create the project

```bash
mkdir 12-monitoring-stack
cd 12-monitoring-stack
yeast init
cp /path/to/yeast/examples/monitoring-stack/yeast.yaml ./yeast.yaml
```

## Boot and validate

```bash
yeast pull ubuntu-24.04
yeast up
yeast status
```

## Test the monitoring stack

```bash
yeast exec monitor -- curl -fsS http://localhost:9090/-/healthy
yeast exec monitor -- curl -fsS http://localhost:3000/api/health
```

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to set up Prometheus for metrics collection
- How to set up Grafana for visualization
- How to configure node exporters on target VMs
- How to create dashboards for monitoring

## Next Steps

- [Tutorial 13 - WireGuard VPN Mesh](./13-wireguard-vpn-mesh) - Encrypted VPN tunnels
