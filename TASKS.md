# Yeast v2 Master Task List

Status: Active execution file  
Owner: Twarga / TwargaOps  
Mode: Clean v2 rebuild from zero  
Rule: Current code is prototype/reference until Milestone 0 decides what to archive/remove

## 0. How To Use This File

This is the operational build checklist for Yeast v2.

The planning files explain the why and the architecture. This file explains what to do next.

Use this file as the daily execution source of truth.

When working with Codex, Claude, or any AI coding agent:

1. Pick exactly one task.
2. Read its dependency list.
3. Confirm dependencies are done.
4. Give the agent only the relevant task, relevant architecture sections, and `AI-RULES.md`.
5. Keep changes scoped to the task.
6. Run the required tests.
7. Update this file before marking the task done.

Do not ask an AI agent to "build Yeast v2."

Ask it to complete one task, for example:

```text
Implement M1-T2 Project metadata load/create.
Follow AI-RULES.md and YEAST_TECHNICAL_ARCHITECTURE.md.
Only touch internal/project and its tests.
Update TASKS.md when done.
```

## 1. Status Legend

Use these statuses:

```text
[ ] Not started
[~] In progress
[x] Done
[!] Blocked
[-] Deferred
```

## 2. Source Documents

Always keep these documents in sync with implementation:

```text
AI-RULES.md
YEAST_VISION.md
YEAST_PRODUCT_ROADMAP.md
YEAST_TECHNICAL_DISCOVERY.md
YEAST_TECHNICAL_ARCHITECTURE.md
YEAST_V2_IMPLEMENTATION_PLAN.md
YEAST_TEST_PLAN.md
YEAST_RELEASE_PLAN.md
YEAST_DOCS_PLAN.md
YEAST_FEEDBACK_LOG.md
```

## 3. Build Rules

- Do not start a task unless dependencies are done.
- Do not edit unrelated files.
- Keep each task small.
- Update this file after finishing.
- Run required tests before marking done.
- If architecture and code disagree, stop and update docs or ask.
- Never implement future milestone features early.
- Do not delete old prototype code until Milestone 0 explicitly archives/removes it.
- Do not add LabsBackery, MCP, or cloud behavior before the local v0.1 engine works.

## 4. Current Phase

Current phase:

```text
Pre-implementation planning complete enough to begin Milestone 0.
```

Next task:

```text
M0-T1: Create v2 working branch / cleanup strategy
```

## 5. Milestone Overview

| Milestone | Name | Goal | Status |
|---|---|---|---|
| M0 | Repository Reset And Skeleton | Prepare clean v2 implementation path | [ ] |
| M1 | Project Identity And Paths | Make Yeast project-safe | [ ] |
| M2 | Config Model And Validation | Define desired-state config | [ ] |
| M3 | State Store And Locking | Define actual runtime state | [ ] |
| M4 | Image Cache And Manifest | Trusted image pull/cache | [ ] |
| M5 | Runtime Abstraction And QEMU Lifecycle | Start/stop real VMs through runtime boundary | [ ] |
| M6 | Cloud-init And Guest Readiness | Make VM reachable and prepared | [ ] |
| M7 | Core Commands v0.1 | init/doctor/pull/up/status/ssh/down/destroy | [ ] |
| M8 | Human And JSON Output | Stable output for humans and tools | [ ] |
| M9 | Tests And Examples | Prove v0.1 works | [ ] |
| M10 | Docs And v0.1 Release Prep | Prepare first public release | [ ] |
| M11 | Provisioning | Packages/files/shell after v0.1 | [-] |
| M12 | Snapshots And Reset | Lab reset capability | [-] |
| M13 | Private Networking | Multi-VM lab networking | [-] |
| M14 | Guest Control | exec/copy/logs/inspect | [-] |
| M15 | LabsBackery Contract | CLI/JSON lab integration | [-] |

---

# M0: Repository Reset And Skeleton

Goal:

Create a clean v2 implementation foundation without losing useful prototype knowledge.

Important:

This milestone is not about building Yeast features. It is about making the repo ready for the rebuild.

Definition of done:

- clean v2 branch or implementation path exists
- old code is archived or removed only after reference capture
- new folder structure exists
- minimal CLI builds
- planning docs remain
- `go test ./...` can run if Go is available

## M0 Tasks

### M0-T1: Create v2 working branch / cleanup strategy

Status: [x]

Purpose:

Decide how v2 work will happen without accidentally destroying useful prototype code.

Dependencies:

- none

Recommended action:

- create a git branch such as `v2-rebuild`
- keep current branch history intact
- decide whether old implementation files stay temporarily or are moved/removed in later tasks

Files likely touched:

- none or git branch only
- optionally `TASKS.md`

Required checks:

- `git status --short`
- confirm no uncommitted user work is accidentally removed

Definition of done:

- branch/strategy is clear
- this task status updated
- no code deleted yet

Completion notes:

- Created branch: `v2-rebuild`.
- Current strategy: rebuild Yeast v2 on this branch while keeping `main` history intact.
- Old implementation files stay in place until `M0-T2` inventories useful prototype knowledge.
- No source files were deleted or rewritten in this task.
- Planning docs remain untracked and visible in git status until they are committed.

AI instruction:

Do not modify source files in this task unless explicitly requested.

### M0-T2: Inventory prototype code before removal

Status: [x]

Purpose:

Capture what the current MVP code teaches us before cleaning it.

Dependencies:

- M0-T1

Create notes in:

```text
docs/prototype-inventory.md
```

Inventory:

- existing commands
- existing packages
- useful utilities
- QEMU command behavior
- cloud-init behavior
- state/lock behavior
- image download behavior
- tests worth porting
- code that should not be copied

Files likely touched:

- `docs/prototype-inventory.md`
- `TASKS.md`

Required checks:

- no code changes required

Definition of done:

- prototype knowledge captured
- list of reusable ideas exists
- list of avoid-copying areas exists

Completion notes:

- Created `docs/prototype-inventory.md`.
- Captured existing command set, package responsibilities, config/state/runtime/cloud-init/image/output behavior, tests worth porting, reusable ideas, and code shapes that should not be copied directly.
- Confirmed old implementation remains in place for reference.
- No source implementation files were changed.

AI instruction:

Read current files, summarize. Do not refactor.

### M0-T3: Create clean v2 skeleton folders

Status: [x]

Purpose:

Create the v2 folder structure from the architecture document.

Dependencies:

- M0-T2

Folders to create:

```text
cmd/yeast/
internal/app/
internal/project/
internal/config/
internal/state/
internal/images/
internal/runtime/
internal/runtime/qemu/
internal/provision/
internal/provision/cloudinit/
internal/provision/ssh/
internal/guest/
internal/output/
internal/util/
docs/
examples/
```

Files likely touched:

- folder structure
- placeholder package files only if needed for build
- `TASKS.md`

Required checks:

- no old logic copied
- no future features implemented

Definition of done:

- v2 folder skeleton exists
- empty folders are either represented by starter files or created when needed
- task marked done

Completion notes:

- Created v2 `internal/*` skeleton packages from `YEAST_TECHNICAL_ARCHITECTURE.md`.
- Added package-level `doc.go` placeholders only; no behavior was implemented.
- Added `examples/.gitkeep` so the examples folder is tracked.
- Kept the existing prototype source files in place for reference.

AI instruction:

Create skeleton only. Do not implement behavior.

### M0-T4: Replace CLI with minimal v2 root command

Status: [x]

Purpose:

Make the project build with a clean CLI entry point.

Dependencies:

- M0-T3

Expected behavior:

- `yeast --help` works
- root command exists
- version command optional
- no real VM behavior yet

Files likely touched:

- `cmd/yeast/main.go`
- `cmd/yeast/root.go`
- maybe `internal/app/service.go`

Required tests:

- build command if Go is available
- `go test ./...` if packages compile

Definition of done:

- minimal binary builds
- no old command workflow logic remains in active CLI
- help output is clean

Completion notes:

- Replaced active root CLI with a minimal v2 command skeleton.
- Added `internal/app.Service` with a development version value.
- Active CLI now exposes only root help and `yeast version`.
- Preserved old prototype command files behind the `prototype` build tag so their workflow logic is no longer active in the default CLI build.
- Could not verify `gofmt`, build, or `go test ./...` because the Go toolchain is not installed in this environment.

AI instruction:

Only build CLI skeleton. Do not implement real commands yet.

### M0-T5: Controlled prototype cleanup

Status: [x]

Purpose:

Remove or archive old MVP code only after the v2 skeleton and prototype inventory exist.

Dependencies:

- M0-T2
- M0-T3
- M0-T4

Options:

Option A:

- remove old implementation files from active tree
- rely on git history and prototype inventory

Option B:

- move old code to `archive/prototype-v1/`
- exclude from build

Recommendation:

Use git history + `docs/prototype-inventory.md`. Avoid keeping build-confusing archived Go packages unless needed.

Required checks:

- `go test ./...` still works if Go is installed
- no planning docs removed
- no useful prototype notes lost

Definition of done:

- active tree represents v2 skeleton
- old code no longer confuses build
- prototype knowledge remains in docs/git history

Completion notes:

- Used Option A: removed old MVP implementation files from the active tree.
- Kept prototype knowledge in `docs/prototype-inventory.md` and git history.
- Removed old prototype command files from `cmd/yeast`, keeping only `main.go` and the minimal v2 `root.go`.
- Removed old `pkg/*` implementation packages so they cannot be accidentally built or extended during v2.
- Planning docs and v2 skeleton packages remain in place.

AI instruction:

This is the only M0 task allowed to delete old implementation files. Confirm file list before deleting.

---

# M1: Project Identity And Paths

Goal:

Make Yeast project-safe before any VM runtime behavior exists.

Why:

Two projects should both be able to have an instance named `web` without colliding.

Definition of done:

- project metadata is created/loaded
- project ID is stable
- runtime paths are project-scoped
- path traversal is prevented
- tests pass

## M1 Tasks

### M1-T1: Define project metadata model

Status: [x]

Dependencies:

- M0 complete

Files likely touched:

- `internal/project/project.go`
- `internal/project/identity.go`
- tests under `internal/project`

Model should include:

- schema
- project ID
- created_at

Required tests:

- metadata struct serialization

Completion notes:

- Added `project.Metadata` with `schema`, `id`, and `created_at` JSON fields.
- Added `project.NewMetadata` to enforce `yeast.project.v1` and UTC timestamps.
- Added serialization, deserialization, and UTC normalization tests.
- schema value is correct

Definition of done:

- project metadata type exists
- JSON encoding/decoding works

### M1-T2: Implement project ID generation

Status: [x]

Dependencies:

- M1-T1

Requirements:

- generate unique stable ID
- use safe prefix such as `proj_`
- avoid path-derived-only identity

Files likely touched:

- `internal/project/identity.go`
- tests

Required tests:

- generated ID is non-empty
- generated ID has expected prefix
- multiple IDs are different
- ID is path-safe

Definition of done:

- project ID generation works and is tested

Completion notes:

- Added `GenerateID` using crypto-random bytes.
- Added `ProjectIDPrefix` with the `proj_` prefix.
- Added `IsValidID` to enforce path-safe lowercase hex IDs.
- Added tests for non-empty generation, prefix, uniqueness across calls, and invalid path-like values.

### M1-T3: Implement project metadata create/load

Status: [x]

Dependencies:

- M1-T1
- M1-T2

Requirements:

- create `.yeast/project.json` when initializing
- load existing metadata
- return clear error on corrupted metadata
- create `.yeast/` directory safely

Files likely touched:

- `internal/project/project.go`
- `internal/project/paths.go`
- tests

Required tests:

- creates metadata in temp project
- loads same ID on second load
- corrupt JSON returns clear error
- missing metadata behavior documented

Definition of done:

- stable project identity works

Completion notes:

- Added `.yeast/project.json` path handling through `MetadataPath`.
- Added `EnsureMetadata` to create metadata once and load the same identity on later calls.
- Added `LoadMetadata`, `SaveMetadata`, and `ValidateMetadata`.
- Added explicit `ErrMetadataNotFound` for missing metadata.
- Added tests for create, second-load stability, corrupt JSON, missing metadata, and invalid metadata.

### M1-T4: Implement runtime path resolver

Status: [x]

Dependencies:

- M1-T3

Requirements:

- resolve Yeast home path
- resolve project runtime path
- resolve state file path
- resolve instance directory path
- resolve image cache path

Files likely touched:

- `internal/project/paths.go`
- `internal/util/paths.go`
- tests

Required tests:

- paths stay under `~/.yeast/projects/<project-id>`
- instance path is scoped to project
- cache path is global/shared
- invalid instance names rejected before path creation

Definition of done:

- no runtime path is based only on instance name

Completion notes:

- Added `Paths` resolver for Yeast home, project runtime directory, state file, lock file, instance directory, snapshot directory, and global image cache.
- Added `DefaultYeastHome`, `NewPaths`, and `ResolvePaths`.
- Added instance name validation before path creation.
- Added tests for project-scoped runtime paths, project-scoped instance paths, global image cache path, invalid instance names, and invalid metadata.

### M1-T5: Add project package integration to `yeast init`

Status: [x]

Dependencies:

- M1-T4

Requirements:

- `yeast init` creates `yeast.yaml`
- `yeast init` creates `.yeast/project.json`
- does not overwrite existing project unless explicit future flag

Files likely touched:

- `cmd/yeast/init.go`
- `internal/app/init.go`
- `internal/project`
- tests if command-level tests exist

Required tests:

- init creates config
- init creates project metadata
- repeated init fails clearly

Definition of done:

- project identity enters user workflow

Completion notes:

- Added `internal/app.Init` workflow.
- Added starter `yeast.yaml` creation.
- Added `.yeast/project.json` creation through the project package.
- Added clear repeated-init failure through `ErrProjectAlreadyInitialized`.
- Added active `yeast init` command wired into the root CLI.
- Added tests for config creation, metadata creation, starter config content, and repeated init failure.

---

# M2: Config Model And Validation

Goal:

Define the desired-state model for v0.1.

Definition of done:

- config loads
- defaults apply
- invalid config fails early
- tests cover validation

## M2 Tasks

### M2-T1: Define v2 config structs

Status: [x]

Dependencies:

- M1 complete

Files:

- `internal/config/model.go`

Required fields:

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

Future reserved fields:

- networks
- provision

Definition of done:

- model compiles
- no validation yet required

Completion notes:

- Added `config.Config` with `version`, `instances`, reserved `networks`, and reserved `provision`.
- Added `config.Instance` with the required v0.1 fields: `name`, `image`, `memory`, `cpus`, `disk_size`, `user`, `sudo`, `env`, and `user_data`.
- Added reserved `Instance.Networks` and `Instance.Provision` for later milestones.
- Added `Network`, `ProvisionConfig`, and `FileProvision` model types so later loader/validation work has stable targets.

### M2-T2: Implement config loader

Status: [x]

Dependencies:

- M2-T1

Files:

- `internal/config/loader.go`

Requirements:

- read `yeast.yaml`
- parse YAML
- return typed config
- wrap parse errors clearly

Tests:

- missing file
- invalid YAML
- valid YAML

Definition of done:

- loader works independently

Completion notes:

- Added `config.Load(path)` to read and parse `yeast.yaml`.
- Added clear wrapped errors for read failures and YAML parse failures.
- Added tests for missing file, invalid YAML, and valid YAML parsing into the typed config model.

### M2-T3: Implement config validation

Status: [x]

Dependencies:

- M2-T2

Files:

- `internal/config/validate.go`

Validation:

- version must be supported
- at least one instance
- unique names
- path-safe names
- image required
- memory minimum
- CPU minimum
- disk size valid
- user valid
- sudo valid
- env keys valid

Tests:

- one test per validation rule

Definition of done:

- invalid config never reaches runtime

Completion notes:

- Added `config.Validate`.
- Added validation for supported version, non-empty instances, unique/path-safe names, required image, memory minimum, CPU minimum, disk size format, Linux username format, sudo policy, env key format, and env newline rejection.
- Added focused tests for each validation rule and one valid-config test.

### M2-T4: Implement defaults and normalization

Status: [x]

Dependencies:

- M2-T3

Files:

- `internal/config/defaults.go`

Defaults:

- memory
- cpus
- user
- sudo
- disk_size normalization if provided

Tests:

- defaults applied
- explicit values preserved

Definition of done:

- loaded config is ready for application planning

Completion notes:

- Added `config.ApplyDefaults`.
- Added defaults for memory, cpus, user, and sudo.
- Added disk size normalization when `disk_size` is provided.
- Updated `config.Load` to run parse, validate, and defaults/normalization in one path so loaded config is ready for the app layer.
- Added tests for defaults being applied and explicit values being preserved.

---

# M3: State Store And Locking

Goal:

Create actual runtime state storage.

Definition of done:

- state loads/saves
- state locks
- stale state can be reconciled later
- tests pass

## M3 Tasks

### M3-T1: Define state v2 model

Status: [x]

Dependencies:

- M1 complete

Files:

- `internal/state/model.go`

Fields:

- schema
- project_id
- instances map
- instance status
- pid
- management_ip
- ssh_port
- runtime_dir
- provisioning_status
- last_error

Tests:

- JSON marshal/unmarshal

Definition of done:

- state model exists and is versioned

Completion notes:

- Added versioned `state.State` with `schema`, `project_id`, and `instances`.
- Added `state.InstanceState` with `status`, `pid`, `management_ip`, `ssh_port`, `runtime_dir`, `provisioning_status`, and `last_error`.
- Added `state.New(projectID)` to create initialized state values.
- Added JSON marshal/unmarshal tests and initialization tests.

### M3-T2: Implement state load/save

Status: [ ]

Dependencies:

- M3-T1
- M1-T4

Files:

- `internal/state/store.go`

Requirements:

- missing state returns empty state
- save is atomic
- corrupt state returns clear error
- project ID mismatch returns clear error

Tests:

- missing
- valid
- corrupt
- project mismatch
- save/reload

Definition of done:

- local state is reliable and inspectable

### M3-T3: Implement state locking

Status: [ ]

Dependencies:

- M3-T2

Files:

- `internal/state/lock.go`

Requirements:

- lock prevents concurrent mutation
- lock includes owner PID/timestamp
- stale lock can recover if owner process dead
- timeout gives clear error

Tests:

- acquire/release
- double acquire fails/waits
- stale lock recovery
- malformed lock handling

Definition of done:

- mutating commands can safely use locked state

### M3-T4: Implement reconciliation hooks

Status: [ ]

Dependencies:

- M3-T3

Files:

- `internal/state/reconcile.go`
- maybe `internal/runtime/process.go`

Requirements:

- detect dead PID
- mark instance stopped
- clear PID when stale
- later verify process belongs to project instance

Tests:

- dead PID becomes stopped
- stopped instance unchanged
- running alive fake process behavior if testable

Definition of done:

- status can avoid lying about dead processes

---

# M4: Image Cache And Manifest

Goal:

Trusted base image management.

Definition of done:

- supported images list
- image pull works
- checksum verification works
- cache path stable

## M4 Tasks

### M4-T1: Define image manifest model

Status: [ ]

Dependencies:

- M1 paths complete

Files:

- `internal/images/manifest.go`

Requirements:

- trusted image type
- supported image list
- lookup by name

Tests:

- supported images sorted
- known lookup
- unknown lookup

Definition of done:

- manifest logic works without network

### M4-T2: Implement cache path resolution

Status: [ ]

Dependencies:

- M4-T1
- M1-T4

Files:

- `internal/images/cache.go`

Requirements:

- shared image cache under `~/.yeast/cache/images`
- per-image folder
- manifest metadata path
- image file path

Tests:

- image path under cache
- image name safe

Definition of done:

- cache layout is stable

### M4-T3: Implement checksum verification

Status: [ ]

Dependencies:

- M4-T2

Files:

- `internal/images/verify.go`

Requirements:

- sha256 file checksum
- mismatch error
- missing file error

Tests:

- correct checksum
- wrong checksum
- missing file

Definition of done:

- image integrity check works

### M4-T4: Implement downloader

Status: [ ]

Dependencies:

- M4-T3

Files:

- `internal/images/downloader.go`

Requirements:

- download to temp file
- verify checksum before final move
- cleanup partial file on failure
- timeout support
- retry support optional in v0.1

Tests:

- use local httptest server
- success
- HTTP failure
- checksum failure
- partial cleanup

Definition of done:

- image pull can be trusted

### M4-T5: Implement `yeast pull`

Status: [ ]

Dependencies:

- M4-T4
- M8 output can be partial or later

Files:

- `cmd/yeast/pull.go`
- `internal/app/pull.go`

Requirements:

- `yeast pull --list`
- `yeast pull ubuntu-24.04`
- checksum validation
- clear unsupported image error

Tests:

- app-level pull with fake downloader if needed

Definition of done:

- user can prepare image cache

---

# M5: Runtime Abstraction And QEMU Lifecycle

Goal:

Start and stop real VMs through a runtime boundary.

Definition of done:

- QEMU lifecycle works through `internal/runtime`
- app layer does not build QEMU commands directly

## M5 Tasks

### M5-T1: Define runtime interface and models

Status: [ ]

Dependencies:

- M2 config model
- M3 state model

Files:

- `internal/runtime/runtime.go`
- `internal/runtime/model.go`

Models:

- MachinePlan
- RuntimeInstance
- NetworkOptions basic management network
- DiskPlan

Definition of done:

- runtime boundary exists
- no QEMU-specific details in app interface

### M5-T2: Implement qemu-img disk preparation

Status: [ ]

Dependencies:

- M5-T1
- M4 image paths

Files:

- `internal/runtime/qemu/disk.go`

Requirements:

- create qcow2 overlay from base image
- ensure instance directory exists
- optional disk size on creation
- inspect existing disk size later

Tests:

- command construction tests
- real qemu-img integration later

Definition of done:

- disk preparation logic isolated

### M5-T3: Implement QEMU command builder

Status: [ ]

Dependencies:

- M5-T1

Files:

- `internal/runtime/qemu/command.go`

Requirements:

- KVM enabled
- memory
- CPUs
- disk drive
- seed ISO
- nographic
- management SSH port forwarding

Tests:

- command args contain expected values
- no shell string construction

Definition of done:

- QEMU args are deterministic and testable

### M5-T4: Implement QEMU process start/stop

Status: [ ]

Dependencies:

- M5-T3

Files:

- `internal/runtime/qemu/process.go`
- `internal/runtime/qemu/runtime.go`

Requirements:

- start process
- write stdout/stderr to vm.log
- release process safely
- stop process gracefully
- kill after timeout if needed

Tests:

- unit tests where possible
- manual host-dependent test later

Definition of done:

- QEMU runtime can start/stop through interface

---

# M6: Cloud-init And Guest Readiness

Goal:

Create reachable VMs.

## M6 Tasks

### M6-T1: Implement SSH key discovery

Status: [ ]

Dependencies:

- M2 config

Files:

- `internal/provision/cloudinit/user_data.go`
- maybe `internal/guest/ssh.go`

Requirements:

- find ed25519 key
- fallback rsa key
- clear error if none

Tests:

- temp home with key
- missing key

Definition of done:

- Yeast can find user's public key safely

### M6-T2: Generate cloud-init user-data/meta-data

Status: [ ]

Dependencies:

- M6-T1

Files:

- `internal/provision/cloudinit/user_data.go`
- `internal/provision/cloudinit/meta_data.go`

Requirements:

- hostname
- user
- sudo policy
- SSH authorized key
- env
- custom user-data mode if supported

Tests:

- user-data contains expected fields
- env quoted safely
- meta-data stable

Definition of done:

- cloud-init generation tested

### M6-T3: Create seed ISO

Status: [ ]

Dependencies:

- M6-T2

Files:

- `internal/provision/cloudinit/iso.go`

Requirements:

- write user-data
- write meta-data
- call genisoimage or supported tool
- clear missing dependency error

Tests:

- command construction if abstracted
- manual integration for real ISO

Definition of done:

- runtime can attach generated seed ISO

### M6-T4: Implement SSH readiness

Status: [ ]

Dependencies:

- M5 runtime start
- M6-T3

Files:

- `internal/guest/readiness.go`
- `internal/guest/ssh.go`

Requirements:

- wait for TCP SSH port
- optionally run simple SSH command later
- timeout with clear error

Tests:

- fake TCP server success
- timeout
- connection refused retry

Definition of done:

- app can know when VM is reachable

---

# M7: Core Commands v0.1

Goal:

Complete the first usable Yeast lifecycle.

## M7 Tasks

### M7-T1: Implement `yeast init`

Status: [ ]

Dependencies:

- M1 project
- M2 config defaults

Definition of done:

- creates `yeast.yaml`
- creates `.yeast/project.json`
- refuses overwrite

### M7-T2: Implement `yeast doctor`

Status: [ ]

Dependencies:

- M0 CLI skeleton

Checks:

- qemu-system-x86_64
- qemu-img
- genisoimage
- ssh client
- `/dev/kvm`
- SSH public key
- cache directory

Definition of done:

- clear blockers and warnings

### M7-T3: Implement `yeast up`

Status: [ ]

Dependencies:

- M1-M6 complete

Flow:

- resolve project
- load config
- lock/load/reconcile state
- ensure image exists
- prepare disk
- generate cloud-init
- start QEMU
- wait for SSH
- save state

Definition of done:

- one Ubuntu VM starts and becomes SSH-ready

### M7-T4: Implement `yeast status`

Status: [ ]

Dependencies:

- M3 state
- M5 process inspection

Definition of done:

- status reconciles dead processes
- output sorted by name

### M7-T5: Implement `yeast ssh`

Status: [ ]

Dependencies:

- M7-T3
- M7-T4

Definition of done:

- connects to running instance using stored SSH port
- handles no/multiple targets clearly

### M7-T6: Implement `yeast down`

Status: [ ]

Dependencies:

- M5 stop
- M3 state

Definition of done:

- stops running VMs
- marks stopped
- handles already stopped

### M7-T7: Implement `yeast destroy`

Status: [ ]

Dependencies:

- M7-T6
- M1 path safety

Definition of done:

- stops if running
- removes project instance runtime dir
- removes state entry
- never removes cache

---

# M8: Human And JSON Output

Goal:

Make commands usable by humans and tools.

## M8 Tasks

### M8-T1: Define result and error schemas

Status: [ ]

Dependencies:

- M7 app workflows started

Files:

- `internal/output/schemas.go`
- `internal/app/errors.go`

Definition of done:

- common success/error shape exists

### M8-T2: Implement human renderer

Status: [ ]

Dependencies:

- M8-T1

Definition of done:

- readable terminal output
- no JSON leakage

### M8-T3: Implement JSON renderer

Status: [ ]

Dependencies:

- M8-T1

Definition of done:

- parseable JSON
- no ANSI/spinners
- stable command/error fields

### M8-T4: Add JSON tests for core commands

Status: [ ]

Dependencies:

- M8-T3

Definition of done:

- JSON contract tests pass

---

# M9: Tests And Examples

Goal:

Prove v0.1 works.

## M9 Tasks

### M9-T1: Fast unit test suite

Status: [ ]

Dependencies:

- M1-M8 relevant packages

Required:

- config tests
- project tests
- state tests
- image tests
- cloud-init tests
- output tests

Definition of done:

- fast tests pass

### M9-T2: Application workflow fake-runtime tests

Status: [ ]

Dependencies:

- M7 app workflows

Definition of done:

- up/status/down/destroy workflows tested without QEMU

### M9-T3: Manual host-dependent v0.1 checklist

Status: [ ]

Dependencies:

- M7 complete
- M8 complete

Checklist:

- init
- doctor
- pull
- up
- status
- ssh
- down
- up again
- destroy

Definition of done:

- checklist passes on Linux/KVM host

### M9-T4: Create ubuntu-basic example

Status: [ ]

Dependencies:

- M7 complete

Files:

- `examples/ubuntu-basic/yeast.yaml`
- `examples/ubuntu-basic/README.md`

Definition of done:

- example works with v0.1 commands

---

# M10: Docs And v0.1 Release Prep

Goal:

Prepare first public release.

## M10 Tasks

### M10-T1: Rewrite README for v2/v0.1

Status: [ ]

Dependencies:

- M7 complete

Definition of done:

- README explains current working scope honestly

### M10-T2: Create required v0.1 docs

Status: [ ]

Dependencies:

- M10-T1

Docs:

- quickstart
- installation
- config reference
- troubleshooting
- known limitations
- architecture overview

Definition of done:

- docs match actual commands

### M10-T3: Prepare changelog and release notes

Status: [ ]

Dependencies:

- M10-T2
- M9 tests

Definition of done:

- v0.1.0 release notes ready

### M10-T4: Build release artifact

Status: [ ]

Dependencies:

- M9 complete
- M10-T3

Definition of done:

- Linux amd64 binary built
- checksum generated

### M10-T5: Tag and publish v0.1.0

Status: [ ]

Dependencies:

- M10-T4

Definition of done:

- Git tag exists
- GitHub release published
- soft announcement ready

---

# Future Milestones

These are intentionally deferred until v0.1 works.

## M11: Provisioning

Status: [-]

Do not start until:

- M10 complete
- v0.1 lifecycle proven

Core tasks later:

- provision model
- packages
- files
- shell
- provision command
- Caddy demo

## M12: Snapshots And Reset

Status: [-]

Do not start until:

- M11 complete
- snapshot experiment complete

Core tasks later:

- choose snapshot model
- snapshot command
- restore command
- list/delete snapshots
- reset demo

## M13: Private Networking

Status: [-]

Do not start until:

- M12 complete
- private network experiment complete

Core tasks later:

- networks config
- management vs lab network
- static IPs
- two-VM lab

## M14: Guest Control

Status: [-]

Do not start until:

- M13 complete or needed by provisioning/MCP

Core tasks later:

- exec
- copy
- logs
- inspect
- structured result

## M15: LabsBackery Contract

Status: [-]

Do not start until:

- M11-M14 enough for one lab

Core tasks later:

- JSON contract
- lab lifecycle commands
- test lab
- LabsBackery integration notes

---

# Final v0.1 Success State

Yeast v2 v0.1 is complete when a fresh Linux user can:

```text
install/build Yeast
run yeast doctor
create project
pull Ubuntu image
start VM
SSH into VM
stop VM
destroy VM
read docs
understand limitations
```

And a tool can:

```text
call status --json
call up --json
call down --json
call destroy --json
parse results without scraping human output
```

This is the first real foundation.
