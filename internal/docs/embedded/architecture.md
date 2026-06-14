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

The job is to keep this local engine reliable while LabsBakery, Yeast MCP, and future cloud workers build on top of the documented command, config, JSON, and event contracts.
