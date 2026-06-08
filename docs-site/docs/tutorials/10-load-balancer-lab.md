---
title: Tutorial 10 - Load Balancer Lab
description: Caddy reverse proxy with 2 Flask backends
---

# Tutorial 10 - Load Balancer Lab

This walkthrough demonstrates setting up a load balancer with Caddy reverse proxy and multiple Flask backends.

## Create the project

```bash
mkdir 10-load-balancer-lab
cd 10-load-balancer-lab
yeast init
cp /path/to/yeast/examples/load-balancer-lab/yeast.yaml ./yeast.yaml
```

## Boot and validate

```bash
yeast pull ubuntu-24.04
yeast up
yeast status
```

## Test the load balancer

```bash
yeast exec caddy -- curl -fsS http://localhost:8080
yeast exec backend1 -- curl -fsS http://localhost:5000
yeast exec backend2 -- curl -fsS http://localhost:5000
```

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to set up a reverse proxy with Caddy
- How to load balance across multiple backends
- How to configure port forwarding
- How to test load balancing behavior

## Next Steps

- [Tutorial 11 - Database + App Stack](./11-database-app-stack) - PostgreSQL + Node.js API
