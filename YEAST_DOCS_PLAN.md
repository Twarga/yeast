# Yeast Documentation Plan

Status: Draft v1  
Owner: Twarga / TwargaOps  
Phase: 11 - Documentation  
Purpose: Define the documentation Yeast needs before v0.1 and future releases

## 1. Purpose

Documentation is part of the Yeast product.

For a developer/infrastructure tool, users often decide whether to trust the project before they run it. They read the README, scan the quickstart, check install requirements, and decide if the tool feels serious.

This document defines the documentation system Yeast needs so users can:

- understand what Yeast is
- know if Yeast is for them
- install it
- run the first VM
- troubleshoot common problems
- understand the config
- understand limitations
- later use provisioning, snapshots, networking, and templates
- later integrate LabsBackery or Yeast MCP

The standard is:

> A user should be able to complete the first successful VM workflow without asking the founder for help.

## 2. Documentation Principles

### Principle 1: First Success Comes First

The first docs priority is not explaining every feature.

The first priority is helping a user reach:

```text
yeast init -> yeast pull -> yeast up -> yeast ssh
```

### Principle 2: Be Honest About Scope

Early Yeast should clearly say what works and what does not.

Users forgive early limitations. They do not forgive hidden limitations.

### Principle 3: Docs Should Teach The Mental Model

Yeast should be simple, but not mysterious.

Docs should explain:

- project
- config
- image cache
- instance disk
- cloud-init
- state
- QEMU/KVM
- SSH access

### Principle 4: Troubleshooting Is A Product Feature

VM tools fail because the host environment is complex.

Good troubleshooting docs are part of Yeast's value.

### Principle 5: Docs Must Match Reality

Every release should test the quickstart exactly as written.

If docs are wrong, the product is not release-ready.

## 3. Documentation Audiences

### Beginner User

This user wants to run Yeast for the first time.

They need:

- what Yeast is
- system requirements
- install steps
- first project
- first VM
- SSH
- cleanup

### Linux Builder

This user wants to use Yeast for local dev/infrastructure work.

They need:

- config reference
- image management
- disk behavior
- project state behavior
- logs
- JSON output

### Lab Builder

This user wants LabsBackery-style environments.

They need later:

- provisioning guide
- multi-VM guide
- networking guide
- snapshot/reset guide
- templates guide

### Integrator

This user is LabsBackery, Yeast MCP, scripts, or future tools.

They need:

- JSON schemas
- error codes
- command behavior
- event model
- integration examples

### Maintainer

This user is the founder or future contributor.

They need:

- architecture overview
- state model
- runtime model
- release process
- testing process
- contribution guide

## 4. Required Docs For v0.1

v0.1 docs should be small, clear, and focused.

Required:

```text
README.md
docs/quickstart.md
docs/installation.md
docs/config-reference.md
docs/troubleshooting.md
docs/known-limitations.md
docs/architecture-overview.md
```

Optional but useful:

```text
examples/ubuntu-basic/README.md
```

## 5. README Plan

Purpose:

The README is the front door.

It should answer in under two minutes:

- what Yeast is
- who it is for
- why it exists
- how to try it
- what works today
- what is coming later

README structure:

```text
# Yeast

Short tagline

## What Is Yeast?

## Why Yeast?

## Current Status

## Requirements

## Quick Start

## Example yeast.yaml

## Commands

## Roadmap

## Known Limitations

## Documentation

## License
```

README should not be too long.

Detailed explanations belong in `docs/`.

## 6. Quickstart Plan

File:

```text
docs/quickstart.md
```

Purpose:

Get user from zero to first running VM.

Flow:

```text
install/check requirements
create project
init config
pull image
start VM
check status
ssh into VM
stop VM
destroy VM
```

Rules:

- one path only
- no advanced options
- no long theory
- every command should be copy-pasteable
- tested before release

Success at end:

User has SSH'd into a real VM.

## 7. Installation Docs Plan

File:

```text
docs/installation.md
```

Should include:

- supported OS
- required packages
- KVM check
- QEMU install
- qemu-img install
- genisoimage/cdrkit install
- SSH key requirement
- build from source
- binary install later
- install script later

Sections:

```text
Supported Platforms
Requirements
Ubuntu/Debian
Fedora
Arch
Build From Source
Verify Installation
Common Install Problems
```

## 8. Config Reference Plan

File:

```text
docs/config-reference.md
```

Purpose:

Explain every supported `yeast.yaml` field.

v0.1 fields:

- version
- instances
- name
- image
- memory
- cpus
- disk_size
- user
- sudo
- env
- user_data

For each field:

- type
- required or optional
- default
- valid values
- example
- notes

Important:

Fields planned for future versions should be marked clearly as future, not supported.

## 9. Troubleshooting Plan

File:

```text
docs/troubleshooting.md
```

Purpose:

Help users fix common VM problems.

Initial issues:

- QEMU not installed
- qemu-img not installed
- genisoimage not installed
- `/dev/kvm` missing
- permission denied on KVM
- SSH key missing
- image not pulled
- VM starts but SSH timeout
- stale state
- port conflict
- destroy did not remove what user expected

Each entry should include:

```text
Symptom
Likely cause
How to confirm
Fix
Related command
```

## 10. Known Limitations Plan

File:

```text
docs/known-limitations.md
```

v0.1 limitations:

- Linux only
- QEMU/KVM only
- Ubuntu images only initially
- no provisioning beyond basic cloud-init
- no snapshots/reset
- no private multi-VM networking
- no LabsBackery integration contract
- no Yeast MCP integration
- no Twarga Cloud worker mode

This file protects trust.

Users should know what Yeast does not do yet.

## 11. Architecture Overview Plan

File:

```text
docs/architecture-overview.md
```

Purpose:

Explain how Yeast works at a high level for contributors and advanced users.

Should include:

- config as desired state
- state as actual runtime reality
- project identity
- image cache
- instance disks
- QEMU runtime
- cloud-init seed ISO
- SSH readiness
- command flow
- storage layout

Do not make this as large as `YEAST_TECHNICAL_ARCHITECTURE.md`.

This should be readable by a contributor in 10 minutes.

## 12. Examples Plan

Examples should be real and runnable.

v0.1 example:

```text
examples/ubuntu-basic/
  yeast.yaml
  README.md
```

Future examples:

```text
examples/caddy-web/
examples/two-vm-lab/
examples/private-network/
examples/snapshot-reset/
```

Each example README should include:

- what it demonstrates
- commands to run
- expected result
- cleanup command

## 13. Future Provisioning Docs

For v0.3.

Files:

```text
docs/provisioning.md
examples/caddy-web/README.md
```

Should explain:

- packages
- files
- shell commands
- cloud-init vs post-boot provisioning
- rerunning provisioning
- debugging provisioning failures
- idempotency expectations

## 14. Future Snapshot Docs

For v0.4.

File:

```text
docs/snapshots.md
```

Should explain:

- what snapshots are
- when to snapshot
- restore behavior
- stopped VM requirement if applicable
- snapshot all
- restore all
- disk usage
- safety warnings

## 15. Future Networking Docs

For v0.5.

File:

```text
docs/networking.md
```

Should explain:

- management networking
- private lab networking
- static IPs
- VM-to-VM traffic
- bridge mode if supported
- host permissions
- troubleshooting network issues

## 16. Future Integrator Docs

For v0.8+ and LabsBackery/MCP.

Files:

```text
docs/json-schemas.md
docs/error-codes.md
docs/events.md
docs/integrations/labsbackery.md
docs/integrations/yeast-mcp.md
```

Should explain:

- command JSON outputs
- schema versions
- error code list
- lifecycle events
- safe command behavior
- integration examples

## 17. Docs Release Checklist

Before v0.1 release:

- README is current
- quickstart works exactly as written
- installation docs are tested
- config reference matches code
- troubleshooting covers known blockers
- known limitations are honest
- architecture overview is not stale
- example project works

If quickstart fails, release fails.

## 18. Documentation Style

Use direct language.

Prefer:

```text
Yeast stores VM runtime files under ~/.yeast/projects/<project-id>.
```

Avoid:

```text
The system may internally provision persistent runtime artifacts in an implementation-defined location.
```

Docs should sound like a serious engineer explaining clearly.

Not marketing fluff.

Not corporate.

Not vague.

## 19. Documentation Maintenance

Docs must be updated when:

- config schema changes
- commands change
- JSON output changes
- state paths change
- install requirements change
- new release is published
- user reports confusion

Add feedback from users into:

```text
YEAST_FEEDBACK_LOG.md
```

Then update docs based on repeated confusion.

## 20. Final Docs Goal

The v0.1 documentation goal is:

> A Linux user can install Yeast, start one Ubuntu VM, SSH into it, stop it, and destroy it by following the docs without asking the founder for help.

That is the documentation bar for the first release.
