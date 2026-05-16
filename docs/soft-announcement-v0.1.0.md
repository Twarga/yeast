# Yeast v0.1.0 Soft Announcement

Yeast v0.1.0 is out as an early prerelease.

Yeast is a Linux-first local VM engine for QEMU/KVM. The goal is simple: make real Linux VMs feel project-native, repeatable, and understandable from a terminal.

This first release focuses on the core lifecycle:

```text
init -> pull -> up -> status -> ssh -> down -> destroy
```

What works now:

- project-based `yeast.yaml`
- Ubuntu cloud image pull and checksum verification
- QEMU/KVM VM startup
- cloud-init bootstrap
- SSH readiness
- status tracking
- human terminal output with Lip Gloss styling
- JSON output for automation
- one-script Linux installer

What is not in this release yet:

- provisioning packages/files/shell workflows
- snapshots and reset
- private multi-VM lab networking
- guest `exec`, `copy`, `logs`, and `inspect`
- LabsBackery integration
- Yeast MCP
- Twarga Cloud workers

This release is intentionally small. It proves the local VM engine before Yeast grows into the full TwargaOps infrastructure layer for LabsBackery, AI-controlled VM workflows, and future hosted labs.

Install:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

Repository:

```text
https://github.com/Twarga/yeast
```

Use this if you are comfortable testing early Linux infrastructure tooling and reporting rough edges.
