# DevOps Bootcamp With Yeast Plan

Status: planning
Audience: private learning first
Scope: private DevOps course using Yeast as the local VM lab engine

## Goal

Build a private 30-lab DevOps bootcamp that uses Yeast to create local VM labs.

This bootcamp is for learning DevOps. It is not the public Yeast docs.

## Hard Separation

There are two separate tracks:

| Track | Count | Purpose | Audience | Style | Location |
|---|---:|---|---|---|---|
| Yeast Mini Bootcamp | 7 labs | Learn Yeast itself | Public docs users | Clean docs tutorial | Public docs |
| DevOps Bootcamp With Yeast | 30 labs | Learn DevOps using Yeast | Private first | Detailed blog/book/course | `_private-devops-bootcamp/` |

The DevOps bootcamp must not be forced into the 7-lab Yeast docs shape.

The public Yeast docs labs teach Yeast.

The private DevOps bootcamp teaches DevOps using Yeast.

## Private Folder Rule

The bootcamp starts in:

```text
_private-devops-bootcamp/
```

This folder is ignored by Git until tested and approved.

Nothing from this private course should be pushed accidentally.

## Writing Style

Use long-form practical teaching style:

- detailed
- blog/book-like
- not academic
- scenario-driven
- explains why each concept matters
- explains how this appears in real work
- includes troubleshooting mindset
- includes validation and failure drills

Example tone:

```text
In real operations, a process existing is not the same as a service being ready.
You verify the service, the port, the logs, and the client path before calling a deployment healthy.
```

## Learning Flow

The bootcamp follows:

```text
systems
-> networking
-> automation
-> containers
-> CI/CD
-> observability
-> reliability
-> IaC
-> GitOps
-> Kubernetes
-> AI/local LLM operations
```

## Target Private Tree

```text
_private-devops-bootcamp/
  README.md
  CURRICULUM.md

  01-linux-server-baseline/
  02-static-site-nginx/
  03-reverse-proxy-backend-app/
  04-database-backed-app/
  05-linux-troubleshooting-drill/
  06-bash-automation-server-setup/
  07-ansible-one-server/
  08-ansible-multi-vm-web-cluster/
  09-secrets-and-configuration-management/
  10-docker-fundamentals-vm/
  11-compose-multi-service-app/
  12-container-build-scan-hardening/
  13-ci-github-actions/
  14-self-hosted-ci-runner/
  15-private-container-registry/
  16-progressive-delivery-rollback/
  17-prometheus-grafana-monitoring/
  18-centralized-logging/
  19-opentelemetry-distributed-tracing/
  20-sre-slos-alerts-error-budgets/
  21-backup-restore-drill/
  22-chaos-failure-recovery-drill/
  23-terraform-fundamentals/
  24-terraform-modules-environments/
  25-gitops-argocd-or-flux/
  26-end-to-end-delivery-platform/
  27-kubernetes-k3s-foundations/
  28-kubernetes-networking-config-storage/
  29-kubernetes-delivery-capstone/
  30-ai-assisted-devops-local-llm-ops/
```

## Per-Lab Deliverables

Each lab should produce:

```text
README.md
BLOG.md
yeast.yaml
architecture.md
files/
scripts/
  validate.sh
checklist.md
troubleshooting.md
```

Purpose of each file:

| File | Purpose |
|---|---|
| `README.md` | Practical lab instructions |
| `BLOG.md` | Long-form teaching version |
| `yeast.yaml` | VM lab definition |
| `architecture.md` | Diagram and system explanation |
| `files/` | Configs, apps, static files, service files |
| `scripts/validate.sh` | Repeatable validation |
| `checklist.md` | Completion checklist |
| `troubleshooting.md` | Known failures and fixes |

## Lab 01: Linux Server Baseline

Goal:

Build one clean server and make it operational.

Teaches:

- users
- SSH keys
- packages
- hostname
- timezone
- updates
- firewall basics
- service checks

Real-world scenario:

You joined a team and received access to a Linux server. Before deploying anything, you need to inspect it and make it operationally safe.

## Lab 02: Static Site With Nginx

Goal:

Deploy a simple static website on one VM.

Teaches:

- Nginx
- ports
- systemd
- web server logs
- basic troubleshooting

Real-world scenario:

A team needs a small internal page online. You must install the service, place files, verify HTTP, and debug failures.

## Lab 03: Reverse Proxy To Backend App

Goal:

Use two VMs: one reverse proxy and one backend app.

Teaches:

- traffic flow
- reverse proxy configuration
- upstreams
- internal networking
- service separation

## Lab 04: Database-Backed App

Goal:

Run an app with PostgreSQL or MariaDB on another VM.

Teaches:

- persistence
- service dependencies
- credentials
- app-to-DB connectivity
- DB troubleshooting

## Lab 05: Linux Troubleshooting Drill

Goal:

Break a Linux service in controlled ways and recover it.

Teaches:

- `journalctl`
- `systemctl`
- `ss`
- `df`
- `du`
- `ps`
- permissions
- failed package installs
- logs-first debugging

Why it exists:

Real DevOps engineers are paid to troubleshoot. This lab turns "I followed steps" into "I can debug."

## Lab 06: Bash Automation For Server Setup

Goal:

Automate earlier manual server setup with shell scripts.

Teaches:

- scripting for ops
- repeatability
- validation
- safe shell habits
- why manual work does not scale

## Lab 07: Ansible For One Server

Goal:

Move from Bash to Ansible for one server.

Teaches:

- inventory
- modules
- tasks
- templates
- handlers
- idempotent configuration

## Lab 08: Ansible For Multi-VM Web Cluster

Goal:

Manage a load balancer and multiple web nodes with Ansible.

Teaches:

- multi-host orchestration
- templating from host data
- basic high-availability thinking
- repeatable cluster changes

## Lab 09: Secrets And Configuration Management

Goal:

Separate config from secrets and avoid hardcoded credentials.

Teaches:

- `.env` files
- secret exposure risk
- Ansible Vault or SOPS concepts
- least exposure
- config separation

## Lab 10: Docker Fundamentals On A VM

Goal:

Install Docker and learn container basics.

Teaches:

- images
- containers
- Docker networks
- bind mounts
- named volumes
- logs
- `docker exec`
- restart policies

## Lab 11: Containerized Multi-Service App With Compose

Goal:

Deploy app, database, and optional reverse proxy with Docker Compose.

Teaches:

- Compose services
- service discovery
- env files
- persistent data
- redeploy flow

## Lab 12: Container Build, Scan, And Hardening

Goal:

Build safer container images.

Teaches:

- Dockerfile basics
- small images
- non-root containers
- image tags
- vulnerability scanning
- SBOM concept
- why `latest` is dangerous

## Lab 13: CI With GitHub Actions

Goal:

Create a pipeline to lint, test, build, and deploy to a lab.

Teaches:

- pipeline as code
- CI basics
- secrets handling
- build/test/deploy stages

## Lab 14: Self-Hosted CI Runner

Goal:

Host your own runner or CI service in the Yeast lab.

Teaches:

- agents
- execution environments
- self-hosted automation
- infrastructure-backed CI

## Lab 15: Private Container Registry Workflow

Goal:

Build, tag, push, and deploy images from a private registry.

Teaches:

- image promotion
- rollback by tag
- release flow
- registry authentication

## Lab 16: Progressive Delivery And Rollback

Goal:

Learn safer release patterns.

Teaches:

- release versions
- blue/green concept
- canary concept
- health checks
- rollback by tag
- deployment verification

## Lab 17: Monitoring With Prometheus And Grafana

Goal:

Monitor VMs, containers, and services.

Teaches:

- exporters
- scraping
- dashboards
- metrics
- defining healthy systems

## Lab 18: Centralized Logging

Goal:

Collect and query logs from multiple services.

Teaches:

- Loki or OpenSearch-style logging
- log collection
- querying
- correlation
- incident debugging

## Lab 19: OpenTelemetry And Distributed Tracing

Goal:

Trace requests across services.

Teaches:

- traces
- spans
- request flow
- metrics/logs/traces differences
- distributed debugging

## Lab 20: SRE Basics: SLOs, Alerts, And Error Budgets

Goal:

Move from "dashboard exists" to reliability thinking.

Teaches:

- SLIs
- SLOs
- alert quality
- alert fatigue
- error budgets
- MTTR
- incident notes

## Lab 21: Backup And Restore Drill

Goal:

Back up app data, configs, and databases, then restore onto fresh infrastructure.

Teaches:

- backup strategy
- restore validation
- recovery thinking
- data safety

## Lab 22: Chaos And Failure Recovery Drill

Goal:

Intentionally break the platform and recover it.

Teaches:

- resilience
- recovery runbooks
- controlled failure
- incident practice

Failure ideas:

- kill backend
- corrupt config
- fill disk
- stop DB
- remove container
- restore service

## Lab 23: Terraform Fundamentals

Goal:

Introduce declarative infrastructure thinking.

Teaches:

- plan
- apply
- state
- variables
- outputs
- modules

## Lab 24: Terraform Modules And Environments

Goal:

Make Terraform reusable and environment-aware.

Teaches:

- reusable modules
- dev/stage/prod layout
- state separation
- variables
- outputs
- drift concept

## Lab 25: GitOps With Argo CD Or Flux

Goal:

Use Git as the source of truth for deployment state.

Teaches:

- desired state
- reconciliation
- app manifests
- rollback through Git
- drift correction

## Lab 26: End-To-End Delivery Platform On VMs

Goal:

Combine app, DB, reverse proxy, CI, registry, and observability into one mini production-style platform.

This is the major VM-platform integration checkpoint before Kubernetes.

## Lab 27: Kubernetes Foundations With k3s

Goal:

Create a small Yeast-based Kubernetes cluster and deploy a familiar app.

Teaches:

- pods
- deployments
- services
- `kubectl`
- cluster basics

## Lab 28: Kubernetes Networking, Config, And Storage

Goal:

Add real platform concepts on top of the cluster.

Teaches:

- ingress
- ConfigMaps
- Secrets
- namespaces
- persistent volumes

## Lab 29: Kubernetes Delivery Capstone

Goal:

Do the full flow on Kubernetes.

Teaches:

- source repo
- pipeline
- image build
- registry
- deploy to Kubernetes
- observe
- roll out a new version
- recover from failure

## Lab 30: AI-Assisted DevOps And Local LLM Ops

Goal:

Use AI safely inside DevOps workflows and run a local LLM service.

Teaches:

- local LLM service basics
- AI-assisted log summarization
- AI-generated infrastructure review
- secret safety
- privacy versus cloud AI
- human approval before infra changes

Possible stack:

- Ollama or llama.cpp
- optional Open WebUI
- small model
- logs from previous labs
- Yeast lab VMs
- validation scripts

Scenario:

An incident happens. You collect logs and metrics, ask a local model to summarize possible causes, then verify manually. AI helps investigation, but does not blindly change infrastructure.

## Standard Lab Template

```markdown
# Lab NN: Title

## Real-World Scenario

Explain the workplace-style situation.

## What You Are Building

Describe the lab architecture.

## Why This Matters

Explain the practical DevOps reason.

## Concepts You Will Learn

List concepts.

## Architecture

Show a diagram.

## Files In This Lab

Explain each file.

## Step 1: Read The Config

Explain `yeast.yaml` deeply.

## Step 2: Start The Environment

Explain what Yeast does.

## Step 3: Verify The Foundation

Check OS, network, services.

## Step 4: Verify The Application Layer

Check the actual service.

## Step 5: Break Something On Purpose

Teach debugging.

## Step 6: Fix It

Explain the reasoning.

## Step 7: Snapshot Or Reset

Explain why reset points matter.

## Step 8: Clean Up

Stop or destroy the lab.

## Real-World Reflection

Connect the lab to real work.

## What You Learned

Recap.

## Next Lab
```

## Acceptance Criteria

The private bootcamp is successful when:

- every lab has a `yeast.yaml`
- every lab has a validation script
- every lab has troubleshooting notes
- labs can be tested on a real Linux/KVM host
- each lab teaches one main concept
- each lab includes a real-world scenario
- failures are intentional and recoverable
- nothing is pushed publicly until approved
