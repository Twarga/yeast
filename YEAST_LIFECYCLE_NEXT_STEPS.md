# Yeast Lifecycle Next Steps

This file applies **The Founder's Product Lifecycle** to Yeast specifically. It shows where Yeast is now, what files we already have, and what to create next before coding the v2 architecture.

## Current Position

Yeast is past the raw idea stage.

The product vision is now clear enough:

> Yeast is the Linux-first local VM engine for TwargaOps. It starts as a modern Vagrant alternative, but its deeper purpose is to power LabsBackery, Yeast MCP, and later Twarga Cloud.

The main product roadmap already exists:

```text
YEAST_PRODUCT_ROADMAP.md
```

That file answers:

- what Yeast is
- who it is for
- why it matters
- what versions should exist
- what features belong in each version
- how Yeast supports LabsBackery, Yeast MCP, and Twarga Cloud

## Yeast Through The Product Lifecycle

| Phase | Yeast Status | Output File |
|---|---|---|
| 1. Vision | Mostly clear, included in roadmap | optional `YEAST_VISION.md` |
| 2. Discovery | Partially clear, based on your own pain and Vagrant/LabsBackery needs | optional `YEAST_DISCOVERY.md` |
| 3. Strategy | Clear enough for now | optional `YEAST_PRODUCT_STRATEGY.md` |
| 4. Roadmap | Done as Draft v1 | `YEAST_PRODUCT_ROADMAP.md` |
| 5. Technical Discovery | Next step | `YEAST_TECHNICAL_DISCOVERY.md` |
| 6. Architecture | After technical discovery | `YEAST_TECHNICAL_ARCHITECTURE.md` |
| 7. PRD / Requirements | After architecture, scoped to v0.1/v0.2 | `YEAST_V0_1_PRD.md` |
| 8. Engineering Planning | After PRD | `YEAST_V2_IMPLEMENTATION_PLAN.md` |
| 9. Implementation | Only after architecture and plan | code |
| 10. Testing / QA | Before release | `YEAST_TEST_PLAN.md` |
| 11. Documentation | Before release | `YEAST_DOCS_PLAN.md` |
| 12. Release | Before public version | `YEAST_RELEASE_PLAN.md` |
| 13. Feedback | After users try it | `YEAST_FEEDBACK_LOG.md` |
| 14. Iteration | After feedback | updated roadmap and plans |

## What We Should Do Next

The next correct phase is **Technical Discovery**.

Do not jump directly into coding.

Do not write final architecture yet.

First, answer the technical unknowns that can affect the architecture.

Create:

```text
YEAST_TECHNICAL_DISCOVERY.md
```

This file should answer:

- Should Yeast use direct QEMU CLI or libvirt?
- How should base images and qcow2 overlays work?
- What snapshot model is safest?
- How should cloud-init and post-boot provisioning cooperate?
- How should Yeast know when a VM is ready?
- How should private VM-to-VM networking work?
- How should project-safe state be stored?
- What does LabsBackery need from Yeast?
- What does Yeast MCP need from Yeast?
- What security risks matter before Twarga Cloud?

## After Technical Discovery

Once technical discovery is done, create:

```text
YEAST_TECHNICAL_ARCHITECTURE.md
```

That file should define:

- final module boundaries
- folder structure
- project model
- config model
- state model
- runtime model
- provisioning model
- networking model
- snapshot model
- guest control model
- JSON/event model
- diagrams

## After Architecture

Then create:

```text
YEAST_V2_IMPLEMENTATION_PLAN.md
```

That file should break the rebuild into milestones and tasks.

The likely order:

1. project identity and storage layout
2. config model
3. state model
4. runtime abstraction
5. QEMU lifecycle
6. image cache
7. cloud-init
8. SSH readiness
9. human/JSON output
10. tests and docs

## The Main Rule

Current Yeast code is a prototype/reference.

Use it to understand:

- how QEMU starts
- how cloud-init works
- how state is saved
- what commands already exist

But do not blindly build on top of messy code if you do not understand it.

The professional path is:

```text
Roadmap -> Technical Discovery -> Architecture -> Implementation Plan -> Code
```

## Immediate Next Action

Create and fill:

```text
YEAST_TECHNICAL_DISCOVERY.md
```

That is the next file before any serious v2 rebuild.
