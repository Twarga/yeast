# Yeast v2 Implementation Plan

Status: Draft v1  
Owner: Twarga / TwargaOps  
Phase: 8 - Engineering Planning  
Decision: Start Yeast v2 from zero, using current Yeast only as prototype/reference

## 1. Purpose

This document turns the Yeast vision, roadmap, technical discovery, and architecture into an execution plan.

The decision is:

> Yeast v2 will start from a clean implementation, not by continuing directly inside the current MVP structure.

This does not mean the current code was useless. The current code proved the core idea:

```text
config -> cloud-init -> disk -> QEMU -> SSH -> state
```

But v2 needs a cleaner foundation:

- project identity
- project-safe paths
- application workflows
- runtime abstraction
- structured state
- future provisioning
- future snapshots
- future networking
- stable JSON/events
- LabsBackery/MCP integration readiness

The goal of this implementation plan is to build v2 in controlled milestones so the project does not become messy again.

## 2. Rebuild Rule

The v2 rebuild should follow this rule:

```text
Use old code as a reference, not as the foundation.
```

Allowed from old code:

- command names and UX ideas
- image download/checksum logic after review
- size parsing utility after review
- file lock idea after review
- QEMU command knowledge after review
- cloud-init generation knowledge after review
- SSH readiness idea after review
- human/JSON output lessons after review

Not allowed blindly:

- copying command files that own too much workflow logic
- keeping instance paths based only on instance name
- keeping state without project identity
- adding provisioning into the old command flow
- adding snapshots before storage model is clear
- adding LabsBackery integration before JSON contract is stable

## 3. Implementation Philosophy

Build a boring engine first.

Do not build LabsBackery features directly inside Yeast.

Do not build Twarga Cloud.

Do not build Yeast MCP yet.

Do not build a daemon.

Build the local engine in this order:

```text
project -> config -> state -> image cache -> runtime -> cloud-init -> up/status/down/destroy -> output
```

Then add:

```text
provisioning -> snapshots -> networking -> guest control -> templates -> events
```

## 4. Target v2 Milestones

The v2 rebuild should be split into milestones:

- Milestone 0: repository reset and skeleton
- Milestone 1: project identity and paths
- Milestone 2: config model and validation
- Milestone 3: state store and locking
- Milestone 4: image cache and manifest
- Milestone 5: runtime abstraction and QEMU lifecycle
- Milestone 6: cloud-init and guest readiness
- Milestone 7: core commands v0.1
- Milestone 8: human and JSON output
- Milestone 9: tests and examples
- Milestone 10: docs and v0.1 release prep

After v0.1:

- Milestone 11: provisioning
- Milestone 12: snapshots/reset
- Milestone 13: private networking
- Milestone 14: guest control
- Milestone 15: LabsBackery integration contract

This document focuses mainly on the v2 foundation through v0.1, then outlines the next milestones.

## 5. Milestone 0: Repository Reset And Skeleton

Goal:

Create a clean v2 structure without carrying old command-flow mess forward.

Tasks:

- Create a v2 branch or clean directory plan.
- Decide whether to keep old code in history only or move it to a prototype folder temporarily.
- Create new folder structure from `YEAST_TECHNICAL_ARCHITECTURE.md`.
- Add minimal `cmd/yeast/main.go`.
- Add minimal root command.
- Add build/test tooling.
- Add README note that v2 rewrite is in progress.
- Keep old docs/planning files.

Definition of done:

- Project builds a minimal `yeast --help`.
- New internal folder structure exists.
- No old command workflow logic has been copied blindly.
- Old code remains available in git history/reference.

Risk:

Starting from zero can become an excuse to rewrite forever.

Control:

Milestone 0 must be short. It only creates the skeleton.

## 6. Milestone 1: Project Identity And Paths

Goal:

Make Yeast project-safe before building runtime behavior.

Why this comes first:

The old model can collide when two projects have the same instance name. v2 must fix this before adding more features.

Tasks:

- Implement project root resolution.
- Implement `.yeast/project.json`.
- Generate stable project ID.
- Load existing project ID.
- Create project runtime directory under `~/.yeast/projects/<project-id>/`.
- Define instance runtime path helper.
- Define cache path helper.
- Add path safety checks.
- Add tests for project path calculation.

Definition of done:

- A project can be initialized with stable ID.
- Moving the folder does not create a new project ID.
- Two projects can both have instance `web` without runtime path collision.
- Path helpers cannot escape Yeast runtime directories.

Commands affected:

- `yeast init`
- future all commands

Output:

- project package implemented
- path tests passing

## 7. Milestone 2: Config Model And Validation

Goal:

Define v2 desired-state model.

Tasks:

- Define config structs.
- Load `yeast.yaml`.
- Validate config version.
- Validate instance names.
- Validate images.
- Validate memory/CPU.
- Validate disk sizes.
- Validate user/sudo.
- Validate basic provisioning fields for future compatibility.
- Apply defaults.
- Add config tests.

Initial v2 config should support:

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

Fields planned but may be disabled until later:

- networks
- provision
- templates

Definition of done:

- Valid config loads with defaults.
- Invalid config fails before runtime starts.
- Error messages point to the bad field.
- Tests cover common invalid cases.

Important rule:

Do not overdesign full v1 config yet. Define enough for v0.1 while leaving clean extension points.

## 8. Milestone 3: State Store And Locking

Goal:

Create reliable actual-state storage.

Tasks:

- Define state schema version.
- Define instance runtime state.
- Implement load state.
- Implement save state atomically.
- Implement lock file.
- Implement stale lock handling.
- Implement state reconciliation interface.
- Add tests for load/save/lock/corrupt state.
- Add migration placeholder.

State v2 should include:

- schema
- project_id
- instances map
- status
- pid
- management_ip
- ssh_port
- runtime_dir
- provisioning_status
- last_error

Definition of done:

- State can be created for a new project.
- State can be loaded and saved.
- Concurrent command attempts are blocked safely.
- Corrupt state gives clear error.
- Stale locks can be handled safely.

Important rule:

State records runtime reality. It must not duplicate desired config fields unnecessarily.

## 9. Milestone 4: Image Cache And Manifest

Goal:

Support trusted base image pull and verification.

Tasks:

- Define image manifest model.
- Add built-in supported images.
- Implement image cache paths.
- Implement `yeast pull --list`.
- Implement `yeast pull <image>`.
- Implement checksum verification.
- Implement partial download cleanup.
- Add progress output later if needed.
- Add tests for manifest and checksum logic.

Initial supported images:

- ubuntu-22.04
- ubuntu-24.04

Definition of done:

- User can list supported images.
- User can pull an image.
- Existing image is verified.
- Bad checksum is detected.
- Cache path is project-independent.

Risk:

Image URLs and checksums can change.

Control:

Use pinned release URLs and documented checksum source.

## 10. Milestone 5: Runtime Abstraction And QEMU Lifecycle

Goal:

Make Yeast able to prepare and run a VM through a runtime boundary.

Tasks:

- Define runtime interface.
- Define machine/runtime plan model.
- Implement QEMU disk creation.
- Implement QEMU command construction.
- Implement QEMU start.
- Implement QEMU stop.
- Implement QEMU process release.
- Implement log file creation.
- Implement basic process inspection.
- Add tests for command construction where possible.

Runtime should support:

- prepare disk
- start instance
- stop instance
- inspect process
- destroy runtime files

Definition of done:

- Application layer can start a QEMU VM without knowing QEMU argument details.
- Runtime returns PID and runtime metadata.
- VM logs are written to predictable path.
- Stop works cleanly.

Important rule:

Runtime does not parse config. Application layer passes resolved machine plan.

## 11. Milestone 6: Cloud-init And Guest Readiness

Goal:

Make started VMs reachable and prepared for login.

Tasks:

- Implement SSH key discovery.
- Generate cloud-init user-data.
- Generate meta-data.
- Create seed ISO.
- Attach seed ISO in runtime plan.
- Implement SSH readiness check.
- Implement readiness levels.
- Add useful timeout errors.
- Add tests for generated user-data.

Readiness levels:

- process running
- SSH reachable
- cloud-init complete if supported
- ready

Definition of done:

- Ubuntu VM boots.
- User exists.
- SSH key works.
- Yeast waits until SSH is reachable.
- Failure to reach SSH gives clear error.

Risk:

Cloud-init may take longer than expected.

Control:

Status should distinguish process running from SSH ready.

## 12. Milestone 7: Core Commands v0.1

Goal:

Ship the core local VM lifecycle.

Commands:

- `yeast init`
- `yeast doctor`
- `yeast pull`
- `yeast up`
- `yeast status`
- `yeast ssh`
- `yeast down`
- `yeast destroy`

Tasks:

- Implement each command as thin CLI wrapper.
- Implement application workflows behind each command.
- Ensure state locking around mutating commands.
- Ensure status reconciles stale processes.
- Ensure destroy removes only project runtime files.
- Ensure SSH uses state info.
- Add clear errors.

Definition of done:

- User can initialize project.
- User can pull image.
- User can start VM.
- User can see status.
- User can SSH.
- User can stop VM.
- User can destroy VM.

This is the first real v2 success.

## 13. Milestone 8: Human And JSON Output

Goal:

Make Yeast useful for humans and tools without duplicating workflow logic.

Tasks:

- Define result types.
- Define error type with code/message/details.
- Implement human renderer.
- Implement JSON renderer.
- Add `--json`.
- Ensure major commands return structured output.
- Add tests for JSON shape.

Commands requiring JSON in v0.1:

- init
- doctor
- pull
- up
- status
- down
- destroy

Definition of done:

- Human output is readable.
- JSON output is parseable.
- Errors have stable codes.
- LabsBackery can theoretically call status/up/down without scraping text.

## 14. Milestone 9: Tests And Examples

Goal:

Prove v0.1 works and create examples users can follow.

Tasks:

- Add unit tests for config.
- Add unit tests for state.
- Add unit tests for paths.
- Add unit tests for image manifest.
- Add unit tests for output JSON.
- Add command construction tests.
- Add manual integration checklist.
- Create example `ubuntu-basic`.

Manual integration checklist:

- fresh project init
- image pull
- VM up
- status
- SSH
- down
- up again
- destroy
- status after destroy

Definition of done:

- Fast tests pass.
- Manual lifecycle checklist passes on real Linux/KVM host.
- Example project works.

## 15. Milestone 10: Docs And v0.1 Release Prep

Goal:

Make v0.1 understandable and releasable.

Tasks:

- Rewrite README around v2.
- Add quickstart.
- Add install guide.
- Add doctor troubleshooting.
- Add config reference for v0.1.
- Add architecture overview link.
- Add known limitations.
- Add release checklist.
- Tag v0.1.0 when ready.

Definition of done:

- New user can install and run first VM from docs.
- Known limitations are honest.
- Release notes explain what works and what does not.

## 16. Post-v0.1 Milestone 11: Provisioning

Goal:

Make VMs useful automatically.

v0.3 contract:

- top-level `provision` applies to every instance
- instance-level `provision` appends after top-level steps
- packages, files, and shell run post-boot over SSH
- cloud-init remains bootstrap for user, SSH key, hostname, sudo, and environment setup
- `yeast up` runs provisioning automatically after SSH readiness
- `yeast provision` reruns the same post-boot plan against an existing reachable VM
- package installation relies on package-manager idempotency
- file provisioning overwrites destination files
- shell commands always run and must be idempotent if reruns matter

Features:

- packages
- files
- shell
- provisioning status
- provisioning logs
- `yeast provision`

Definition of done:

- Caddy web VM demo works.
- User can rerun provisioning.
- Failure is visible and recoverable.

## 17. Post-v0.1 Milestone 12: Snapshots And Reset

Goal:

Support lab reset.

Features:

- snapshot
- restore
- list snapshots
- delete snapshot
- snapshot all
- restore all

Execution contract:

- `v0.4` snapshot create and restore only support stopped VMs
- restore must fail clearly for running targets
- project-wide snapshot/restore is sequential, not atomic
- metadata is tracked in state per instance
- first metadata fields are `name`, `created_at`, `description`, `disk_path`, and optional `source_disk_size`

Definition of done:

- VM can be provisioned, snapshotted, broken, restored, and verified.
- one stopped-instance snapshot/restore loop works reliably on a real host
- command scope is fixed before runtime helpers are built

Blocked by:

- snapshot technical experiment

## 18. Post-v0.1 Milestone 13: Private Networking

Goal:

Support multi-VM LabsBackery topologies.

Features:

- project networks
- private VM-to-VM network
- static IPs
- management vs lab network separation

Execution contract:

- `v0.5` supports one project-level private lab network in the first pass
- instances can attach to that network with one static IPv4
- management SSH remains on the current host-forwarded path and is not replaced by the lab network
- cloud-init renders the guest-side static lab address
- bridge mode, DHCP, and multiple private networks are explicitly out of scope for the first pass

Definition of done:

- attacker VM can reach target VM on private network.
- Yeast can still control both VMs.
- status exposes the configured lab IP clearly

## 19. Post-v0.1 Milestone 14: Guest Control

Goal:

Support automation and MCP primitives.

Features:

- `yeast exec`
- `yeast copy`
- `yeast logs`
- `yeast inspect`
- structured command result

Definition of done:

- Yeast can run a command in VM and return stdout/stderr/exit code.
- File copy works both directions.

## 20. Post-v0.1 Milestone 15: LabsBackery Contract

Goal:

Make Yeast usable as LabsBackery backend.

Tasks:

- Define CLI/JSON contract.
- Define lab lifecycle calls.
- Define status schema.
- Define reset behavior.
- Build one LabsBackery test lab.

Definition of done:

- LabsBackery can start, inspect, reset, and destroy one lab through Yeast.

## 21. What To Build First

The first implementation task after this plan is:

```text
Milestone 0: Repository Reset And Skeleton
```

Do not start with QEMU.

Do not start with provisioning.

Do not start with snapshots.

Start with the clean project structure.

Then:

```text
project identity
config
state
image cache
runtime
cloud-init
core commands
output
tests
docs
```

## 22. Definition Of Done For Yeast v2 v0.1

v0.1 is done when:

- clean v2 architecture exists
- `yeast init` works
- `yeast doctor` works
- `yeast pull ubuntu-24.04` works
- `yeast up` starts one Ubuntu VM
- `yeast status` is accurate
- `yeast ssh` connects
- `yeast down` stops the VM
- `yeast destroy` removes project runtime files
- JSON output exists for core commands
- docs explain the quickstart
- manual lifecycle checklist passes on Linux/KVM

v0.1 is not done when:

- only QEMU starts manually
- only the founder can use it
- docs are missing
- status lies
- project paths can collide
- JSON output is not stable enough for scripts

## 23. Work Rules During Rebuild

Rule 1:

One milestone at a time.

Rule 2:

Do not add features from future milestones unless required by current architecture.

Rule 3:

If the implementation contradicts the architecture, update the architecture or fix the implementation. Do not let them silently diverge.

Rule 4:

Every major command must have human output and JSON output from the start.

Rule 5:

Every destructive command must be path-safe.

Rule 6:

Every external command execution must use explicit arguments, not unsafe shell strings.

Rule 7:

Tests come with the layer they protect.

Rule 8:

Docs must be updated before release, not after.

## 24. Files To Keep Updated

During v2 implementation, keep these files alive:

```text
YEAST_VISION.md
YEAST_PRODUCT_ROADMAP.md
YEAST_TECHNICAL_DISCOVERY.md
YEAST_TECHNICAL_ARCHITECTURE.md
YEAST_V2_IMPLEMENTATION_PLAN.md
YEAST_TEST_PLAN.md
YEAST_RELEASE_PLAN.md
```

If implementation changes the architecture, update the architecture file.

If a milestone changes, update this implementation plan.

If users or tests reveal something new, update the roadmap or feedback log later.

## 25. Final Execution Sequence

The rebuild sequence is:

```text
0. Skeleton
1. Project identity
2. Config
3. State
4. Images
5. Runtime/QEMU
6. Cloud-init/readiness
7. Core commands
8. Output/JSON
9. Tests/examples
10. Docs/release
```

After that:

```text
11. Provisioning
12. Snapshots
13. Networking
14. Guest control
15. LabsBackery contract
```

The goal is not to rewrite forever.

The goal is to build a foundation you understand, can explain, and can extend.
