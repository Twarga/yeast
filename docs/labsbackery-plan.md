# LabsBackery Planning Notes

Status: Draft

Owner: Twarga / TwargaOps

Purpose: define how old LabsBackery work should inform the new LabsBackery product without pulling old code directly into Yeast.

## Position

LabsBackery should be the cybersecurity lab product. Yeast should remain the infrastructure engine.

The old LabsBackery code is useful as reference material only. It can show the original product idea, lab model, UI expectations, terminal workflow, and reset workflow. It should not become the foundation for the new engine, and it should not force Yeast into old architecture decisions.

## Repository Setup

Keep the projects separate:

```text
~/Projects/yeast
~/Projects/labsbackery-old
~/Projects/labsbackery
```

Use `labsbackery-old` for inspection only. Use `labsbackery` later for the rebuilt product.

Do not copy old code into Yeast.

Do not add LabsBackery-specific commands to Yeast unless the same primitive is useful to normal Yeast users and automation.

## What To Extract From Old LabsBackery

The inspection should answer these questions:

- What was a lab?
- What was a machine?
- What was a challenge or exercise?
- How did users start a lab?
- How did users reset a lab?
- How did terminals connect to machines?
- What information did the UI need from the backend?
- What state did the backend track?
- Which workflows felt right?
- Which workflows were too complex?
- Which parts can be replaced by Yeast templates, snapshots, networking, and guest control?

## What To Ignore From Old LabsBackery

Ignore or delay:

- outdated dependency choices
- old UI styling
- old auth model
- old deployment model
- direct VM management code
- unfinished abstractions
- code that duplicates Yeast runtime behavior
- cloud hosting decisions
- billing, teams, RBAC, and enterprise scope

## New Product Shape

LabsBackery should eventually provide:

- a lab catalog
- lab detail pages
- start lab
- stop lab
- reset lab
- destroy lab
- browser terminal
- lab progress/status
- instructor/admin lab creation later

Yeast should provide the underlying primitives:

- template init
- multi-VM up/down/destroy
- provisioning
- baseline snapshots
- restore/reset
- status JSON
- guest inspect
- guest logs
- guest exec/copy where needed

## First Integration Target

The first real LabsBackery target should be small:

```text
one vulnerable target VM
one attacker VM
one private lab network
one clean baseline snapshot
one browser terminal per VM
one reset button
one JSON status view
```

This is enough to prove the product without building cloud, teams, payments, or advanced course logic.

## Required Yeast Foundation

Before LabsBackery depends on Yeast, these Yeast pieces should be stable:

- v0.8 JSON envelope and schemas
- stable error codes
- lifecycle events for long-running actions
- template metadata
- status output for multi-VM labs
- snapshot baseline workflow
- reset/restore behavior
- guest terminal connection info
- documented integration contract

## LabsBackery Integration Contract Draft

LabsBackery should treat Yeast as a command engine first.

Expected operations:

```text
yeast init --template <lab-template> --json
yeast up --json
yeast status --json
yeast snapshots <instance> --json
yeast snapshot <instance> clean --json
yeast restore <instance> clean --json
yeast down --json
yeast destroy --json
yeast inspect <instance> --json
yeast logs <instance> --json
```

Later, LabsBackery may use event streaming:

```text
yeast up --events --json
yeast restore <instance> clean --events --json
```

The UI should never scrape human terminal output.

## Data Model Draft

LabsBackery-owned concepts:

- Lab
- LabTemplate
- LabSession
- User
- TerminalSession
- Exercise
- Step or Task

Yeast-owned concepts:

- Project
- Instance
- Image
- Disk
- Network
- Snapshot
- Provisioning state
- Runtime state

The boundary is important. LabsBackery owns learning/product state. Yeast owns VM state.

## First Planning Tasks

1. Download or clone old LabsBackery into `~/Projects/labsbackery-old`.
2. Create an inventory document of useful old concepts.
3. Map old concepts to new Yeast primitives.
4. Define the first lab template.
5. Define the minimal LabsBackery backend API.
6. Define the terminal strategy.
7. Define the reset strategy.
8. Build one static prototype or wireframe.
9. After Yeast v0.8, write the official integration contract.
10. After the contract, rebuild LabsBackery as a clean consumer of Yeast.

## Decision Rules

- If old LabsBackery duplicates Yeast, Yeast wins.
- If LabsBackery needs a generic VM primitive, add it to Yeast only when it helps normal Yeast users too.
- If LabsBackery needs product state, keep it in LabsBackery.
- If a feature requires stable JSON/events, defer it until v0.8 is done.
- If a feature requires cloud hosting, defer it until after the local lab product works.

## Near-Term Recommendation

Finish Yeast v0.8 before serious LabsBackery implementation.

In parallel, inspect old LabsBackery and extract product knowledge. The output should be notes and diagrams, not code migration.
