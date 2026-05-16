# Yeast Technical Discovery

Status: Draft v1  
Owner: Twarga / TwargaOps  
Phase: 5 - Technical Discovery  
Purpose: Identify technical unknowns, risks, options, and experiments before designing Yeast v2 architecture

## 1. What This File Is

This file is not the final technical architecture.

This file is the research and decision-prep phase before architecture.

The goal is to answer:

> What technical things could break the Yeast roadmap if we design too early?

The product roadmap says what Yeast should become. Technical discovery asks whether the technical path is realistic, what tradeoffs exist, and what we need to test before committing to the v2 architecture.

The next file after this should be:

```text
YEAST_TECHNICAL_ARCHITECTURE.md
```

That file will define the final module boundaries, folder structure, models, and flows. This file prepares those decisions.

## 2. Discovery Summary

Yeast has several technical areas that determine the whole architecture:

- runtime backend
- image cache
- disk and overlay model
- cloud-init model
- provisioning model
- SSH readiness
- state storage
- project identity
- networking
- snapshots/reset
- guest control
- JSON/events
- security and isolation
- LabsBackery integration
- Yeast MCP integration
- future Twarga Cloud worker model

The biggest discovery conclusion is:

> Yeast v1 should be Linux-first, QEMU/KVM-first, CLI-first, project-based, and local-first.

Do not start with libvirt-first, cloud-first, daemon-first, or multi-provider-first unless a technical experiment proves direct QEMU is not enough.

## 3. Main Technical Unknowns

These are the questions that must be understood before architecture is finalized.

### Runtime Backend

Question:

Should Yeast control QEMU directly, or should it use libvirt?

Why it matters:

The runtime decision affects every other part of Yeast: process lifecycle, networking, snapshots, state, dependencies, install complexity, and future cloud worker design.

### Disk Model

Question:

How should Yeast store base images, project disks, overlays, and snapshots?

Why it matters:

Disk layout affects speed, safety, reset behavior, project isolation, and storage cleanup.

### Snapshot Model

Question:

Should Yeast use qcow2 internal snapshots, external overlays, full disk copies, or another model?

Why it matters:

Snapshots are core for LabsBackery. A bad snapshot model can corrupt disks, waste storage, or make reset unreliable.

### Cloud-init Model

Question:

How much should Yeast do through cloud-init, and how much should it do after SSH?

Why it matters:

Cloud-init is powerful for first boot, but not every provisioning task belongs there. Yeast needs a clean mental model.

### Provisioning Model

Question:

Should Yeast implement simple built-in provisioning only, or support external provisioners like Ansible later?

Why it matters:

Provisioning can easily become too broad. Yeast should support the common case without becoming a full configuration management system.

### Networking Model

Question:

What is the simplest reliable way to support management SSH plus private VM-to-VM lab networks?

Why it matters:

LabsBackery depends on realistic networking. Networking is also one of the easiest parts to make too complex.

### Guest Control Model

Question:

Should Yeast control guests only through SSH, or support other guest agents later?

Why it matters:

Yeast MCP and LabsBackery need exec, copy, logs, and inspect. SSH is simple and understandable, but a future agent could be more powerful.

### State Model

Question:

What should Yeast store in state, and what should it derive from disk/process reality?

Why it matters:

Bad state design creates lying status output and broken recovery.

### JSON/Event Model

Question:

How should Yeast expose machine-readable output without making CLI output ugly?

Why it matters:

LabsBackery and Yeast MCP should never scrape human terminal output.

## 4. Runtime Backend Discovery

### Option A: Direct QEMU CLI

Yeast directly starts `qemu-system-x86_64` with generated arguments.

Benefits:

- fewer moving parts
- easy to understand
- no libvirt dependency
- good fit for local Linux-first tool
- full control over QEMU command line
- easier to debug because the exact QEMU command is visible

Costs:

- Yeast owns process management
- Yeast owns networking setup complexity
- Yeast owns snapshot behavior
- Yeast must handle QEMU differences carefully
- advanced networking can become painful

Best for:

- v0.1 to v1.0 local engine
- simple install
- founder understanding
- direct control

### Option B: libvirt

Yeast talks to libvirt, and libvirt manages QEMU/KVM.

Benefits:

- mature VM lifecycle management
- better networking primitives
- better domain management
- common in Linux virtualization stacks
- useful for advanced host setups

Costs:

- heavier dependency
- more host setup
- libvirt permissions can be confusing
- less direct mental model
- harder for beginner install
- pushes Yeast closer to existing VM manager territory

Best for:

- future advanced backend
- server/worker mode
- users already running libvirt

### Recommendation

Use **direct QEMU CLI first**.

Reason:

Yeast v1 should optimize for being understandable, Linux-first, and founder-owned. Direct QEMU gives more control and fewer dependencies. libvirt can become a future optional backend if direct QEMU becomes limiting.

Architecture implication:

Even if Yeast starts with direct QEMU, design a runtime boundary so libvirt can be added later without rewriting the application layer.

Discovery experiment:

- Start one VM through direct QEMU.
- Confirm SSH port forwarding works.
- Confirm clean stop works.
- Confirm logs are captured.
- Confirm state can reconcile dead process.
- Document the exact QEMU arguments needed.

## 5. Image Cache Discovery

### Product Need

Yeast should not download a base image per project. It should keep trusted base images in a shared cache.

Expected model:

```text
~/.yeast/cache/images/
  ubuntu-24.04/
    image.qcow2
    manifest.json
```

### Main Questions

- Should images be hardcoded in Yeast or loaded from a remote manifest?
- Should Yeast support only trusted official images in v1?
- Should users be able to add custom images?
- How should checksums be stored and verified?
- How should image updates work?

### Options

Option A: Built-in trusted manifest.

Benefits:

- simple
- safe
- predictable
- no remote manifest infrastructure needed

Costs:

- adding images requires a new Yeast release
- less flexible

Option B: Remote official manifest.

Benefits:

- update supported images without releasing Yeast
- easier template ecosystem later

Costs:

- needs hosting
- needs signing/trust model eventually
- more complexity

Option C: User-defined custom images.

Benefits:

- power users can use their own images
- useful for labs

Costs:

- security/trust responsibility shifts to user
- more validation complexity

### Recommendation

For v1:

- built-in trusted manifest
- optional custom local image path later
- remote manifest after v1 if needed

Discovery experiment:

- Verify Ubuntu 22.04 and 24.04 cloud images.
- Confirm qemu-img overlay creation works from the cached image.
- Measure first pull time and disk use.

## 6. Disk And Overlay Discovery

### Product Need

Yeast should keep base images clean and create per-instance writable disks.

Expected relationship:

```text
base image -> instance disk
```

The base image is shared. The instance disk belongs to a project instance.

### Main Questions

- Should instance disks be qcow2 overlays backed by base images?
- Should Yeast support resizing disks on creation?
- Should Yeast allow resizing existing disks?
- Where should disks live?
- How should disk cleanup work?

### Recommended Model

Use qcow2 overlays for instance disks.

Reason:

- saves disk space
- faster project creation
- keeps base image clean
- fits local VM workflows

Storage direction:

```text
~/.yeast/projects/<project-id>/instances/<instance-name>/disk.qcow2
```

Discovery experiments:

- Create overlay from Ubuntu base image.
- Boot overlay.
- Resize overlay at creation.
- Resize existing overlay.
- Destroy overlay safely.
- Confirm base image is unchanged.

## 7. Snapshot Discovery

### Product Need

LabsBackery requires reset. Reset requires snapshots or a similar baseline system.

### Options

Option A: qcow2 internal snapshots.

Benefits:

- built into qcow2
- conceptually simple
- easy to list snapshots

Costs:

- can become hard to manage at scale
- internal snapshots have operational caveats
- may be less transparent than external files

Option B: external overlay snapshots.

Benefits:

- explicit file structure
- easier to reason about for lab reset
- baseline can remain untouched while active layer changes
- fits copy-on-write mental model

Costs:

- overlay chains can become complex
- restore logic must be carefully designed
- cleanup must be correct

Option C: full disk copy.

Benefits:

- simple and safe conceptually
- easy restore

Costs:

- slow
- uses much more disk space
- bad for large labs

### Recommendation

Do not finalize snapshot model until experiments are run.

Likely direction:

- v0.4 first implementation can be conservative
- stop VM before snapshot/restore
- prefer explicit external snapshot/baseline model if manageable
- avoid clever live snapshot behavior early

Discovery experiments:

- Test qcow2 internal snapshot create/list/restore.
- Test external overlay baseline/active model.
- Measure restore speed.
- Test failure case: restore while VM is running.
- Decide safest model for LabsBackery.

## 8. Cloud-init Discovery

### Product Need

Cloud-init is how Yeast prepares the guest on first boot.

Current expected uses:

- hostname
- user creation
- SSH key injection
- sudo policy
- environment variables
- package installation
- simple files
- first boot commands

### Main Questions

- Should custom user-data replace generated user-data or merge with it?
- Should Yeast support user-provided cloud-init snippets?
- How should Yeast expose generated cloud-init for debugging?
- How does Yeast know cloud-init finished?

### Current Risk

If user-provided `user_data` fully replaces generated cloud-init, users can accidentally remove SSH access.

This is powerful but dangerous.

### Recommended Direction

Support two modes:

1. Generated mode:
   Yeast generates cloud-init from structured config.

2. Advanced override mode:
   User provides full custom user-data and accepts responsibility.

Later, support merge/composition if needed.

Discovery experiments:

- Boot Ubuntu with generated user and SSH key.
- Install package through cloud-init.
- Write file through cloud-init.
- Run command through cloud-init.
- Detect whether cloud-init finished.
- Test failed cloud-init and inspect logs.

## 9. Provisioning Discovery

### Product Need

Provisioning turns blank machines into useful machines.

Yeast should support common provisioning without becoming Ansible.

### Provisioning Phases

Phase 1: first boot provisioning.

Best for:

- user
- SSH key
- hostname
- packages
- simple files
- early boot setup

Phase 2: post-boot provisioning.

Best for:

- copying local project files
- running scripts
- restarting services
- verifying service health

### Main Questions

- What belongs in cloud-init?
- What belongs in post-boot SSH provisioning?
- How does Yeast track provisioning status?
- How does Yeast retry failed provisioning?
- Should provisioning be idempotent?

### Recommendation

Start with simple built-in provisioners:

- packages
- files
- shell

Do not start with Ansible.

Later:

- add Ansible provisioner if real users need it

Discovery experiments:

- Install Caddy through cloud-init.
- Copy local app files after SSH.
- Run shell command after SSH.
- Verify Caddy responds.
- Re-run provisioning and observe behavior.

## 10. SSH Readiness Discovery

### Product Need

Yeast should not say a VM is ready just because QEMU started.

A VM is ready when Yeast can control it.

### Main Questions

- Is SSH reachability enough?
- Should Yeast wait for cloud-init completion?
- What timeout is reasonable?
- How should readiness failures be shown?
- Should Yeast retry with new SSH ports if port forwarding fails?

### Recommendation

Readiness should have levels:

1. Process running.
2. SSH reachable.
3. Cloud-init complete.
4. Provisioning complete.
5. Ready.

This makes status more honest.

Discovery experiments:

- Measure Ubuntu cloud-init time.
- Test SSH reachable before cloud-init finishes.
- Test failed cloud-init.
- Test port already used.
- Test VM boots but SSH never starts.

## 11. State Model Discovery

### Product Need

State records actual runtime reality.

It should not become a second config file.

### Main Questions

- What belongs in state?
- What belongs only in config?
- Should state be JSON, SQLite, or another format?
- How should state migrations work?
- How should Yeast handle stale state?

### Options

Option A: JSON state file.

Benefits:

- simple
- inspectable
- easy to edit in emergencies
- good for local project tool

Costs:

- migrations need care
- concurrent writes need locking
- querying is limited

Option B: SQLite.

Benefits:

- stronger query/update model
- good for complex future state
- transactional

Costs:

- heavier
- less transparent for beginners
- maybe unnecessary for v1

### Recommendation

Use JSON state for v1.

Reason:

Yeast is local-first and project-based. JSON is enough if locking and migrations are handled carefully.

State should include:

- schema version
- project ID
- instance runtime status
- PID
- SSH port
- management IP
- runtime paths
- provisioning status
- snapshot metadata later
- last error maybe

State should not include:

- desired memory
- desired CPUs
- desired image
- desired provisioning config

Those belong in config.

Discovery experiments:

- Design state v2 example.
- Simulate stale PID.
- Simulate corrupted state file.
- Simulate migration from v1 to v2.

## 12. Project Identity Discovery

### Product Need

Two different projects should be able to have an instance named `web` without conflict.

### Main Questions

- How should project ID be generated?
- Should project ID depend on absolute path?
- Should project ID be stored in a project file?
- What happens if project folder moves?

### Options

Option A: hash absolute project path.

Benefits:

- automatic
- no extra file needed

Costs:

- moving folder changes identity

Option B: store project ID in hidden file.

Benefits:

- stable if folder moves
- explicit identity

Costs:

- creates another project file
- users may delete it

Option C: derive from config name plus path.

Benefits:

- more human-readable

Costs:

- collisions still possible if not careful

### Recommendation

Use stored project ID.

Possible file:

```text
.yeast/project.json
```

Reason:

Labs and projects may move. Stable identity is better than path hash only.

Discovery experiments:

- Create project ID on init.
- Move project folder.
- Confirm runtime path still resolves.
- Decide whether `.yeast/` should be committed or ignored.

## 13. Networking Discovery

### Product Need

Yeast needs two networking ideas:

1. Management access:
   Yeast can SSH/control the VM.

2. Lab network:
   VMs can talk to each other for labs.

### Current Simple Model

QEMU user networking with host port forwarding is good for management SSH.

### Future Lab Need

LabsBackery needs private VM-to-VM networks with predictable IPs.

### Main Questions

- Can direct QEMU create private networks without libvirt?
- Should Yeast use QEMU socket networking, tap devices, bridges, or user-mode networking?
- How much host network setup should Yeast automate?
- How should static IPs be assigned inside guests?
- Should private networks require root or special permissions?

### Options

Option A: QEMU user networking only.

Benefits:

- simple
- no root
- easy SSH forwarding

Costs:

- weak for VM-to-VM lab networks
- not enough for realistic labs

Option B: tap/bridge networks.

Benefits:

- realistic networking
- VM-to-VM communication
- good for labs

Costs:

- host setup complexity
- permissions
- cleanup risks

Option C: libvirt networks.

Benefits:

- mature network management
- easier private networks if libvirt is installed

Costs:

- libvirt dependency
- not ideal for simple install

### Recommendation

For v0.1/v0.2:

- use QEMU user networking for management SSH

For v0.5:

- research tap/bridge private networks
- consider optional libvirt backend only if direct QEMU networking is too painful

Discovery experiments:

- Start two VMs with user networking only and test limitations.
- Create tap/bridge private network manually.
- Test two VMs pinging each other.
- Test static IP through cloud-init/netplan.
- Document permissions required.

## 14. Guest Control Discovery

### Product Need

Yeast needs to run commands and copy files into VMs.

This powers:

- provisioning
- `yeast exec`
- `yeast copy`
- LabsBackery terminals
- Yeast MCP operations

### Options

Option A: SSH only.

Benefits:

- simple
- standard
- understandable
- no guest agent to install

Costs:

- depends on SSH readiness
- file copy needs scp/sftp
- not ideal for early boot control

Option B: custom guest agent.

Benefits:

- more control
- richer operations
- possibly faster structured communication

Costs:

- must install and maintain agent
- security risk
- more complex
- not needed for v1

### Recommendation

Use SSH-only for v1.

Add guest agent only much later if real limitations appear.

Discovery experiments:

- Run command over SSH.
- Capture stdout/stderr/exit code.
- Copy file upload.
- Copy file download.
- Handle command timeout.
- Handle SSH failure clearly.

## 15. JSON And Event Discovery

### Product Need

Yeast needs machine-readable output for LabsBackery and Yeast MCP.

### Main Questions

- Should every command support `--json`?
- Should long-running commands stream events?
- How should errors be structured?
- Should schema names be versioned?

### Recommendation

Use two output modes:

1. Human mode:
   pretty terminal output

2. JSON mode:
   structured object or event stream

Long-running commands like `up`, `provision`, `snapshot`, and `restore` should eventually emit events.

Example event names:

- project.loaded
- state.locked
- image.verified
- disk.created
- cloudinit.generated
- instance.started
- ssh.ready
- provision.started
- provision.finished
- state.saved

Discovery experiments:

- Define sample JSON response for status.
- Define sample JSON response for up.
- Define sample error shape.
- Decide whether event streaming is newline-delimited JSON.

## 16. Security And Isolation Discovery

### Product Need

Yeast runs real VMs. LabsBackery and Twarga Cloud will eventually run potentially dangerous labs.

Security must be considered early, even if not fully solved until cloud.

### Local Security Questions

- What host permissions does Yeast require?
- Can a malicious config run dangerous host commands?
- How are SSH keys handled?
- Are generated files safe permissions?
- Can instance names escape paths?
- Can provision file paths escape project boundaries?

### Cloud Security Questions

- How are users isolated from each other?
- Can a lab attack the host or another lab?
- Should outbound internet be restricted?
- How are browser terminals secured?
- How are lab files cleaned up?
- How are worker machines reset?

### Recommendation

For v1 local:

- validate names and paths strongly
- avoid shelling out with unsafe string interpolation
- keep SSH key handling explicit
- document trust assumptions

For Twarga Cloud later:

- treat as separate security architecture project
- do not reuse local assumptions blindly

Discovery experiments:

- Path traversal tests for instance names and file provisioning.
- Permission checks for generated SSH/cloud-init files.
- Review QEMU process isolation assumptions.
- List host commands Yeast executes.

## 17. LabsBackery Integration Discovery

### LabsBackery Needs From Yeast

LabsBackery will need Yeast to provide:

- start lab
- stop lab
- reset lab
- destroy lab
- list lab VMs
- show per-VM status
- expose SSH/web terminal info
- provision per VM
- snapshot baseline
- restore baseline
- report progress as events

### Main Question

Should LabsBackery call Yeast CLI, use Yeast as a library, or talk to a Yeast daemon/API?

### Options

Option A: call Yeast CLI.

Benefits:

- simple
- works early
- decoupled process boundary
- JSON output is enough

Costs:

- process overhead
- harder to stream rich events unless designed
- must parse stable output

Option B: use Yeast as Go library.

Benefits:

- direct integration
- no process spawn

Costs:

- LabsBackery is Python/FastAPI in old plan
- language boundary problem
- tighter coupling

Option C: Yeast daemon/API.

Benefits:

- clean web integration
- long-running event streams
- future cloud fit

Costs:

- much more complex
- security and lifecycle issues
- not needed early

### Recommendation

Start with Yeast CLI + JSON.

Reason:

It is simplest and forces Yeast to have stable automation output.

Later:

- add daemon/API only when LabsBackery proves real need

Discovery experiments:

- Mock LabsBackery calling `yeast status --json`.
- Mock LabsBackery calling `yeast up --json`.
- Test long-running progress output.
- Define integration contract.

## 18. Yeast MCP Discovery

### Yeast MCP Needs From Yeast

Yeast MCP needs safe, structured operations:

- list projects
- list instances
- status
- exec
- copy
- logs
- inspect
- snapshot
- restore

### Main Questions

- What operations are safe for an AI agent?
- Should MCP require user approval for destructive actions?
- How should command output be limited?
- How should secrets be protected?
- Should MCP expose raw shell exec or controlled commands first?

### Recommendation

Do not design MCP before Yeast guest control and JSON are stable.

For Yeast:

- make operations structured
- add clear destructive/non-destructive distinction
- return predictable results

Discovery experiments:

- Define MCP-safe command list.
- Define dangerous operations.
- Define approval requirements.
- Define output limits for logs and exec.

## 19. Twarga Cloud Worker Discovery

### Future Need

Twarga Cloud may run Yeast on Hetzner bare-metal servers as lab workers.

### Main Questions

- Can local Yeast architecture become a worker later?
- What must change for multi-user remote execution?
- How are users isolated?
- How are resources limited?
- How are labs cleaned up?
- How does billing connect to resource usage?

### Recommendation

Do not build cloud worker in v1.

But avoid architecture that blocks it.

Design principles:

- project identity should be explicit
- runtime paths should be isolated
- JSON/events should be stable
- cleanup should be reliable
- no hidden global assumptions

Discovery experiments later:

- Run Yeast on one Hetzner server.
- Start one lab remotely.
- Auto-destroy after timeout.
- Measure resource usage.
- Test isolation assumptions.

## 20. Recommended Technical Direction

Based on current discovery, Yeast v1 should choose:

- Language: Go
- Host support: Linux first
- Runtime backend: direct QEMU/KVM first
- Future backend: optional libvirt later
- Config: YAML
- State: JSON with schema version and locking
- Project identity: stored project ID
- Image cache: built-in trusted manifest first
- Disk model: qcow2 base image plus project overlay
- Provisioning: cloud-init plus post-boot SSH
- Guest control: SSH first
- Networking v0.1: user networking with SSH forwarding
- Networking later: private VM-to-VM lab network
- Snapshot model: undecided until experiments
- Output: human + JSON
- Events: versioned events for long-running commands
- LabsBackery integration: CLI + JSON first
- Yeast MCP integration: after guest control is stable
- Twarga Cloud: future worker model, not v1

## 21. Experiments To Run Before Architecture

### Experiment 1: QEMU Lifecycle

Goal:

Prove direct QEMU can support the basic v0.1 lifecycle cleanly.

Test:

- create disk
- boot VM
- forward SSH
- wait for SSH
- stop VM
- reconcile state after manual kill

### Experiment 2: Cloud-init Completion

Goal:

Understand when a VM is truly ready.

Test:

- user creation
- SSH key injection
- package install
- cloud-init completion signal
- failed cloud-init logs

### Experiment 3: Provisioning Caddy

Goal:

Prove Yeast can turn Ubuntu into a useful web VM.

Test:

- install Caddy
- copy index file
- start service
- verify HTTP response
- rerun provisioning

### Experiment 4: Snapshot Strategy

Goal:

Choose snapshot model.

Test:

- qcow2 internal snapshot
- external overlay model
- restore speed
- disk growth
- safety behavior

### Experiment 5: Private Lab Network

Goal:

Prove two VMs can communicate on a private network.

Test:

- create private network
- assign static IPs
- ping between VMs
- keep management SSH working
- document host permissions

### Experiment 6: CLI JSON Contract

Goal:

Prove LabsBackery can call Yeast cleanly.

Test:

- `status --json`
- `up --json`
- error JSON
- progress events
- mock LabsBackery parser

## 22. Decisions Required Before Architecture

These decisions should be made before writing `YEAST_TECHNICAL_ARCHITECTURE.md`.

Decision 1:

Runtime backend for v1.

Recommended:

Direct QEMU/KVM.

Decision 2:

State storage for v1.

Recommended:

JSON state with schema version and lock file.

Decision 3:

Project identity model.

Recommended:

Stored project ID.

Decision 4:

Provisioning model.

Recommended:

Cloud-init plus post-boot SSH provisioners.

Decision 5:

LabsBackery integration style.

Recommended:

CLI plus stable JSON first.

Decision 6:

Snapshot model.

Recommended:

Run experiments before final decision.

Decision 7:

Private networking model.

Recommended:

Run experiments before final decision.

## 23. What Architecture Must Support

The future architecture must support:

- project-safe storage
- direct QEMU runtime implementation
- possible future libvirt runtime implementation
- clean config validation
- JSON state with migrations
- image cache and verification
- cloud-init generation
- post-boot SSH provisioning
- guest control through SSH
- event-based output
- LabsBackery calling Yeast through CLI/JSON
- future MCP safety boundaries
- future remote worker mode without full rewrite

## 24. Final Discovery Conclusion

Yeast v1 should not try to be a cloud, daemon, full hypervisor platform, or multi-provider manager.

Yeast v1 should be:

- local
- Linux-first
- QEMU/KVM-first
- project-based
- CLI-first
- JSON-friendly
- provisioning-capable
- snapshot-aware
- lab-ready

The next step is to turn these decisions into architecture.

Next file:

```text
YEAST_TECHNICAL_ARCHITECTURE.md
```
