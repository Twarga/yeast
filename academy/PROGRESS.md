# DevOps Bootcamp With Yeast — Progress

## Current Reality

The writing draft is complete. Runtime verification is not complete.

Do not mark a lab as tested until it has been run on a real Linux/KVM host with Yeast v1.1 and its validation script passes after following the lab.

## Status Legend

```
[ ] Not started
[~] In progress
[x] Written — lab.md and assets exist
[t] Tested on real KVM host
```

## Hardening Checklist

Before marking a lab `[t]`:

- `yeast doctor` passes on the host.
- `yeast up` succeeds.
- `yeast status` shows all expected instances running.
- Every Yeast command used by the lab exists in `SUPPORTED_YEAST_V1_1.md`.
- No unsupported `yeast.yaml` fields are used.
- Any laptop browser access uses `ACCESS.md` SSH tunnel patterns.
- `bash assets/validate.sh` passes.
- `yeast destroy` cleans up successfully.

## Labs

| # | Lab | Written | Tested |
|---:|---|---|---|
| 01 | Linux Server Baseline | [x] | [ ] |
| 02 | Static Site With Nginx | [x] | [ ] |
| 03 | Reverse Proxy To Backend App | [x] | [ ] |
| 04 | Database-Backed App | [x] | [ ] |
| 05 | Linux Troubleshooting Drill | [x] | [ ] |
| 06 | Bash Automation For Server Setup | [x] | [ ] |
| 07 | Ansible For One Server | [x] | [ ] |
| 08 | Ansible For Multi-VM Web Cluster | [x] | [ ] |
| 09 | Secrets And Configuration Management | [x] | [ ] |
| 10 | Docker Fundamentals On A VM | [x] | [ ] |
| 11 | Compose Multi-Service App | [x] | [ ] |
| 12 | Container Build, Scan, And Hardening | [x] | [ ] |
| 13 | CI With GitHub Actions | [x] | [ ] |
| 14 | Self-Hosted CI Runner | [x] | [ ] |
| 15 | Private Container Registry | [x] | [ ] |
| 16 | Progressive Delivery And Rollback | [x] | [ ] |
| 17 | Prometheus And Grafana Monitoring | [x] | [ ] |
| 18 | Centralized Logging | [x] | [ ] |
| 19 | OpenTelemetry Distributed Tracing | [x] | [ ] |
| 20 | SRE: SLOs, Alerts, Error Budgets | [x] | [ ] |
| 21 | Backup And Restore Drill | [x] | [ ] |
| 22 | Chaos And Failure Recovery Drill | [x] | [ ] |
| 23 | Terraform Fundamentals | [x] | [ ] |
| 24 | Terraform Modules And Environments | [x] | [ ] |
| 25 | GitOps With Argo CD | [x] | [ ] |
| 26 | End-To-End Delivery Platform | [x] | [ ] |
| 27 | Kubernetes Foundations With k3s | [x] | [ ] |
| 28 | Kubernetes Networking, Config, Storage | [x] | [ ] |
| 29 | Kubernetes Delivery Capstone | [x] | [ ] |
| 30 | AI-Assisted DevOps And Local LLM Ops | [x] | [ ] |
