---
title: Tutorial 11 - Database + App Stack
description: PostgreSQL + Node.js Express API
---

# Tutorial 11 - Database + App Stack

This walkthrough demonstrates setting up a stateful application with PostgreSQL database and Node.js Express API.

## Create the project

```bash
mkdir 11-database-app-stack
cd 11-database-app-stack
yeast init
cp /path/to/yeast/examples/database-app-stack/yeast.yaml ./yeast.yaml
```

## Boot and validate

```bash
yeast pull ubuntu-24.04
yeast up
yeast status
```

## Test the application

```bash
yeast exec app -- curl -fsS http://localhost:3000/health
yeast exec db -- sudo -u postgres psql -c "SELECT 1"
```

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to set up a PostgreSQL database
- How to create a Node.js API that connects to the database
- How to work with stateful applications
- How to test database connectivity

## Next Steps

- [Tutorial 12 - Monitoring Stack](./12-monitoring-stack) - Prometheus + Grafana
