# DevOps Bootcamp With Yeast — Curriculum

## Course Contract

This curriculum is for learning DevOps with Yeast, not for documenting imaginary Yeast features.

Every lab must stay inside the Yeast v1.1 surface documented in `SUPPORTED_YEAST_V1_1.md`.

Most importantly:

- Yeast v1.1 supports SSH management forwarding through `ssh_port`.
- Yeast v1.1 does not document a general `ports:` field in `yeast.yaml`.
- Browser access from the laptop must use SSH local forwarding. See `ACCESS.md`.
- `yeast ssh` is interactive and supports `--verbose`; it is not a raw SSH argument passthrough command.

## Learning Flow

```
Labs 01-05  →  Systems & Linux fundamentals
Labs 06-09  →  Automation (Bash, Ansible, secrets)
Labs 10-12  →  Containers
Labs 13-16  →  CI/CD
Labs 17-20  →  Observability & SRE
Labs 21-22  →  Reliability drills
Labs 23-24  →  Terraform / IaC
Labs 25-26  →  GitOps & VM platform capstone
Labs 27-29  →  Kubernetes
Lab  30     →  AI/LLM ops
```

## Labs

| # | Lab | Focus |
|---:|---|---|
| 01 | Linux Server Baseline | Clean operational Linux server |
| 02 | Static Site With Nginx | Web server setup and logs |
| 03 | Reverse Proxy To Backend App | Proxy/backend traffic flow |
| 04 | Database-Backed App | Persistence and app-to-DB connectivity |
| 05 | Linux Troubleshooting Drill | Logs-first debugging |
| 06 | Bash Automation For Server Setup | Repeatable shell automation |
| 07 | Ansible For One Server | Idempotent config management |
| 08 | Ansible For Multi-VM Web Cluster | Multi-host orchestration |
| 09 | Secrets And Configuration Management | Config and secret separation |
| 10 | Docker Fundamentals On A VM | Containers, volumes, logs |
| 11 | Compose Multi-Service App | Compose service discovery and data |
| 12 | Container Build, Scan, And Hardening | Safer images and tagging |
| 13 | CI With GitHub Actions | Pipeline basics |
| 14 | Self-Hosted CI Runner | Runner infrastructure |
| 15 | Private Container Registry | Image promotion and rollback |
| 16 | Progressive Delivery And Rollback | Safer release patterns |
| 17 | Prometheus And Grafana Monitoring | Metrics and dashboards |
| 18 | Centralized Logging | Multi-service log search |
| 19 | OpenTelemetry Distributed Tracing | Traces and spans |
| 20 | SRE: SLOs, Alerts, Error Budgets | Reliability thinking |
| 21 | Backup And Restore Drill | Recovery validation |
| 22 | Chaos And Failure Recovery Drill | Controlled failure practice |
| 23 | Terraform Fundamentals | Plan/apply/state |
| 24 | Terraform Modules And Environments | Reusable IaC |
| 25 | GitOps With Argo CD | Reconciliation and desired state |
| 26 | End-To-End Delivery Platform | VM platform capstone |
| 27 | Kubernetes Foundations With k3s | Cluster basics |
| 28 | Kubernetes Networking, Config, Storage | Ingress, ConfigMaps, PVCs |
| 29 | Kubernetes Delivery Capstone | Full K8s delivery flow |
| 30 | AI-Assisted DevOps And Local LLM Ops | Local AI for ops workflows |

## Per-Lab Structure

Each lab folder contains:

```
lab.md          ← the complete lab guide (read this)
assets/
  yeast.yaml    ← VM definition
  validate.sh   ← run to verify your work
  ...           ← any additional configs or scripts
```
