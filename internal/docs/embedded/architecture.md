# Yeast Architecture Overview

Yeast is split into a small set of layers:

- CLI layer
- application workflows
- project metadata and paths
- config loader and validator
- state store and reconciliation
- image cache and manifest
- runtime abstraction
- cloud-init generation
- SSH readiness and guest access
- human and JSON output

Current runtime focus:

- Linux
- QEMU/KVM
- cloud-init
- SSH port forwarding

The v0.1 job is to make the lifecycle reliable before Yeast grows into provisioning, lab reset, networking, and AI control.
