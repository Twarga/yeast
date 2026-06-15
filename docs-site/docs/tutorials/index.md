---
title: Tutorials
description: Step-by-step guided labs for learning Yeast
---

# Tutorials

Welcome to the Yeast tutorials! These hands-on labs will teach you how to use Yeast step by step.

## Prerequisites

Before starting the tutorials, make sure you have:
- Yeast installed (`yeast version`)
- KVM support enabled (`yeast doctor`)
- SSH key configured
- Basic command line knowledge

## Tutorial Map

### Fundamentals

Start here to learn the basics:

| # | Tutorial | Difficulty | Time | What You'll Learn |
|---|----------|------------|------|-------------------|
| 01 | [First VM](./01-first-vm) | Beginner | 15 min | Create, start, SSH, stop, destroy |
| 02 | [Provisioning](./02-provisioning) | Beginner | 20 min | Install packages, copy files, run commands |
| 03 | [Snapshots](./03-snapshots) | Beginner | 20 min | Take and restore snapshots |
| 04 | [Multi-VM Lab](./04-multi-vm-lab) | Intermediate | 30 min | Multiple VMs with private networking |

### Advanced

Build on the fundamentals:

| # | Tutorial | Difficulty | Time | What You'll Learn |
|---|----------|------------|------|-------------------|
| 05 | [Guest Control](./05-guest-control) | Intermediate | 20 min | exec, copy, logs, inspect |
| 06 | [LabsBackery Lab](./06-labsbackery-lab) | Intermediate | 45 min | Cybersecurity lab environment |
| 07 | [Templates](./07-templates) | Intermediate | 25 min | Built-in and custom templates |
| 08 | [JSON Automation](./08-json-automation) | Intermediate | 30 min | JSON output and event streams |

### Project Labs

Complex, real-world projects:

| # | Tutorial | Difficulty | Time | What You'll Learn |
|---|----------|------------|------|-------------------|
| 09 | [Nodi Home Lab](./09-nodi-home-lab) | Advanced | 60 min | 4-VM service architecture |
| 10 | [Load Balancer](./10-load-balancer-lab) | Advanced | 45 min | Caddy reverse proxy |
| 11 | [Database + App](./11-database-app-stack) | Advanced | 50 min | PostgreSQL + Node.js API |
| 12 | [Monitoring Stack](./12-monitoring-stack) | Advanced | 50 min | Prometheus + Grafana |
| 13 | [WireGuard VPN](./13-wireguard-vpn-mesh) | Advanced | 50 min | Encrypted VPN tunnels |
| 14 | [GitOps/CI](./14-gitops-ci-lab) | Advanced | 50 min | Gitea + Drone CI pipeline |

## How to Use Tutorials

1. **Start with Fundamentals** - Complete tutorials 01-04 first
2. **Follow the Steps** - Each tutorial has step-by-step instructions
3. **Run the Code** - Type the commands yourself
4. **Experiment** - Try modifying the examples
5. **Check Your Understanding** - Review the "What You Learned" section

## Getting Help

If you get stuck:
- Check the [Troubleshooting](/docs/troubleshooting) page
- Look at the "Common Failures" section in each tutorial
- Search [GitHub Issues](https://github.com/Twarga/yeast/issues)

## Next Steps

Start with [Tutorial 01: First VM](./01-first-vm) to begin your Yeast journey!
