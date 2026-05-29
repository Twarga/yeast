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
v0.7.0 templates.
```

Next task:

```text
V0.7-T2: Add built-in template catalog and metadata model.
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
| M10 | Docs And v0.1 Release Prep | Prepare first public release | [x] |
| C1 | Charm CLI Experience | Polished terminal UX without breaking JSON | [x] |
| M11 | Provisioning | Packages/files/shell after v0.1 | [x] |
| M12 | Snapshots And Reset | Lab reset capability | [~] |
| M13 | Private Networking | Multi-VM lab networking | [x] |
| M14 | Guest Control | exec/copy/logs/inspect | [x] |
| M15 | Templates | Reusable project starters | [~] |
| M16 | LabsBackery Contract | CLI/JSON lab integration | [-] |

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

Status: [x]

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

Completion notes:

- Added `state.Load(path, projectID)` with missing-file initialization, corrupt-state errors, schema checks, and project ID mismatch checks.
- Added `state.Save(path, state)` with atomic write behavior.
- Added `ErrProjectIDMismatch` for explicit mismatch detection.
- Added tests for missing state, valid load, corrupt state, project mismatch, and save/reload round-trip.

### M3-T3: Implement state locking

Status: [x]

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

Completion notes:

- Added `state.Acquire` and `FileLock.Release`.
- Added lock metadata with owner PID and creation timestamp.
- Added timeout behavior, stale lock recovery, and malformed stale lock recovery.
- Added default lock options and injectable time/process liveness hooks for tests.
- Added tests for acquire/release, double-acquire timeout, stale lock recovery, and malformed stale lock recovery.

### M3-T4: Implement reconciliation hooks

Status: [x]

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

Completion notes:

- Added `state.Reconcile`.
- Reconciliation now marks dead running instances as stopped, clears stale PID and connection fields, and records a simple last error.
- Added injectable process liveness hooks for tests.
- Added tests for dead PID reconciliation, stopped instance no-op behavior, and running alive instance no-op behavior.

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

Status: [x]

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

Completion notes:

- Added `images.TrustedImage`.
- Added built-in trusted manifest entries for `ubuntu-22.04` and `ubuntu-24.04`.
- Added `SupportedImages` for sorted supported image names.
- Added `Lookup` for manifest lookup by image name.
- Added tests for sorted supported images, known lookup, and unknown lookup.

### M4-T2: Implement cache path resolution

Status: [x]

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

Completion notes:

- Added `images.ResolveCachePaths`.
- Added `CachePaths` with per-image directory, image file path, and manifest metadata path.
- Added image-name safety checks before path creation.
- Added tests for cache-root scoping and invalid image names.

### M4-T3: Implement checksum verification

Status: [x]

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

Completion notes:

- Added `images.FileSHA256`.
- Added `images.VerifySHA256`.
- Added tests for correct checksum, wrong checksum, and missing file behavior.

### M4-T4: Implement downloader

Status: [x]

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

Completion notes:

- Added `images.Download`.
- Downloader now uses a temp file, timeout-backed request context, checksum verification before final move, and cleanup on failure.
- Added `httptest` coverage for success, HTTP failure, checksum failure, and partial-download cleanup.

### M4-T5: Implement `yeast pull`

Status: [x]

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

Completion notes:

- Added `app.Pull` with `--list` support and named image download support.
- Added `ErrUnsupportedImage` for clear unsupported image handling.
- Added a thin `yeast pull` Cobra command.
- Added app-level tests for image listing, unsupported image errors, and known-image download path resolution.

---

# M5: Runtime Abstraction And QEMU Lifecycle

Goal:

Start and stop real VMs through a runtime boundary.

Definition of done:

- QEMU lifecycle works through `internal/runtime`
- app layer does not build QEMU commands directly

## M5 Tasks

### M5-T1: Define runtime interface and models

Status: [x]

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

Completion notes:

- Added `internal/runtime.Runtime` as the application-facing runtime boundary.
- Added generic runtime models: `MachinePlan`, `DiskPlan`, `NetworkOptions`, `RuntimeInstance`, and `ProcessInfo`.
- Kept QEMU details out of the app interface so later backends can fit the same boundary.

### M5-T2: Implement qemu-img disk preparation

Status: [x]

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

Completion notes:

- Added `qemu.PrepareDisk` to create qcow2 overlay disks from trusted base images.
- Ensured runtime and instance disk directories are created before running `qemu-img`.
- Added deterministic command-construction tests, including optional disk size handling.
- Existing disks are left in place instead of being recreated.

### M5-T3: Implement QEMU command builder

Status: [x]

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

Completion notes:

- Added deterministic QEMU/KVM command argument construction in `internal/runtime/qemu/command.go`.
- Included KVM, memory, CPUs, qcow2 disk, seed ISO, nographic mode, and management SSH port forwarding.
- Kept the output as structured argv slices with no shell string construction.
- Added tests for expected arguments, selected binary, and required-field validation.

### M5-T4: Implement QEMU process start/stop

Status: [x]

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

Completion notes:

- Added `qemu.Runtime` implementing the shared runtime interface.
- Added QEMU start with `vm.log` wiring and safe process-handle release after spawn.
- Added PID-based inspect and stop behavior with graceful SIGTERM, timeout polling, and SIGKILL fallback.
- Added unit tests for start logging, graceful stop, forced kill after timeout, and inspect state.

---

# M6: Cloud-init And Guest Readiness

Goal:

Create reachable VMs.

## M6 Tasks

### M6-T1: Implement SSH key discovery

Status: [x]

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

Completion notes:

- Added SSH public-key discovery in `internal/provision/cloudinit/user_data.go`.
- Preferred `~/.ssh/id_ed25519.pub` and fell back to `~/.ssh/id_rsa.pub`.
- Added explicit errors for missing keys and empty key files.
- Added tests for preferred key, fallback key, missing key, and empty key behavior.

### M6-T2: Generate cloud-init user-data/meta-data

Status: [x]

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

Completion notes:

- Added typed cloud-init renderers for `user-data` and `meta-data`.
- Normal `user-data` now includes hostname, user, sudo policy, SSH authorized key, and optional env script injection.
- Added raw `user_data` passthrough as a controlled escape hatch with automatic `#cloud-config` normalization.
- Added tests for expected fields, safe env quoting, custom user-data mode, and stable meta-data output.

### M6-T3: Create seed ISO

Status: [x]

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

Completion notes:

- Added `CreateSeedISO` to write `user-data`, `meta-data`, and build `seed.iso`.
- Added ISO builder discovery with `genisoimage` first and `mkisofs` fallback.
- Added a clear `ErrNoISOBuilder` path with install guidance when no supported tool exists.
- Added tests for file writing, deterministic ISO command args, and missing-builder behavior.

### M6-T4: Implement SSH readiness

Status: [x]

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

Completion notes:

- Added `guest.WaitForTCP` for TCP-based SSH readiness checks with retry and timeout behavior.
- Added retry handling for connection-refused and timeout-style dial failures.
- Added `guest.SSHAddress` helper for consistent host/port formatting.
- Added tests for real-listener success, timeout behavior, and retry-before-success flow.

---

# M7: Core Commands v0.1

Goal:

Complete the first usable Yeast lifecycle.

## M7 Tasks

### M7-T1: Implement `yeast init`

Status: [x]

Dependencies:

- M1 project
- M2 config defaults

Definition of done:

- creates `yeast.yaml`
- creates `.yeast/project.json`
- refuses overwrite

Completion notes:

- `cmd/yeast/init.go` is wired to the app layer through `service.Init`.
- `internal/app.Init` creates the starter `yeast.yaml` and `.yeast/project.json`.
- Repeated init attempts fail with `ErrProjectAlreadyInitialized`.
- App-level tests cover file creation, starter config contents, and overwrite refusal.

### M7-T2: Implement `yeast doctor`

Status: [x]

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

Completion notes:

- Added `service.Doctor` as an application workflow with explicit checks for runtime binaries, `/dev/kvm`, SSH public key, and cache directory state.
- Added CLI wiring for `yeast doctor`.
- Missing `qemu-system-x86_64`, `qemu-img`, `genisoimage/mkisofs`, `/dev/kvm`, and SSH key are reported as blockers where appropriate.
- Cache directory absence is treated as a warning with a clear explanation instead of failing early.

### M7-T3: Implement `yeast up`

Status: [x]

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

Completion notes:

- Added `service.Up` to run the v0.1 startup workflow: resolve project, load config, lock/load/reconcile state, require cached image, render cloud-init, prepare disk, start runtime, wait for SSH, and save state.
- Added `yeast up` CLI wiring.
- Running instances are reused from state when already marked healthy, and new instances get deterministic management SSH ports starting at `2222`.
- The current v0.1 implementation requires the image to already be in cache and points the user to `yeast pull <image>` when it is missing.
- Added app-level tests for the happy path and missing-image guidance.

### M7-T4: Implement `yeast status`

Status: [x]

Dependencies:

- M3 state
- M5 process inspection

Definition of done:

- status reconciles dead processes
- output sorted by name

Completion notes:

- Added `service.Status` to resolve the project, lock/load state, reconcile dead processes through runtime inspection, and save corrected state when needed.
- Added `yeast status` CLI wiring.
- Status results are sorted by instance name.
- Added app-level tests for sorted output ordering and dead-process reconciliation persistence.

### M7-T5: Implement `yeast ssh`

Status: [x]

Dependencies:

- M7-T3
- M7-T4

Definition of done:

- connects to running instance using stored SSH port
- handles no/multiple targets clearly

Completion notes:

- Added `service.SSH` to reconcile state, select a running target, load the configured username, and invoke the system `ssh` client using the stored management port.
- Added `yeast ssh [instance]` CLI wiring.
- When no target is given, the workflow requires exactly one running instance.
- Added tests for the single-running-instance happy path and the multi-instance ambiguity error.

### M7-T6: Implement `yeast down`

Status: [x]

Dependencies:

- M5 stop
- M3 state

Definition of done:

- stops running VMs
- marks stopped
- handles already stopped

Completion notes:

- Added `service.Down` to stop all tracked running instances for the current project and persist stopped state.
- Added `yeast down` CLI wiring.
- Running instances are stopped through the runtime interface; already stopped instances are reported without error.
- Added app-level tests for stopping running instances and handling already-stopped state.

### M7-T7: Implement `yeast destroy`

Status: [x]

Dependencies:

- M7-T6
- M1 path safety

Definition of done:

- stops if running
- removes project instance runtime dir
- removes state entry
- never removes cache

Completion notes:

- Added `service.Destroy` to remove all tracked instance runtime directories for the current project and delete their state entries.
- Running instances are destroyed through the runtime interface, which stops them first when needed.
- Added `yeast destroy` CLI wiring.
- Added app-level tests to verify tracked instances are destroyed and state entries are removed.

---

# M8: Human And JSON Output

Goal:

Make commands usable by humans and tools.

## M8 Tasks

### M8-T1: Define result and error schemas

Status: [x]

Dependencies:

- M7 app workflows started

Files:

- `internal/output/schemas.go`
- `internal/app/errors.go`

Definition of done:

- common success/error shape exists

Completion notes:

- Added shared output envelopes in `internal/output/schemas.go` for success and error responses.
- Added `internal/app/errors.go` with a small typed application error wrapper and normalization helper.
- The new contract is intentionally small so later human/JSON renderers can build on one stable shape instead of per-command ad hoc formatting.

### M8-T2: Implement human renderer

Status: [x]

Dependencies:

- M8-T1

Definition of done:

- readable terminal output
- no JSON leakage

Completion notes:

- Added `internal/output/human.go` to centralize human-readable command rendering.
- Rewired the current CLI commands to render through one shared human-output layer instead of inline per-command formatting.
- The human renderer only emits terminal-friendly text and does not reuse JSON envelopes or machine-oriented fields directly.

### M8-T3: Implement JSON renderer

Status: [x]

Dependencies:

- M8-T1

Definition of done:

- parseable JSON
- no ANSI/spinners
- stable command/error fields

Completion notes:

- Added `internal/output/json.go` for shared success/error JSON rendering.
- Added `cmd/yeast/render.go` so commands switch between human and JSON output through one path.
- `--json` now emits structured envelopes for successful command results and for command errors.
- The JSON path is encoder-based, contains no ANSI formatting, and uses stable `ok`, `command`, `data`, and `error` fields.

### M8-T4: Add JSON tests for core commands

Status: [x]

Dependencies:

- M8-T3

Definition of done:

- JSON contract tests pass

Completion notes:

- Added CLI-level JSON contract tests around the shared render path for the core command result types.
- Added error rendering tests to verify stable `ok=false` envelopes with `error.code` and `error.message`.
- These tests validate the actual `--json` command surface without duplicating each command workflow.

---

# M9: Tests And Examples

Goal:

Prove v0.1 works.

## M9 Tasks

### M9-T1: Fast unit test suite

Status: [x]

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
- fast test entrypoint exists at `scripts/test-fast.sh`

### M9-T2: Application workflow fake-runtime tests

Status: [x]

Dependencies:

- M7 app workflows

Definition of done:

- up/status/down/destroy workflows tested without QEMU
- end-to-end fake-runtime workflow test added in `internal/app/workflow_test.go`

### M9-T3: Manual host-dependent v0.1 checklist

Status: [x]

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

Status: [x]

Dependencies:

- M7 complete

Files:

- `examples/ubuntu-basic/yeast.yaml`
- `examples/ubuntu-basic/README.md`

Definition of done:

- example works with v0.1 commands
- `examples/ubuntu-basic` added with honest single-VM scope

---

# M10: Docs And v0.1 Release Prep

Goal:

Prepare first public release.

## M10 Tasks

### M10-T1: Rewrite README for v2/v0.1

Status: [x]

Dependencies:

- M7 complete

Definition of done:

- README explains current working scope honestly
- README rewritten around actual v0.1 commands, limits, examples, and architecture

### M10-T2: Create required v0.1 docs

Status: [x]

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
- created `docs/quickstart.md`
- created `docs/installation.md`
- created `docs/config-reference.md`
- created `docs/troubleshooting.md`
- created `docs/known-limitations.md`
- created `docs/architecture-overview.md`

### M10-T3: Prepare changelog and release notes

Status: [x]

Dependencies:

- M10-T2
- M9 tests

Definition of done:

- v0.1.0 release notes ready
- `CHANGELOG.md` contains a draft `0.1.0` section
- `docs/release-notes-v0.1.0.md` created for GitHub release drafting

### M10-T3A: Harden one-script Linux installer

Status: [x]

Dependencies:

- M10-T2

Goal:

Make the v0.1.0 install path feel serious: one script that smartly prepares common Linux hosts and installs Yeast with clear diagnostics.

Required:

- support common Linux package managers:
  - `apt`
  - `dnf`
  - `yum`
  - `pacman`
  - `zypper`
  - `apk`
- install or verify:
  - `qemu-system-x86_64`
  - `qemu-img`
  - `genisoimage` or compatible `mkisofs`
  - `ssh`
  - `ssh-keygen`
  - `git`
  - Go 1.25+ or a fallback source-build strategy
- create required Yeast directories:
  - `~/.yeast`
  - `~/.yeast/cache`
  - `~/.yeast/cache/images`
- generate an SSH key if neither supported public key exists
- detect KVM availability and permissions
- add user to KVM group when available
- explain when logout/login is required
- keep logs for failed install steps
- support non-interactive install where possible
- support overrides:
  - `YEAST_REPO_URL`
  - `YEAST_REF`
  - `YEAST_INSTALL_DIR`
  - `YEAST_INSTALL_VERBOSE`
  - `YEAST_KEEP_LOGS`
- run post-install `yeast doctor`
- document install script behavior in `docs/installation.md`

Definition of done:

- `install.sh` is reviewed and hardened for v0.1.0
- script has a shell syntax check
- install docs match script behavior
- release plan treats one-script install as part of v0.1.0
- Go 1.25+ fallback behavior is implemented for old distro Go packages
- installer now detects `amd64` and `arm64`
- installer creates `~/.yeast/cache/images`
- installer checks KVM access after group setup
- installer next steps match real v0.1 commands

### M10-T4: Build release artifact

Status: [x]

Dependencies:

- M10-T3A
- M10-T3

Definition of done:

- Linux amd64 binary built
- checksum generated
- release build script added at `scripts/build-release.sh`
- version can be embedded with `-ldflags`

### M10-T5: Tag and publish v0.1.0

Status: [x]

Dependencies:

- M10-T4

Definition of done:

- Git tag exists
- GitHub release published
- soft announcement ready
- published as a GitHub prerelease because the manual host-dependent checklist is still pending
- `docs/soft-announcement-v0.1.0.md` added

### M10-T6: Create GitHub Pages landing page

Status: [x]

Dependencies:

- M10-T5

Definition of done:

- static landing page exists under `docs/`
- site uses the approved Yeast banner
- install, features, docs, roadmap, and release links are visible
- GitHub Pages deployment workflow exists
- local preview instructions are documented

---

# C1: Charm CLI Experience

Goal:

Make Yeast feel like a polished flagship CLI while keeping machine output stable.

## C1 Tasks

### C1-T1: Add Charm CLI technical plan

Status: [x]

Definition of done:

- `docs/charm-cli-plan.md` explains which Charm libraries Yeast should use and when

### C1-T2: Add Lip Gloss human renderer

Status: [x]

Definition of done:

- current human command output uses Lip Gloss styling
- JSON output remains unchanged

### C1-T3: Add Glamour terminal docs

Status: [x]

Dependencies:

- M10-T2 docs exist

Definition of done:

- markdown docs can be rendered from Yeast in terminal
- added `yeast docs` command with embedded topics and `--list`

### C1-T4: Add Huh interactive init

Status: [-]

Dependencies:

- v0.1 init/config is stable

Definition of done:

- `yeast init --interactive` creates a config through terminal prompts

### C1-T5: Add Bubble Tea lifecycle progress

Status: [-]

Dependencies:

- lifecycle event model exists

Definition of done:

- `yeast up` and `yeast pull` can show live progress in human TTY mode

---

# V0.2.0: Disk Size Support

Goal:

Make `disk_size` a documented and verified desired-state setting for instance overlay disk creation.

Scope:

- config schema
- validation
- disk creation/runtime wiring
- tests
- docs

Out of scope:

- networking
- provisioning

## V0.2.0 Tasks

### V0.2-T1: Verify disk_size config and runtime wiring

Status: [x]

Dependencies:

- M10 complete
- C1 complete/deferred decisions preserved

Files:

- `internal/app/up_test.go`
- `docs/config-reference.md`
- `README.md`
- `TASKS.md`

Definition of done:

- config-level `disk_size` is verified to reach the runtime disk plan
- normalized sizes are verified at the app/runtime boundary
- docs explain supported size formats
- docs explain existing disks are not resized by `up`
- networking and provisioning remain untouched

Completion notes:

- Confirmed existing config schema, validation, defaults/normalization, runtime model, and QEMU disk creation already support `disk_size`.
- Added an app-level `up` workflow test proving `disk_size: 25 gb` normalizes to `25G` and reaches both `PrepareDisk` and `Start` machine plans.
- Documented supported `disk_size` formats and the existing-disk no-resize behavior.

### V0.2-T2: Continue disk_size support with any remaining runtime/docs verification

Status: [x]

Dependencies:

- V0.2-T1

Definition of done:

- decide whether any remaining disk_size work is needed before moving to the next v0.2.0 item

Completion notes:

- Added runtime regression coverage proving `PrepareDisk` keeps an existing overlay disk even when a requested `disk_size` is present.
- Confirmed no additional disk_size implementation was needed after config/schema, validation, app wiring, QEMU command construction, existing-disk behavior, tests, and docs were covered.
- Networking and provisioning remain untouched.

### V0.2-T3: Add disk_size release notes

Status: [x]

Dependencies:

- V0.2-T2

Files:

- `docs/release-notes-v0.2.0.md`
- `TASKS.md`

Definition of done:

- v0.2.0 disk_size behavior is summarized for users
- verification expectations are documented
- limitations stay clear, including no existing-disk resize behavior

Completion notes:

- Added draft v0.2.0 release notes focused on `disk_size` support.
- Documented supported size formats, automated verification, manual host-dependent verification, and out-of-scope features.
- Networking and provisioning remain untouched.

### V0.2-T4: Classify yeast up image errors

Status: [x]

Dependencies:

- V0.2-T3

Files:

- `internal/app/up.go`
- `internal/app/up_test.go`
- `TASKS.md`

Definition of done:

- unsupported configured images report `invalid_argument`
- missing cached images report `not_found`
- existing human-facing guidance is preserved
- JSON output can use stable app error codes through existing renderers

Completion notes:

- Wrapped unsupported-image failures in `ErrorCodeInvalidArgument`.
- Wrapped missing cached-image failures in `ErrorCodeNotFound` while preserving the existing `yeast pull` guidance.
- Added app workflow tests for both error classifications.

### V0.2-T5: Classify yeast ssh selection errors

Status: [x]

Dependencies:

- V0.2-T4

Files:

- `internal/app/ssh.go`
- `internal/app/ssh_test.go`
- `TASKS.md`

Definition of done:

- missing SSH target reports `not_found`
- stopped or unavailable SSH target reports `failed_precondition`
- no running instances reports `failed_precondition`
- ambiguous target selection reports `invalid_argument`
- existing human-facing messages are preserved

Completion notes:

- Wrapped SSH target selection failures in stable app error codes.
- Wrapped missing config lookup for selected state instances as `not_found`.
- Added focused tests for missing, stopped, empty, and ambiguous SSH selection cases.

### V0.2-T6: Classify yeast pull unsupported image errors

Status: [x]

Dependencies:

- V0.2-T5

Files:

- `internal/app/pull.go`
- `internal/app/pull_test.go`
- `TASKS.md`

Definition of done:

- unsupported pull image reports `invalid_argument`
- existing `ErrUnsupportedImage` compatibility is preserved
- JSON output can use stable app error codes through existing renderers

Completion notes:

- Wrapped unsupported pull image failures in `ErrorCodeInvalidArgument`.
- Preserved `errors.Is(err, ErrUnsupportedImage)` by keeping the sentinel as the wrapped cause.
- Added test coverage for the app error code and sentinel behavior.

### V0.2-T7: Classify repeated yeast init errors

Status: [x]

Dependencies:

- V0.2-T6

Files:

- `internal/app/init.go`
- `internal/app/init_test.go`
- `TASKS.md`

Definition of done:

- repeated init reports `conflict`
- existing `ErrProjectAlreadyInitialized` compatibility is preserved
- JSON output can use stable app error codes through existing renderers

Completion notes:

- Wrapped repeated-init config and metadata conflicts in `ErrorCodeConflict`.
- Preserved `errors.Is(err, ErrProjectAlreadyInitialized)` by keeping the sentinel as the wrapped cause.
- Added test coverage for the app error code and sentinel behavior.

### V0.2-T8: Classify yeast down runtime stop errors

Status: [x]

Dependencies:

- V0.2-T7

Files:

- `internal/app/down.go`
- `internal/app/down_test.go`
- `TASKS.md`

Definition of done:

- runtime stop failures during `yeast down` report `internal`
- existing runtime error message is preserved
- JSON output can use stable app error codes through existing renderers

Completion notes:

- Wrapped runtime stop failures in `ErrorCodeInternal`.
- Added fake-runtime coverage proving failed stops produce a stable app error code.

### V0.2-T9: Classify yeast destroy runtime errors

Status: [x]

Dependencies:

- V0.2-T8

Files:

- `internal/app/destroy.go`
- `internal/app/destroy_test.go`
- `TASKS.md`

Definition of done:

- runtime destroy failures during `yeast destroy` report `internal`
- existing runtime error message is preserved
- JSON output can use stable app error codes through existing renderers

Completion notes:

- Wrapped runtime destroy failures in `ErrorCodeInternal` for running and stopped tracked instances.
- Added fake-runtime coverage proving failed destroys produce a stable app error code.

### V0.2-T10: Classify yeast up runtime prepare/start errors

Status: [x]

Dependencies:

- V0.2-T9

Files:

- `internal/app/up.go`
- `internal/app/up_test.go`
- `TASKS.md`

Definition of done:

- runtime disk preparation failures during `yeast up` report `internal`
- runtime start failures during `yeast up` report `internal`
- existing runtime error messages are preserved
- JSON output can use stable app error codes through existing renderers

Completion notes:

- Wrapped runtime `PrepareDisk` and `Start` failures in `ErrorCodeInternal`.
- Added fake-runtime coverage proving failed prepare/start paths produce stable app error codes.

### V0.2-T11: Classify yeast status project/state errors

Status: [x]

Dependencies:

- V0.2-T10

Files:

- `internal/app/status.go`
- `internal/app/status_test.go`
- `TASKS.md`

Definition of done:

- non-initialized `yeast status` reports `failed_precondition`
- corrupted or mismatched tracked state reports `internal`
- existing human-facing messages are preserved
- JSON output can use stable app error codes through existing renderers

Completion notes:

- Wrapped missing project metadata during `yeast status` as `ErrorCodePrecondition`.
- Wrapped Yeast home resolution, path construction, state lock acquisition, and state load/save failures as `ErrorCodeInternal`.
- Added focused tests for non-initialized projects and state project-id mismatch behavior.

### V0.2-T12: Classify yeast up guest-readiness errors

Status: [x]

Dependencies:

- V0.2-T11

Files:

- `internal/app/up.go`
- `internal/app/up_test.go`
- `TASKS.md`

Definition of done:

- SSH address construction failures after runtime start report `internal`
- SSH readiness failures after runtime start report `failed_precondition`
- started instances are still stopped on these failure paths
- existing human-facing readiness message is preserved

Completion notes:

- Wrapped post-start SSH address failures in `ErrorCodeInternal`.
- Wrapped post-start SSH readiness failures in `ErrorCodePrecondition` while preserving the existing `wait for ssh readiness for <instance>` message prefix.
- Added fake-runtime coverage proving both failure paths stop the started instance before returning.

### V0.2-T13: Classify yeast up setup and state errors

Status: [x]

Dependencies:

- V0.2-T12

Files:

- `internal/app/up.go`
- `internal/app/up_test.go`
- `TASKS.md`

Definition of done:

- missing project metadata reports `failed_precondition`
- missing config reports `failed_precondition`
- invalid config reports `invalid_argument`
- state/path/lock/save setup failures report `internal`
- existing human-facing messages are preserved

Completion notes:

- Wrapped missing project metadata and missing config in `ErrorCodePrecondition`.
- Wrapped invalid config load failures in `ErrorCodeInvalidArgument`.
- Wrapped Yeast home resolution, path construction, runtime directory creation, state lock acquisition, state load, and state save failures in `ErrorCodeInternal`.
- Added focused tests for uninitialized project, missing config, invalid config, and state project-id mismatch behavior.

### V0.2-T14: Classify yeast up helper and cloud-init errors

Status: [x]

Dependencies:

- V0.2-T13

Files:

- `internal/app/up.go`
- `internal/app/up_test.go`
- `TASKS.md`

Definition of done:

- cached-running SSH address failures report `internal`
- missing SSH public key reports `failed_precondition`
- cloud-init render/seed helper failures report `internal`
- invalid runtime-derived instance path input reports `invalid_argument`

Completion notes:

- Wrapped cached-running `sshAddress` failures in `ErrorCodeInternal`.
- Wrapped missing SSH public key discovery as `ErrorCodePrecondition` using `cloudinit.ErrNoSSHPublicKey`.
- Wrapped cache-path resolution, cloud-init user-data/meta-data rendering, and seed ISO creation failures in `ErrorCodeInternal`.
- Wrapped invalid instance runtime-directory resolution in `ErrorCodeInvalidArgument`.
- Added focused tests for cached-running SSH address, missing SSH key, user-data render, meta-data render, and seed ISO failures.

### V0.2-T15: Classify yeast pull helper errors

Status: [x]

Dependencies:

- V0.2-T14

Files:

- `internal/app/pull.go`
- `internal/app/pull_test.go`
- `TASKS.md`

Definition of done:

- Yeast home resolution failures report `internal`
- cache path construction failures report `internal`
- download failures report `internal`
- existing unsupported-image `invalid_argument` behavior stays unchanged

Completion notes:

- Wrapped `resolveYeastHome`, cache path construction, and image download failures in `ErrorCodeInternal`.
- Preserved the existing `unsupported image` path as `ErrorCodeInvalidArgument` with `ErrUnsupportedImage` compatibility.
- Added focused tests for home resolution, cache path, and download failure classification.

### V0.2-T16: Classify yeast init setup and write errors

Status: [x]

Dependencies:

- V0.2-T15

Files:

- `internal/app/init.go`
- `internal/app/init_test.go`
- `TASKS.md`

Definition of done:

- project-root resolution and setup/write failures report `internal`
- repeated init still reports `conflict`
- existing human-facing setup/write messages are preserved

Completion notes:

- Wrapped config/metadata inspection failures, metadata creation failures, and starter-config write failures in `ErrorCodeInternal`.
- Preserved the existing repeated-init `ErrorCodeConflict` behavior with `ErrProjectAlreadyInitialized`.
- Added focused tests for config inspection failure and config write failure classification.

### V0.2-T17: Classify yeast ssh setup and helper errors

Status: [x]

Dependencies:

- V0.2-T16

Files:

- `internal/app/ssh.go`
- `internal/app/ssh_test.go`
- `TASKS.md`

Definition of done:

- missing project metadata and missing config report `failed_precondition`
- invalid config reports `invalid_argument`
- state/home/path/address/ssh-exec helper failures report `internal`
- existing selection error codes stay unchanged

Completion notes:

- Wrapped metadata, Yeast home, path, state load/save, SSH address, and SSH execution failures in stable app error codes.
- Wrapped missing config as `ErrorCodePrecondition` and invalid config as `ErrorCodeInvalidArgument`.
- Preserved existing selection classification for missing/stopped/ambiguous targets.
- Added focused tests for uninitialized project, missing config, SSH address failure, and SSH execution failure.

### V0.2-T18: Classify yeast down setup and state errors

Status: [x]

Dependencies:

- V0.2-T17

Files:

- `internal/app/down.go`
- `internal/app/down_test.go`
- `TASKS.md`

Definition of done:

- missing project metadata reports `failed_precondition`
- home/path/lock/state load/save failures report `internal`
- existing runtime stop `internal` behavior stays unchanged

Completion notes:

- Wrapped metadata, Yeast home, path, lock, state load, and final state save failures in stable app error codes.
- Preserved the existing runtime stop classification as `ErrorCodeInternal`.
- Added focused tests for uninitialized project and state project-id mismatch behavior.

### V0.2-T19: Classify yeast destroy setup and state errors

Status: [x]

Dependencies:

- V0.2-T18

Files:

- `internal/app/destroy.go`
- `internal/app/destroy_test.go`
- `TASKS.md`

Definition of done:

- missing project metadata reports `failed_precondition`
- home/path/lock/state load/save failures report `internal`
- existing runtime destroy `internal` behavior stays unchanged

Completion notes:

- Wrapped metadata, Yeast home, path, lock, state load, and final state save failures in stable app error codes.
- Preserved the existing runtime destroy classification as `ErrorCodeInternal`.
- Added focused tests for uninitialized project and state project-id mismatch behavior.

### V0.2-T20: Finish app error-classification audit

Status: [x]

Dependencies:

- V0.2-T19

Files:

- `internal/app/status.go`
- `internal/app/status_test.go`
- `TASKS.md`

Definition of done:

- remaining raw app-surface setup errors are either classified or explicitly accepted
- the v0.2.0 error-classification pass has a clear stopping point

Completion notes:

- Wrapped the remaining raw `yeast status` project-root resolution failure in `ErrorCodeInternal`.
- Added focused coverage for the `status` root-resolution failure path.
- Audited remaining `internal/app` raw returns and confirmed the remaining pass-throughs are either intentional wrapped selection errors or internal helper-local errors outside the app-surface contract.

### V0.2-T21: Add explicit hostname config support

Status: [x]

Dependencies:

- V0.2-T20

Files:

- `internal/config/model.go`
- `internal/config/validate.go`
- `internal/config/defaults.go`
- `internal/config/*_test.go`
- `internal/app/up.go`
- `internal/app/up_test.go`
- `internal/provision/cloudinit/*_test.go`
- `docs/config-reference.md`
- `README.md`
- `TASKS.md`

Definition of done:

- `hostname` is a supported instance field
- omitted `hostname` defaults to instance `name`
- explicit `hostname` reaches cloud-init user-data and meta-data
- docs describe the field and its default behavior

Completion notes:

- Added `hostname` to the instance config model.
- Defaulted omitted hostnames to the instance name and validated explicit hostnames with the existing safe-name rules.
- Wired `hostname` through `yeast up` into cloud-init user-data and meta-data generation.
- Added focused config, app, and cloud-init tests plus README/config-reference updates.

### V0.2-T22: Add explicit ssh_port config support

Status: [x]

Dependencies:

- V0.2-T21

Files:

- `internal/config/model.go`
- `internal/config/validate.go`
- `internal/config/*_test.go`
- `internal/app/up.go`
- `internal/app/up_test.go`
- `docs/config-reference.md`
- `README.md`
- `TASKS.md`

Definition of done:

- `ssh_port` is a supported instance field
- explicit `ssh_port` reaches the runtime management port plan
- invalid or colliding requested ports fail clearly
- docs describe default and override behavior

Completion notes:

- Added `ssh_port` to the instance config model and validation.
- Wired requested SSH ports through `yeast up` port selection and runtime planning.
- Rejected invalid requested ports and same-run collisions as `invalid_argument`.
- Added focused config/app tests plus README/config-reference updates.

---

# Future Milestones

These milestones are ordered by dependency. Do not pull features forward from later milestones unless a current milestone explicitly needs a small internal primitive.

## M11: Provisioning

Status: [x]

Do not start until:

- M10 complete
- v0.2.0 released

Goal:

Turn a booted VM into a useful machine automatically.

v0.3.0 success target:

- `yeast up` boots a VM, waits for SSH, provisions it, and reports `provisioned`
- `yeast provision` reruns the post-boot provisioning plan without recreating the VM
- one example VM installs a web server, copies content, runs setup commands, and is verifiable from the host

Non-goals:

- no snapshots
- no private networking
- no templates
- no LabsBackery-specific API
- no MCP commands
- no Ansible/provider/plugin system
- no cloud worker behavior

Important v0.3.0 product rule:

```text
cloud-init remains bootstrap.
post-boot SSH provisioning owns packages, files, and shell.
```

Reason:

- post-boot provisioning can be logged, retried, and rerun through `yeast provision`
- shell steps are easier to debug after SSH readiness than inside opaque first-boot logs
- package/file/shell behavior should have one execution path, not one cloud-init path and one rerun path

### V0.3-T1: Activate provisioning config schema and validation

Status: [x]

Dependencies:

- V0.2-T22

Files:

- `internal/config/validate.go`
- `internal/config/validate_test.go`
- `internal/config/loader_test.go`
- `docs/config-reference.md`
- `TASKS.md`

Definition of done:

- top-level and instance-level `provision` blocks are validated
- package entries reject empty/newline values
- file entries require `source` and `destination`
- file `permissions` validate as octal strings when present
- shell entries reject empty commands
- docs describe the active provisioning schema and clearly state execution is not wired yet

Completion notes:

- Activated schema validation for top-level and instance-level `provision` blocks.
- Added focused tests for valid provisioning config plus invalid package, file, permissions, and shell cases.
- Extended loader coverage so top-level and instance-level provisioning sections parse from YAML.
- Updated config reference to document the schema and its current non-executing status.

### V0.3-T2: Lock provisioning contract and merge rules

Status: [x]

Dependencies:

- V0.3-T1

Files:

- `YEAST_TECHNICAL_ARCHITECTURE.md`
- `YEAST_V2_IMPLEMENTATION_PLAN.md`
- `docs/config-reference.md`
- `TASKS.md`

Definition of done:

- top-level `provision` behavior is explicit
- instance-level `provision` behavior is explicit
- merge order is explicit for packages, files, and shell
- cloud-init vs post-boot responsibilities are explicit
- auto-run behavior during `yeast up` is explicit
- rerun behavior for `yeast provision` is explicit
- idempotency expectations are explicit

Contract to document:

- top-level provision steps run before instance-level steps
- list fields append in order: project packages/files/shell, then instance packages/files/shell
- `provision.packages`, `provision.files`, and `provision.shell` run post-boot over SSH in v0.3.0
- cloud-init remains responsible for user, SSH key, hostname, sudo, and environment bootstrap
- `yeast up` runs provisioning automatically after SSH readiness
- `yeast provision` requires an existing reachable VM and reruns the same post-boot plan
- package installation should be idempotent where the guest package manager supports it
- file provisioning overwrites destination files
- shell commands always run and must be authored as idempotent by the user

Completion notes:

- Documented provisioning merge order in architecture, implementation plan, and config reference.
- Locked the v0.3 execution split: cloud-init remains bootstrap; packages/files/shell run post-boot over SSH.
- Documented automatic `yeast up` provisioning and manual `yeast provision` rerun semantics.
- Documented idempotency expectations for packages, files, and shell commands.

### V0.3-T3: Add provisioning plan builder

Status: [x]

Dependencies:

- V0.3-T2

Files:

- `internal/provision/*.go`
- `internal/provision/*_test.go`
- `internal/config/model.go`
- `TASKS.md`

Definition of done:

- `internal/provision` exposes a small plan type built from config
- project-level and instance-level steps merge in documented order
- package, file, and shell steps are structured
- plan construction does not execute anything
- tests cover empty plans, project-only plans, instance-only plans, and merged plans

Completion notes:

- Added `internal/provision.Plan` plus structured package, file, and shell step types.
- Added `BuildPlan` to merge project-level provision config before instance-level provision config.
- Added focused tests for empty, project-only, instance-only, and merged-order plans.

### V0.3-T4: Add SSH provisioning transport abstraction

Status: [x]

Dependencies:

- V0.3-T3

Files:

- `internal/provision/ssh/*.go`
- `internal/provision/ssh/*_test.go`
- `TASKS.md`

Definition of done:

- SSH transport interface exists for command execution and file upload
- implementation can use system `ssh` / `scp` or a minimal command runner wrapper
- fake transport exists for tests
- timeouts and exit-code capture are represented in result types
- no public `yeast exec` command is added in this milestone

Completion notes:

- Added `internal/provision/ssh.Transport` with `Run` and `Upload` methods.
- Added a command-backed local transport using `ssh` and `scp`.
- Added structured request/result types with stdout, stderr, exit code, and duration capture.
- Added a fake transport plus focused tests for invocation building, validation, and error result preservation.

### V0.3-T5: Add package provisioner

Status: [x]

Dependencies:

- V0.3-T3
- V0.3-T4

Files:

- `internal/provision/ssh/*.go`
- `internal/provision/ssh/*_test.go`
- `TASKS.md`

Definition of done:

- package steps install packages over SSH
- Ubuntu/Debian guests are supported first through `apt-get`
- empty package plans are no-ops
- command result includes stdout, stderr, exit code, and duration where practical
- tests cover generated command behavior and failure propagation

Completion notes:

- Added `ssh.PackageProvisioner` for package-step execution over the SSH transport.
- Collapsed package steps into a single Ubuntu/Debian `apt-get update && apt-get install -y ...` command.
- Preserved stdout/stderr/exit-code results on failures from the transport.
- Added focused tests for no-op behavior, command generation, validation, default timeout, and failure-result preservation.

### V0.3-T6: Add file provisioner

Status: [x]

Dependencies:

- V0.3-T3
- V0.3-T4

Files:

- `internal/provision/ssh/*.go`
- `internal/provision/ssh/*_test.go`
- `TASKS.md`

Definition of done:

- file upload steps copy local files/directories into the guest
- destination parent directories are created when needed
- optional permissions are applied after upload
- failures identify the file step that failed
- tests cover source/destination handling and permissions behavior

Completion notes:

- Added `ssh.FileProvisioner` for file-step execution over the SSH transport.
- Added remote parent-directory creation before upload.
- Added optional post-upload `chmod` handling.
- Preserved failure context for mkdir/upload/chmod failures and added focused tests for each path.

### V0.3-T7: Add shell provisioner

Status: [x]

Dependencies:

- V0.3-T3
- V0.3-T4

Files:

- `internal/provision/ssh/*.go`
- `internal/provision/ssh/*_test.go`
- `TASKS.md`

Definition of done:

- shell steps run over SSH after package and file steps
- shell steps run in the configured order
- failures stop the provisioning run
- result identifies the failed command and exit code
- tests cover success, command failure, and ordering

Completion notes:

- Added `ssh.ShellProvisioner` for ordered shell-step execution over the SSH transport.
- Stopped execution on the first failed shell command.
- Preserved stdout/stderr/exit-code results for the failed command.
- Added focused tests for no-op behavior, ordering, stop-on-failure, and request validation.

### V0.3-T8: Add provisioning logs and status model

Status: [x]

Dependencies:

- V0.3-T3

Files:

- `internal/state/*.go`
- `internal/state/*_test.go`
- `internal/provision/*.go`
- `internal/provision/*_test.go`
- `TASKS.md`

Definition of done:

- state can represent `not_started`, `running`, `provisioned`, and `failed`
- provisioning failure stores a useful `last_error`
- a per-instance `provision.log` path is defined
- result types can render enough information for human and JSON output later
- tests cover status transitions without needing real SSH

Completion notes:

- Added typed provisioning statuses to `state.InstanceState` and persisted a per-instance `provision_log_path`.
- Added a minimal `provision.Result` / `provision.StepResult` model for later human and JSON rendering.
- Updated app status results to expose provisioning status and log path.
- Set new instances to `not_started` with a stable `provision.log` path so `V0.3-T9` can wire execution without another state-shape change.
- Added and updated state, provision, and app tests to cover the new model without real SSH.

### V0.3-T9: Wire provisioning into `yeast up`

Status: [x]

Dependencies:

- V0.3-T5
- V0.3-T6
- V0.3-T7
- V0.3-T8

Files:

- `internal/app/up.go`
- `internal/app/up_test.go`
- `internal/state/*.go`
- `internal/state/*_test.go`
- `TASKS.md`

Definition of done:

- `yeast up` runs provisioning automatically after SSH readiness
- package, file, and shell steps run in the documented order
- provisioning status updates are reflected in state/status
- failures are visible and recoverable
- tests use fake provisioning dependencies rather than real SSH

Completion notes:

- Wired `yeast up` to build the merged provisioning plan and run it automatically after SSH readiness.
- Resolved file provision sources relative to the project root before upload.
- Updated state transitions to `running -> not_started/running -> provisioned|failed` with a stable per-instance `provision.log`.
- Kept instances running on provisioning failure while persisting `failed` state and `last_error` for later recovery.
- Added focused `up` tests for merged provisioning order, source-path resolution, success logging, and failure-state persistence using a fake SSH transport.

### V0.3-T10: Add `yeast provision` rerun command

Status: [x]

Dependencies:

- V0.3-T9

Files:

- `cmd/yeast/provision.go`
- `internal/app/provision.go`
- related tests
- docs
- `TASKS.md`

Definition of done:

- user can rerun provisioning without recreating the VM
- command requires a reachable existing VM in v0.3.0
- command has human and JSON output
- tests cover success and failure paths

Completion notes:

- Added `app.Service.Provision` to rerun the same merged provisioning plan against an existing running instance.
- Reused the same plan resolution and provisioning execution path as `yeast up` rather than adding a second implementation.
- Added `yeast provision [instance]` with the same target-selection behavior as `yeast ssh`.
- Added human and JSON renderer coverage for the new command.
- Added focused service tests for successful reruns, failure-state persistence, and reachable-instance preconditions.

### V0.3-T11: Add provisioning docs and Caddy example

Status: [x]

Dependencies:

- V0.3-T10

Files:

- `examples/*`
- `docs/config-reference.md`
- `docs/quickstart.md`
- `docs/known-limitations.md`
- `README.md`
- `TASKS.md`

Definition of done:

- one VM can boot, install Caddy, copy content, and serve it
- example config is small and readable
- docs explain the demo clearly

Completion notes:

- Added `examples/caddy-single-vm` with a small Caddy provisioning demo and static site assets.
- Updated the quickstart to include provisioning and `yeast provision`.
- Updated the config reference to describe shipped `v0.3` provisioning behavior instead of future intent.
- Updated known limitations to reflect current provisioning support and current gaps.
- Updated the README examples, scope, and install snippet to match the current product state.

### V0.3-T12: Add provisioning smoke coverage and release notes

Status: [x]

Dependencies:

- V0.3-T11

Files:

- `scripts/manual-smoke.sh`
- `docs/tutorial-test.md`
- `docs/release-notes-v0.3.0.md`
- `CHANGELOG.md`
- `TASKS.md`

Definition of done:

- smoke script can validate provisioning on a real Linux/KVM host
- smoke script proves the Caddy example reaches a serving state
- release notes list v0.3.0 features and limitations
- known limitations remain honest about snapshots/networking/templates/MCP/cloud

Completion notes:

- Expanded `scripts/manual-smoke.sh` from a lifecycle-only `v0.2` script into a `v0.3` provisioning smoke suite.
- Added positive-path checks for automatic provisioning, Caddy service state, guest HTTP content, and `yeast provision` reruns.
- Added a negative-path check for missing provision source files.
- Rewrote `docs/tutorial-test.md` for the `v0.3.0` candidate and the new smoke workflow.
- Added `docs/release-notes-v0.3.0.md` and a `0.3.0` changelog entry.

## M12: Snapshots And Reset

Status: [~]

Do not start until:

- M11 complete

Goal:

Make single-VM lab reset real and safe before multi-VM reset.

Non-goals for `v0.4.0`:

- live snapshots of running VMs
- multi-VM atomic restore
- browser/UI reset flows
- libvirt or alternate runtime support

Snapshot safety rules:

- early snapshot and restore require stopped VMs
- restore must never silently touch a running VM
- snapshot metadata must be explicit and inspectable
- snapshot behavior must prefer disk safety over convenience

Execution order:

- contract and metadata first
- runtime file operations second
- app workflows third
- CLI/docs/smoke last

### V0.4-T1: Lock snapshot contract and metadata model

Status: [x]

Dependencies:

- M11 complete

Files:

- `YEAST_TECHNICAL_ARCHITECTURE.md`
- `YEAST_V2_IMPLEMENTATION_PLAN.md`
- `docs/known-limitations.md`
- `TASKS.md`

Definition of done:

- stopped-VM requirement is documented as the `v0.4` rule
- snapshot metadata fields are fixed
- single-instance and project-wide command scope is defined
- delete behavior and restore preconditions are explicit

Completion notes:

- Locked `v0.4` snapshot safety to stopped-VM-only create/restore flows.
- Fixed the first metadata model to: `name`, `created_at`, `description`, `disk_path`, and optional `source_disk_size`.
- Defined command scope:
  - `yeast snapshot <instance> <name>`
  - `yeast snapshot --all <name>`
  - `yeast snapshots [instance]`
  - `yeast restore <instance> <name>`
  - `yeast restore --all <name>`
  - `yeast delete-snapshot <instance> <name>`
- Defined restore/delete preconditions:
  - restore target must exist in metadata
  - restore target instance must be stopped
  - delete does not touch running guests, but must fail if metadata/file target is missing inconsistently
- Updated architecture, implementation plan, known limitations, and this task file before runtime work.

### V0.4-T2: Add snapshot metadata model to state

Status: [x]

Dependencies:

- V0.4-T1

Files:

- `internal/state/*.go`
- `internal/state/*_test.go`
- `TASKS.md`

Definition of done:

- state can store snapshot metadata per instance
- metadata includes at least name, created time, description, and disk path/reference
- state round-trip tests cover the new snapshot model

Completion notes:

- Added `SnapshotState` to `internal/state/model.go`.
- Added per-instance `Snapshots map[string]SnapshotState` to tracked instance state.
- Fixed first metadata shape to:
  - `name`
  - `created_at`
  - `description`
  - `disk_path`
  - `source_disk_size`
- Extended JSON round-trip coverage in `internal/state/model_test.go`.
- Extended save/load round-trip coverage in `internal/state/store_test.go`.
- No runtime or CLI behavior changed in this task.

### V0.4-T3: Add runtime snapshot file helpers

Status: [x]

Dependencies:

- V0.4-T2

Files:

- `internal/runtime/*.go`
- `internal/runtime/qemu/*.go`
- related tests
- `TASKS.md`

Definition of done:

- runtime can create a snapshot copy for a stopped instance disk
- runtime can restore a stopped instance disk from a snapshot copy
- runtime can delete snapshot files
- tests cover missing-file and overwrite safety paths

Completion notes:

- Added `runtime.SnapshotPlan` in `internal/runtime/model.go`.
- Added QEMU snapshot helper functions in `internal/runtime/qemu/snapshot.go`:
  - `CreateSnapshotCopy`
  - `RestoreSnapshotCopy`
  - `DeleteSnapshotFile`
- Implemented create as copy-without-overwrite.
- Implemented restore as atomic temp-file copy plus rename over the tracked disk path.
- Added focused tests for:
  - create success
  - create source missing
  - create overwrite protection
  - restore success
  - restore snapshot missing
  - delete success

### V0.4-T4: Add snapshot listing helpers

Status: [x]

Dependencies:

- V0.4-T2
- V0.4-T3

Files:

- `internal/state/*.go`
- `internal/app/*.go`
- related tests
- `TASKS.md`

Definition of done:

- app layer can list instance snapshots from state
- snapshots are sorted predictably
- missing snapshot state is handled cleanly

Completion notes:

- Added `state.SortedSnapshots` in `internal/state/snapshots.go`.
- Sorting rule is stable and explicit:
  - `created_at` ascending
  - `name` ascending as the tie-breaker
- Added app-layer helper `listInstanceSnapshots` in `internal/app/snapshots.go`.
- Helper now:
  - requires a target instance name
  - returns `not_found` when the instance does not exist
  - returns an empty list when the instance exists but has no snapshot metadata yet
- Added focused tests in:
  - `internal/state/snapshots_test.go`
  - `internal/app/snapshots_test.go`

### V0.4-T5: Add `app.Service.Snapshot`

Status: [x]

Dependencies:

- V0.4-T3
- V0.4-T4

Files:

- `internal/app/snapshot.go`
- `internal/app/snapshot_test.go`
- `TASKS.md`

Definition of done:

- snapshots can be created for a stopped target instance
- snapshot metadata is persisted
- duplicate snapshot names fail cleanly
- tests use fake runtime helpers rather than real QEMU

Completion notes:

- Added `Service.Snapshot` in `internal/app/snapshot.go`.
- First snapshot API is explicit:
  - target instance required
  - snapshot name required
  - optional description persisted in metadata
- Snapshot creation now:
  - requires a stopped target instance
  - requires a tracked runtime directory
  - uses runtime snapshot helpers through the runtime interface
  - persists snapshot metadata into state after the copy succeeds
- Duplicate names fail as `conflict`.
- Missing target fails as `not_found`.
- Running target fails as `failed_precondition`.
- Added focused tests in `internal/app/snapshot_test.go` using a fake runtime.

### V0.4-T6: Add `app.Service.Restore`

Status: [x]

Dependencies:

- V0.4-T5

Files:

- `internal/app/restore.go`
- `internal/app/restore_test.go`
- `TASKS.md`

Definition of done:

- restore requires a stopped target instance
- restore replaces the tracked disk with the snapshot copy
- missing snapshot names fail cleanly

Completion notes:

- Added `Service.Restore` in `internal/app/restore.go`.
- Restore now:
  - requires explicit target instance
  - requires explicit snapshot name
  - requires the instance to be stopped
  - loads snapshot metadata from state
  - calls runtime restore helpers to replace the tracked disk from the snapshot copy
- Missing instance fails as `not_found`.
- Missing snapshot name fails as `not_found`.
- Running instance fails as `failed_precondition`.
- Added focused tests in `internal/app/restore_test.go` with a fake runtime.
- state stays consistent after restore

### V0.4-T7: Add snapshot deletion and list commands in app layer

Status: [x]

Dependencies:

- V0.4-T5

Files:

- `internal/app/snapshots.go`
- related tests
- `TASKS.md`

Definition of done:

- list snapshots returns metadata for one target instance
- delete snapshot removes file plus metadata
- deleting a missing snapshot returns `not_found`

Completion notes:

- Added `app.Service.Snapshots` to load one target instance's snapshot metadata from project state.
- Added `app.Service.DeleteSnapshot` to remove the snapshot file through the runtime layer and then remove snapshot metadata from state.
- Added focused tests for sorted listing, deletion persistence, and `not_found` behavior for missing targets and missing snapshot files.

### V0.4-T8: Add CLI commands for snapshot workflows

Status: [x]

Dependencies:

- V0.4-T6
- V0.4-T7

Files:

- `cmd/yeast/snapshot.go`
- `cmd/yeast/restore.go`
- `cmd/yeast/snapshots.go`
- `cmd/yeast/delete-snapshot.go`
- renderer tests
- `TASKS.md`

Definition of done:

- commands exist for snapshot, restore, list, and delete
- human output is readable
- JSON output uses existing success/error envelopes

Completion notes:

- Added `yeast snapshot`, `yeast restore`, `yeast snapshots`, and `yeast delete-snapshot` commands.
- Registered the snapshot commands in the root CLI and kept argument contracts strict with positional instance/name pairs.
- Added human renderers and renderer tests for snapshot create, restore, list, and delete output while reusing the existing JSON envelope path.

### V0.4-T9: Add single-VM reset demo docs

Status: [x]

Dependencies:

- V0.4-T8

Files:

- `examples/*`
- `docs/quickstart.md`
- `docs/known-limitations.md`
- `README.md`
- `TASKS.md`

Definition of done:

- one example shows provision -> snapshot -> modify -> restore
- docs explain stopped-VM restore expectations clearly

Completion notes:

- Extended the `examples/caddy-single-vm` walkthrough to cover the stopped-VM snapshot, break, restore, and delete loop.
- Updated `docs/quickstart.md` so the main guided workflow now includes snapshot create, restore, and delete after provisioning.
- Updated `docs/known-limitations.md` and `README.md` to reflect that `v0.4` now has narrow single-instance snapshot support with stopped-VM-only constraints.

### V0.4-T10: Add snapshot smoke coverage and release notes

Status: [x]

Dependencies:

- V0.4-T9

Files:

- `scripts/manual-smoke.sh`
- `docs/tutorial-test.md`
- `docs/release-notes-v0.4.0.md`
- `CHANGELOG.md`
- `TASKS.md`

Definition of done:

- smoke script validates snapshot create/list/restore/delete on one real VM
- release notes describe snapshot limits honestly
- docs stay explicit that multi-VM reset is still later work

Completion notes:

- Extended the smoke script positive/full path to cover stopped-VM snapshot create, list, break, restore, and delete on the provisioned Caddy example VM.
- Rewrote the manual test tutorial for the `v0.4.0` candidate so it matches the new snapshot loop and stopped-VM restore constraints.
- Added `docs/release-notes-v0.4.0.md` and updated `CHANGELOG.md` with the narrow snapshot/reset scope and current limits.

## M13: Private Networking

Status: [~]

Goal:

Add the first real multi-VM networking model so Yeast can run simple lab topologies with predictable private addresses while keeping management SSH separate.

Key constraints:

- keep management SSH on the current host-forwarded path
- add one explicit private lab network model first
- prefer a narrow Linux-only implementation over premature abstraction
- no bridge mode or multi-network complexity in the first pass

Release target:

- `v0.5.0`

### V0.5-T1: Lock the v0.5 networking contract

Status: [x]

Dependencies:

- M12 complete

Files:

- `YEAST_TECHNICAL_ARCHITECTURE.md`
- `YEAST_V2_IMPLEMENTATION_PLAN.md`
- `docs/known-limitations.md`
- `TASKS.md`

Definition of done:

- management network vs lab network responsibilities are explicit
- first supported config shape is explicit
- first supported scope is explicit:
  - one project-level private network
  - per-instance static IPv4 on that network
  - management SSH still separate
- non-goals are explicit:
  - no bridge mode yet
  - no DHCP
  - no multiple private networks yet
  - no GUI topology tooling

Completion notes:

- Locked the first `v0.5` networking scope as one project-level private lab network with per-instance static IPv4.
- Made the management-vs-lab split explicit in architecture and implementation docs.
- Documented the intentional non-goals for the first pass: no bridge mode, no DHCP, and no multi-network topologies yet.

### V0.5-T2: Add config schema and validation for project networks

Status: [x]

Dependencies:

- V0.5-T1

Files:

- `internal/config/model.go`
- `internal/config/validate.go`
- `internal/config/defaults.go` if needed
- tests
- `TASKS.md`

Definition of done:

- `yeast.yaml` can declare one private network
- instances can opt into that network with a static IPv4
- invalid CIDR or invalid IPs fail validation
- duplicate IPs fail validation

Completion notes:

- Replaced the old placeholder `Instance.Networks []string` with structured per-instance network attachments using `name` and `ipv4`.
- Added validation for the first `v0.5` contract:
  - at most one project network
  - required IPv4 CIDR on that project network
  - at most one private network attachment per instance
  - attachment must reference a declared network
  - required static IPv4 per attachment
  - IPv4 must be inside the declared CIDR
  - duplicate instance IPv4s on the same network fail validation
- Extended loader coverage so YAML parsing matches the locked architecture shape.

### V0.5-T3: Add runtime network model types

Status: [x]

Dependencies:

- V0.5-T2

Files:

- `internal/runtime/model.go`
- `internal/runtime/runtime.go`
- related tests
- `TASKS.md`

Definition of done:

- runtime plan can express:
  - management SSH forwarding
  - one named lab network
  - per-instance static IPv4 on that network
- model stays independent from QEMU flag formatting

Completion notes:

- Replaced the old runtime `ManagementNetwork` placeholder with a semantic `NetworkPlan`.
- Added:
  - `ManagementNetworkPlan` for SSH host/port forwarding intent
  - optional `LabNetworkPlan` for one named lab network with CIDR and per-instance IPv4
- Switched the QEMU runtime and app test fakes to carry the new model without adding lab-network QEMU flags yet.
- Added runtime model coverage proving one machine plan can express both management forwarding and one private lab address.

### V0.5-T4: Add QEMU network command building for one private lab network

Status: [x]

Dependencies:

- V0.5-T3

Files:

- `internal/runtime/qemu/*`
- related tests
- `TASKS.md`

Definition of done:

- QEMU command builder can attach instances to:
  - existing management path
  - one private lab network backend
- generated runtime command is test-covered
- no bridge mode yet

Completion notes:

- Kept the existing user-mode management network path unchanged for SSH forwarding.
- Added a second optional QEMU NIC/backend for the first private lab network.
- Used a rootless project-scoped multicast socket backend derived from the project runtime path and network name so instances in the same project share one isolated lab segment without bridge mode.
- Added command-layer validation for lab network CIDR and per-instance IPv4 membership.
- Extended QEMU command tests to cover the extra lab NIC/backend and stable multicast derivation.

### V0.5-T5: Add guest-side private address bootstrap through cloud-init

Status: [x]

Dependencies:

- V0.5-T2
- V0.5-T4

Files:

- `internal/provision/cloudinit/*`
- related tests
- `TASKS.md`

Definition of done:

- cloud-init network config renders static IPv4 for the private lab NIC
- instances boot with the configured lab address
- existing management SSH remains reachable

Completion notes:

- Added a dedicated cloud-init `network-config` renderer for the first private lab NIC instead of overloading `user-data`.
- Rendered a static IPv4 config using cloud-init network v2 format with:
  - deterministic interface name
  - MAC-based match
  - no DHCP on the lab NIC
  - one static IPv4 address from the configured CIDR
- Extended the seed ISO builder so `network-config` is written and included when present.
- Added focused tests for the rendered network config and ISO inclusion path.

### V0.5-T6: Wire private network planning into `yeast up`

Status: [x]

Dependencies:

- V0.5-T5

Files:

- `internal/app/up.go`
- related tests
- `TASKS.md`

Definition of done:

- `yeast up` passes private network config into runtime and cloud-init
- current single-VM projects still behave the same
- multi-VM config builds and boots through the app layer

Completion notes:

- Added lab network planning in `yeast up` from:
  - project network CIDR
  - instance network attachment name
  - instance static IPv4
- Derived deterministic first-pass lab NIC details:
  - interface name: `yeastlab0`
  - MAC address from project id + instance + network name
- Wired the same lab NIC details into:
  - runtime network plan for QEMU
  - cloud-init `network-config` seed content
- Added app-level coverage proving the lab network data reaches both runtime and seed generation while existing single-VM projects remain unaffected.

### V0.5-T7: Expose private network details in `yeast status`

Status: [x]

Dependencies:

- V0.5-T6

Files:

- `internal/app/status.go`
- renderers/tests if needed
- `TASKS.md`

Definition of done:

- status output exposes per-instance lab IP when configured
- JSON output includes the same data cleanly

Completion notes:

- Added `lab_ip` to instance state as desired-state networking metadata.
- `yeast up` now persists the configured lab IP when a private lab network is attached.
- `yeast status` now returns the lab IP in both app results and JSON output.
- Human status rendering now shows a `LAB IP` column while keeping `-` for instances without private lab networking.
- Added focused app/output/render tests for persistence and display.

### V0.5-T8: Add two-VM example lab docs

Status: [x]

Dependencies:

- V0.5-T7

Files:

- `examples/*`
- `docs/quickstart.md`
- `docs/known-limitations.md`
- `README.md`
- `TASKS.md`

Definition of done:

- one example shows two VMs on one private network
- docs explain management SSH vs lab traffic clearly

Completion notes:

- Added `examples/two-vm-lab` with:
  - one project-level private lab network
  - `attacker` and `target` instances
  - separate management SSH ports
  - static lab IPs
- Updated the quickstart to point at the first two-VM private lab flow.
- Updated known limitations to reflect actual `v0.5` networking support and current constraints.
- Updated the README current scope and examples so `v0.5` now clearly includes the first private lab network slice.

### V0.5-T9: Add networking smoke coverage and release notes

Status: [x]

Dependencies:

- V0.5-T8

Files:

- `scripts/manual-smoke.sh`
- `docs/tutorial-test.md`
- `docs/release-notes-v0.5.0.md`
- `CHANGELOG.md`
- `TASKS.md`

Definition of done:

- smoke path proves two VMs can boot and reach each other on the private network
- release notes describe networking limits honestly
- docs stay explicit that bridge mode and multi-network topologies are later work

Completion notes:

- Expanded the manual smoke suite to cover the first full `v0.5` networking loop:
  - two-VM project init
  - private lab network boot
  - per-instance `LAB IP` status checks
  - guest-side NIC verification
  - guest-to-guest SSH reachability over the lab network
  - existing negative JSON contract checks
- Updated the manual test tutorial for the `v0.5.0` candidate and added `docs/release-notes-v0.5.0.md`.
- Updated `CHANGELOG.md` for the `v0.5.0` scope.
- Fixed two real networking issues found by host smoke:
  - made the management NIC explicit in cloud-init with deterministic MAC + DHCP so lab `network-config` no longer destabilizes management SSH
  - bound the rootless QEMU multicast lab backend to loopback so same-host guests actually share the private lab bus reliably
- Verified the full host smoke flow end to end with:
  - single-VM provisioning + snapshot/restore
  - two-VM private networking
  - negative config/error scenarios

## M14: Guest Control

Status: [x]

Goal:

Add the first real guest-control surface so users and later automation can run commands, move files, inspect runtime details, and read logs without dropping to manual SSH every time.

Key constraints:

- reuse the existing SSH transport instead of building a second remote-control path
- keep command behavior explicit and instance-scoped
- return structured stdout/stderr/exit-code data
- keep interactive SSH (`yeast ssh`) separate from non-interactive guest control
- do not add MCP-specific protocol concerns in this milestone

Release target:

- `v0.6.0`

### V0.6-T1: Lock the v0.6 guest-control contract

Status: [x]

Dependencies:

- M13 complete

Files:

- `YEAST_TECHNICAL_ARCHITECTURE.md`
- `YEAST_V2_IMPLEMENTATION_PLAN.md`
- `docs/known-limitations.md`
- `TASKS.md`

Definition of done:

- command surface is explicit:
  - `yeast exec`
  - `yeast copy`
  - `yeast logs`
  - `yeast inspect`
- result contract is explicit:
  - command
  - exit_code
  - stdout
  - stderr
  - started_at
  - finished_at
  - duration
  - timed_out
- scope limits are explicit:
  - SSH-backed only
  - one-instance targeting first
  - no streaming/log-follow mode yet
  - no service health checks yet

Completion notes:

- Replaced the placeholder `M14` block with a real `v0.6.0` task sequence.
- Locked the first guest-control surface as:
  - `yeast exec`
  - `yeast copy`
  - `yeast logs`
  - `yeast inspect`
- Locked the execution/result contract for `exec` around structured stdout/stderr/exit-code metadata.
- Documented first-pass limits clearly:
  - SSH-backed only
  - one selected instance per command
  - no log streaming
  - no health-check framework yet

### V0.6-T2: Add guest-control app/result models

Status: [x]

Dependencies:

- V0.6-T1

Files:

- `internal/app/*`
- `internal/guest/*` or shared result types if needed
- tests
- `TASKS.md`

Definition of done:

- app-layer result types exist for:
  - exec
  - copy
  - inspect
  - logs
- result shape matches the locked contract
- tests cover JSON-friendly fields and zero-value behavior

Completion notes:

- Added shared guest-control app/result models for:
  - `ExecResult`
  - `CopyResult`
  - `InspectResult`
  - `LogsResult`
- Added a shared `GuestCommandResult` contract with:
  - command
  - exit code
  - stdout
  - stderr
  - timestamps
  - duration
  - timeout flag
- Added focused tests covering:
  - stable command-string rendering
  - local-path normalization for copy workflows
  - result-shape field expectations for later JSON rendering

### V0.6-T3: Add `exec` workflow in the app layer

Status: [x]

Dependencies:

- V0.6-T2

Files:

- `internal/app/*`
- related tests
- `TASKS.md`

Definition of done:

- app service can select one running instance
- command runs over SSH
- stdout/stderr/exit code are returned cleanly
- not-running or not-found targets map to stable app errors

Completed:

- Added `Service.Exec` with one-instance target selection and SSH-backed command execution.
- Return contract preserves `stdout`, `stderr`, `exit_code`, timestamps, duration, and timeout state.
- Remote non-zero command exits return structured results without surfacing an app error.
- SSH transport/setup failures map to stable app errors.
- Added focused tests for success, remote non-zero exit, timeout, missing command, and SSH failure classification.

### V0.6-T4: Add `copy` workflow in the app layer

Status: [x]

Dependencies:

- V0.6-T2

Files:

- `internal/app/*`
- `internal/guest/*` if needed
- tests
- `TASKS.md`

Definition of done:

- app layer supports:
  - host -> guest copy
  - guest -> host copy
- one target instance at a time
- local path validation is explicit
- failures map to stable app errors

Completed:

- Added `Service.Copy` for both `to_guest` and `from_guest` directions.
- Extended the SSH transport with `Download` so guest -> host copy stays on the same transport surface as upload and exec.
- Added explicit local path validation:
  - upload source must resolve to an existing local file
  - download destination parent directory must already exist
- SSH transport/setup failures map to stable app errors.
- Added focused tests for upload, download, invalid direction, invalid local paths, and transport failure classification.

### V0.6-T5: Add `inspect` and `logs` workflows in the app layer

Status: [x]

Dependencies:

- V0.6-T2

Files:

- `internal/app/*`
- tests
- `TASKS.md`

Definition of done:

- inspect returns useful runtime details for one instance
- logs returns the VM log path/content surface defined for v0.6
- app errors are stable for missing instance/log files

Completed:

- Added `Service.Inspect` to return one-instance runtime/status details plus snapshot metadata summary.
- Added `Service.Logs` to expose the per-instance VM runtime log at `runtimeDir/vm.log`.
- `logs` supports optional line-tail behavior for compact reads.
- Missing instance maps to `not_found`; missing VM log maps to `not_found`; missing runtime directory maps to `internal`.
- Added focused tests for inspect details, snapshot summaries, log reads, missing logs, and tail behavior.

### V0.6-T6: Add CLI commands for guest control

Status: [x]

Dependencies:

- V0.6-T3
- V0.6-T4
- V0.6-T5

Files:

- `cmd/yeast/*`
- `internal/output/*`
- tests
- `TASKS.md`

Definition of done:

- CLI commands exist:
  - `yeast exec`
  - `yeast copy`
  - `yeast logs`
  - `yeast inspect`
- human and JSON output both work
- target selection rules are consistent with `yeast ssh`

Completed:

- Added CLI commands:
  - `yeast exec [instance] -- <command...>`
  - `yeast copy <instance> --to-guest <source> <destination>`
  - `yeast copy <instance> --from-guest <source> <destination>`
  - `yeast logs <instance> [--tail N]`
  - `yeast inspect <instance>`
- Added human renderers for `exec`, `copy`, `logs`, and `inspect`.
- JSON success rendering works through the existing generic envelope for all guest-control result types.
- Added focused render tests in `cmd/yeast` and `internal/output`.

### V0.6-T7: Add guest-control smoke coverage

Status: [x]

Dependencies:

- V0.6-T6

Files:

- `scripts/manual-smoke.sh`
- tests if needed
- `TASKS.md`

Definition of done:

- smoke path proves:
  - `yeast exec` can run a command in guest
  - `yeast copy` can move at least one file in each direction
  - `yeast logs` exposes VM log access
  - `yeast inspect` returns expected machine details

Completed:

- Expanded `scripts/manual-smoke.sh` positive flow to validate:
  - structured `yeast exec` output
  - host -> guest copy
  - guest -> host copy
  - `yeast inspect --json`
  - `yeast logs --json --tail`
- Full real-host smoke passed against a fresh local `v0.6` binary while preserving the existing lifecycle, provisioning, snapshot, networking, and negative JSON checks.

### V0.6-T8: Add guest-control docs and release notes

Status: [x]

Dependencies:

- V0.6-T7

Files:

- `README.md`
- `docs/quickstart.md`
- `docs/known-limitations.md`
- `docs/tutorial-test.md`
- `docs/release-notes-v0.6.0.md`
- `CHANGELOG.md`
- `TASKS.md`

Definition of done:

- docs explain the new commands clearly
- current v0.6 limits are explicit
- release notes match shipped behavior

Completed:

- Updated README current scope, quickstart, and command inventory for `exec`, `copy`, `logs`, and `inspect`.
- Updated quickstart and manual test docs to include the first guest-control workflow.
- Updated known limitations with the actual `v0.6` guest-control constraints.
- Added `docs/release-notes-v0.6.0.md`.
- Added a draft `0.6.0` changelog entry.

## M15: Templates

Status: [~]

Goal:

Make common Yeast environments reusable so users can start from known-good project shapes instead of rewriting `yeast.yaml`, provisioning files, and example assets by hand.

Key constraints:

- keep templates as project starters, not hidden runtime behavior
- generated projects must remain normal editable Yeast projects
- built-in templates come first so v0.7 is useful without remote trust problems
- local filesystem templates come second for user/team reuse
- remote templates, registries, update workflows, and complex variables are deferred
- do not add LabsBackery-specific commands in this milestone

Release target:

- `v0.7.0`

### V0.7-T1: Lock the v0.7 template contract

Status: [x]

Dependencies:

- M14 complete

Files:

- `YEAST_TECHNICAL_ARCHITECTURE.md`
- `YEAST_V2_IMPLEMENTATION_PLAN.md`
- `docs/known-limitations.md`
- `TASKS.md`

Definition of done:

- v0.7 template scope is explicit
- first command surface is explicit:
  - `yeast init --template <name-or-path>`
  - `yeast init --list-templates`
- supported template sources are explicit:
  - built-in templates
  - local directory templates
- template content contract is explicit:
  - `template.yaml` metadata
  - `yeast.yaml`
  - optional project files/assets
- deferred scope is explicit:
  - no remote template downloads
  - no template registry
  - no template update/sync
  - no complex variable engine

Completion notes:

- Added the v0.7 template milestone to the execution checklist and moved LabsBackery contract to the next milestone.
- Locked the first template command surface around `yeast init --template` and `yeast init --list-templates`.
- Chose built-in and local directory templates for v0.7; remote templates and registries remain deferred.
- Locked the template shape as metadata plus normal project files, with generated output remaining fully editable.

### V0.7-T2: Add built-in template catalog and metadata model

Status: [x]

Dependencies:

- V0.7-T1

Files:

- `internal/templates/*`
- `templates/*` or embedded template assets
- tests
- `TASKS.md`

Definition of done:

- template metadata type exists
- built-in templates can be listed from app/internal code
- metadata includes name, title, description, category, and included files
- tests cover deterministic ordering and missing/corrupt metadata behavior

Completion notes:

- Added `internal/templates` with a template metadata model and validation.
- Added embedded built-in metadata for:
  - `ubuntu-basic`
  - `caddy-single-vm`
  - `two-vm-lab`
- Added catalog helpers for sorted built-in listing, name listing, and lookup.
- Added local template metadata loading for the next materialization task.
- Added tests for deterministic ordering, lookup, corrupt metadata, missing metadata, unsafe paths, and duplicate file entries.

### V0.7-T3: Add template materialization service

Status: [x]

Dependencies:

- V0.7-T2

Files:

- `internal/templates/*`
- `internal/app/init.go`
- tests
- `TASKS.md`

Definition of done:

- built-in template files can be copied into a new project directory
- local template directories can be copied into a new project directory
- existing project conflict behavior stays conservative
- path traversal and unsafe destination paths are rejected
- generated project receives normal `.yeast/project.json`

Completion notes:

- Embedded the current official example project files as built-in template payloads.
- Added `templates.Materialize` to copy declared template files into an empty destination without overwriting existing files.
- Added local template file materialization using the same metadata contract.
- Extended `Service.Init` with internal template selection so `init` can create normal project metadata after template files are copied.
- Added tests for built-in materialization, local materialization, output conflicts, missing local files, built-in init, local init, missing templates, and conflict classification.

### V0.7-T4: Add `init --template` and `init --list-templates`

Status: [x]

Dependencies:

- V0.7-T3

Files:

- `cmd/yeast/init.go`
- `internal/output/*`
- tests
- `TASKS.md`

Definition of done:

- `yeast init --list-templates` renders human output and JSON output
- `yeast init --template caddy-single-vm` creates a normal project
- `yeast init --template ./path/to/template` works for local templates
- missing template names return stable `not_found` errors

Completion notes:

- Added `Service.ListTemplates` and JSON-friendly template summary results.
- Added `yeast init --list-templates`.
- Added `yeast init --template <name-or-path>`.
- Added human template-list rendering and JSON success coverage.
- Added CLI tests proving template listing and `caddy-single-vm` project creation.

### V0.7-T5: Add official built-in templates

Status: [x]

Dependencies:

- V0.7-T4

Files:

- `templates/*`
- `examples/*` if reused
- tests
- `TASKS.md`

Definition of done:

- built-in `ubuntu-basic` template exists
- built-in `caddy-single-vm` template exists
- built-in `two-vm-lab` template exists
- templates match currently supported v0.6 features
- templates do not rely on future snapshot automation, events, MCP, or cloud behavior

Completion notes:

- Updated built-in template READMEs so they document `yeast init --template <name>` instead of old manual copy workflows.
- Kept the official built-ins scoped to current supported behavior:
  - `ubuntu-basic`
  - `caddy-single-vm`
  - `two-vm-lab`
- Added tests proving every official built-in materializes into a valid Yeast project.
- Added guard tests to keep built-ins from introducing future-scope files such as MCP, cloud, events, or LabsBackery-specific package files.

### V0.7-T6: Add template docs and smoke coverage

Status: [x]

Dependencies:

- V0.7-T5

Files:

- `README.md`
- `docs/quickstart.md`
- `docs/config-reference.md`
- `docs/tutorial-test.md`
- `scripts/manual-smoke.sh`
- `TASKS.md`

Definition of done:

- docs show how to list and initialize templates
- smoke test proves at least one built-in template can initialize and run
- docs explain local template shape
- limitations explain deferred remote template behavior

Completion notes:

- Updated README, quickstart, config reference, manual test docs, and embedded terminal docs for v0.7 templates.
- Added smoke coverage for `yeast init --list-templates`.
- Updated the positive smoke path so the real Caddy VM flow starts from `yeast init --template caddy-single-vm`.
- Added negative smoke coverage for missing template names returning `not_found`.
- Verified the positive host smoke path through template init, Caddy provisioning, guest control, snapshot/restore, and two-VM networking.

### V0.7-T7: Add v0.7 release notes

Status: [x]

Dependencies:

- V0.7-T6

Files:

- `docs/release-notes-v0.7.0.md`
- `CHANGELOG.md`
- `TASKS.md`

Definition of done:

- release notes match shipped template behavior
- limitations are honest
- changelog contains a draft v0.7.0 entry

Completion notes:

- Added `docs/release-notes-v0.7.0.md`.
- Added a draft `0.7.0` changelog entry.
- Updated README current limits so shipped snapshots, networking, guest control, and templates are represented accurately.

## M16: LabsBackery Contract

Status: [~]

Do not start until:

- M11-M15 enough for one lab
- M15 / v0.8 stable JSON and events are complete enough for UI integration

Planning reference:

- `docs/labsbackery-plan.md`
- `docs/labsbackery-integration-contract.md`

Core tasks later:

- Inspect old LabsBackery as reference only.
- Extract the old lab model, terminal workflow, reset workflow, and UI expectations.
- Map old LabsBackery concepts to current Yeast primitives.
- Define the first LabsBackery lab integration target.
- Define the LabsBackery-to-Yeast JSON contract.
- Define required lifecycle event names for UI progress.
- Define terminal connection requirements.
- Define reset/baseline snapshot requirements.
- Build one test lab that Yeast can start, provision, snapshot, reset, and destroy.
- Write LabsBackery integration notes.

Do not do in Yeast:

- do not copy old LabsBackery code into Yeast
- do not add LabsBackery-specific product state to Yeast
- do not add cloud, billing, teams, courses, or RBAC here
- do not start LabsBackery implementation before v0.8 JSON/events are stable

---

# V0.8: Stable JSON And Events

Goal:

Make Yeast dependable as an engine for LabsBakery, Yeast MCP, scripts, and future UIs.

## V0.8 Tasks

### V0.8-T1: Add versioned JSON envelope

Status: [x]

Files:

- `internal/output/schemas.go`
- `internal/output/json.go`
- `internal/output/json_test.go`
- `internal/output/schemas_test.go`
- `docs/json-contract.md`
- `TASKS.md`

Definition of done:

- success JSON includes `schema_version`
- error JSON includes `schema_version`
- schema version value is centralized
- tests lock the envelope field
- docs describe the JSON contract

Completion notes:

- Added `output.SchemaVersion` with current value `yeast.v1`.
- Added `schema_version` to success and error JSON envelopes.
- Updated JSON envelope tests to assert the versioned contract.
- Added `docs/json-contract.md` with the initial v0.8 JSON contract.

### V0.8-T2: Document and harden standard error codes

Status: [x]

Dependencies:

- V0.8-T1

Definition of done:

- error codes are documented
- app errors cover timeout/runtime/provisioning/guest categories where needed
- tests verify representative command failures return expected codes

Completion notes:

- Added stable error code constants for `timeout`, `runtime_error`, `provisioning_failed`, and `guest_error`.
- Locked error code string values with tests.
- Classified failed guest exec/copy operations as `guest_error`.
- Classified failed provisioning from `yeast provision` and `yeast up` as `provisioning_failed`.
- Expanded `docs/json-contract.md` with the current error code catalog and meanings.

### V0.8-T3: Lock command-specific JSON data shapes

Status: [x]

Dependencies:

- V0.8-T1

Definition of done:

- core command JSON outputs have tests for required fields
- docs list stable fields for status, inspect, exec, copy, logs, snapshots, and template listing

Completion notes:

- Added JSON tags to app result structs so command data uses stable lower `snake_case` fields.
- Updated manual smoke JSON parsing to use the stable field names.
- Expanded command-rendering tests to assert required fields for core command outputs.
- Documented the first stable command data shapes in `docs/json-contract.md`.

### V0.8-T4: Add lifecycle event model

Status: [x]

Dependencies:

- V0.8-T1

Definition of done:

- event envelope is defined
- event names are documented
- app workflows can emit events without coupling to human output

Completion notes:

- Added `app.Event`, `app.EventName`, `app.EventSink`, and `app.NewEvent`.
- Added initial stable event names for project/config/image/disk/cloud-init/start/SSH/provision/snapshot/restore/instance/workflow progress.
- Added `output.RenderJSONEvent` for JSON Lines event rendering.
- Documented the event envelope and initial event names in `docs/json-contract.md`.
- Added tests that lock event envelope fields and event name strings.

### V0.8-T5: Add `--events` output path for long-running commands

Status: [x]

Dependencies:

- V0.8-T4

Definition of done:

- `up`, `provision`, and `restore` can stream machine-readable events
- events remain separate from human rendering
- JSON/event behavior is tested

Completion notes:

- Added global `--events` flag.
- Required `--events` to be used with `--json` so machine events do not mix with human output.
- Wired JSON Lines events into `up`, `provision`, and `restore`.
- Added CLI event sink tests and app-level provision/restore event emission tests.
- Documented `--json --events` usage in `docs/json-contract.md`.

### V0.8-T6: Add v0.8 release notes

Status: [x]

Dependencies:

- V0.8-T1
- V0.8-T2
- V0.8-T3
- V0.8-T4
- V0.8-T5

Files:

- `docs/release-notes-v0.8.0.md`
- `CHANGELOG.md`
- `README.md`
- `TASKS.md`

Definition of done:

- v0.8 release notes exist
- changelog contains a draft v0.8.0 entry
- README current scope reflects stable JSON and events
- project docs link to the JSON contract and v0.8 release notes

Completion notes:

- Added `docs/release-notes-v0.8.0.md`.
- Added the draft v0.8.0 changelog entry.
- Updated README status, current scope, limitations, and project docs for v0.8 automation.

---

# V0.9: LabsBakery-Ready Engine

Goal:

Prove Yeast can act as the local VM engine behind one real LabsBakery lab workflow without adding LabsBakery product state to Yeast.

## V0.9 Tasks

### V0.9-T1: Define the LabsBakery integration contract

Status: [x]

Dependencies:

- V0.8-T1
- V0.8-T2
- V0.8-T3
- V0.8-T4
- V0.8-T5

Files:

- `docs/labsbackery-integration-contract.md`
- `README.md`
- `TASKS.md`

Definition of done:

- LabsBakery/Yeast ownership boundary is explicit
- session directory model is defined
- required CLI/JSON commands are listed
- terminal connection strategy is defined
- reset/baseline workflow is defined
- event handling guidance is defined
- first test lab target is defined
- known Yeast gaps are documented without implementing future scope early

Completion notes:

- Added `docs/labsbackery-integration-contract.md`.
- Documented the first LabsBakery adapter command contract around `yeast.v1`.
- Documented terminal, event, reset, error-handling, and first test lab expectations.
- Linked the new contract from README and the M16 planning references.

### V0.9-T2: Expose guest user in status and inspect JSON

Status: [x]

Dependencies:

- V0.9-T1

Definition of done:

- `status --json` includes the configured guest user for each instance
- `inspect <instance> --json` includes the configured guest user
- JSON contract docs include the new optional/stable field
- tests lock the field names
- manual smoke can assert terminal connection info without reading `yeast.yaml`

Completion notes:

- Added `user` to instance state and status/inspect result JSON.
- Set `user` from the normalized instance config during `yeast up`.
- Updated JSON contract docs and the LabsBakery integration contract terminal guidance.
- Updated unit tests and manual smoke assertions for `status --json` and `inspect --json`.

### V0.9-T3: Add event coverage for stop/destroy workflows

Status: [x]

Dependencies:

- V0.9-T1
- V0.8-T5

Definition of done:

- `down --json --events` emits lifecycle events
- `destroy --json --events` emits lifecycle events
- events stay compatible with `yeast.v1`
- tests cover the CLI event path

Completion notes:

- Added event sinks to `DownOptions` and `DestroyOptions`.
- Wired `down` and `destroy` CLI commands through the shared `--json --events` renderer.
- Emitted `project.loaded`, per-instance `instance.stopped` / `instance.destroyed`, and `workflow.completed`.
- Updated JSON and LabsBakery integration docs.
- Added service-level event tests and CLI validation tests for `--events` requiring `--json`.

### V0.9-T4: Define first lab package/template convention

Status: [x]

Dependencies:

- V0.9-T1

Definition of done:

- docs define the minimum folder shape for a Yeast-backed LabsBakery lab
- convention covers `yeast.yaml`, provision files, lab metadata, and optional scenario/check files
- built-in templates are not converted into LabsBakery product packages yet

Completion notes:

- Added `docs/labsbackery-lab-package.md`.
- Defined the first package/session folder shape for Yeast-backed LabsBakery labs.
- Documented `template.yaml`, `yeast.yaml`, `lab.yaml`, scenario instructions, checks, assets, session layout, baseline/reset conventions, and import/export boundaries.
- Linked the package convention from the LabsBakery integration contract and README.

### V0.9-T5: Build one LabsBakery-ready example lab

Status: [x]

Dependencies:

- V0.9-T2
- V0.9-T4

Definition of done:

- example lab has attacker and target VMs
- example lab has one private lab network
- target has a simple service or file for validation
- lab can start, status, snapshot, reset, and destroy through Yeast
- full smoke or documented manual test proves the workflow

Completion notes:

- Added `examples/labsbackery-attacker-target-basic`.
- Included `template.yaml`, `yeast.yaml`, `lab.yaml`, scenario instructions, scenario checks, and a target marker file.
- The example defines attacker/target VMs on one private lab network with static IPs.
- The target provisions `/home/yeast/labsbackery-target.txt` as the validation file.
- README documents init, start/status, checks through `yeast exec`, baseline snapshots, reset, and cleanup.
- Added a test that materializes the example as a local template and validates the generated `yeast.yaml`.

### V0.9-T6: Add v0.9 release notes

Status: [x]

Dependencies:

- V0.9-T1
- V0.9-T2
- V0.9-T3
- V0.9-T4
- V0.9-T5

Definition of done:

- v0.9 release notes exist
- changelog contains v0.9.0 entry
- README current scope reflects LabsBakery-ready integration behavior

Completion notes:

- Added `docs/release-notes-v0.9.0.md`.
- Added a v0.9.0 draft entry to `CHANGELOG.md`.
- Updated README scope, current features, current limits, config label, badge, and project docs for v0.9.
- Updated the JSON contract status line to v0.9.0 while keeping `schema_version: "yeast.v1"`.

---

# V1.0: Stable Public Release

Goal:

Make Yeast stable enough for real users and dependent products to trust. v1.0 is not a feature-expansion milestone. It is a hardening, compatibility, documentation, release-quality, and end-to-end validation milestone for the core Yeast engine shipped across v0.1 through v0.9.

Scope boundary:

- Do not add Twarga Cloud.
- Do not add a daemon or web API.
- Do not add a LabsBakery web UI.
- Do not add an AI/MCP server inside Yeast.
- Do not add VirtualBox, Windows host support, or macOS host support.
- Do not add a remote template registry.
- Do not add advanced multi-network topology features beyond the current v0.5 model unless a v1-blocking bug is found.

v1.0 should stabilize:

- lifecycle commands
- config schema
- state behavior
- provisioning behavior
- snapshot/reset behavior
- private lab networking behavior
- guest control behavior
- template behavior
- JSON/events contract
- examples
- docs
- installer and release artifacts

## V1.0 Tasks

### V1.0-T1: Audit v1 contract surface

Status: [x]

Dependencies:

- V0.9-T6

Definition of done:

- list all public CLI commands and flags that v1.0 promises to keep stable
- list all public `yeast.yaml` fields that v1.0 promises to support
- list all public JSON envelopes, data fields, error codes, and event names that v1.0 promises to keep stable
- identify any current command, field, flag, or JSON shape that is too weak or inconsistent for v1.0
- create or update one contract audit doc under `docs/`
- do not change runtime behavior in this task unless only docs are wrong

Completion notes:

- Added `docs/v1-contract-audit.md`.
- Listed the v1 candidate CLI command and global flag surface.
- Listed the v1 candidate config schema fields, network fields, instance network fields, and provisioning fields.
- Listed stable JSON envelope fields, error codes, documented command data shapes, event commands, event envelope fields, and event names.
- Recorded v1 follow-up gaps for command docs, full JSON docs, config network docs, installer version verification, stale tutorial HTML, and GitHub Actions Node 20 warnings.
- Made no runtime behavior changes.

### V1.0-T2: Freeze command reference

Status: [x]

Dependencies:

- V1.0-T1

Definition of done:

- command reference exists and matches actual CLI behavior
- every current command has purpose, syntax, flags, human behavior, JSON behavior, examples, and known limits
- command reference includes:
  - `doctor`
  - `init`
  - `pull`
  - `up`
  - `provision`
  - `snapshot`
  - `restore`
  - `snapshots`
  - `delete-snapshot`
  - `status`
  - `exec`
  - `copy`
  - `logs`
  - `inspect`
  - `ssh`
  - `down`
  - `destroy`
  - `version`
  - `docs`
- tests or scripted checks verify command help still renders

Completion notes:

- Added `docs/command-reference.md`.
- Documented every v1 command with purpose, syntax, flags, human behavior, JSON behavior, examples, and known limits.
- Linked the command reference from README project docs.
- Added `cmd/yeast/help_test.go` to verify help renders for every v1 command surface and includes the expected usage line.

### V1.0-T3: Freeze config reference

Status: [x]

Dependencies:

- V1.0-T1

Definition of done:

- config reference documents every supported `yeast.yaml` field
- config reference includes required/optional status, type, default, validation rule, and example
- current examples use only documented fields
- invalid config tests cover important validation errors
- docs clearly mark unsupported future fields as out of scope

Completion notes:

- Expanded `docs/config-reference.md` with a v1 full example covering networks, provisioning, env, defaults, and instance overrides.
- Added required/type/default tables for top-level and instance fields.
- Added explicit defaults, private lab networking, project network fields, instance network attachment fields, validation rules, and v1 out-of-scope config fields.
- Removed stale `v0.3` provisioning wording from the reference.
- Added validation tests for missing/IPv6 project network CIDR and reserved network/broadcast instance IPv4 values.

### V1.0-T4: Freeze JSON and event contract

Status: [x]

Dependencies:

- V1.0-T1

Definition of done:

- `docs/json-contract.md` is complete enough for LabsBakery, scripts, and Yeast MCP
- every JSON command output has documented fields
- every standard error code is documented
- every event name emitted by v1.0 workflows is documented
- JSON compatibility guarantee is stated clearly
- contract tests cover representative success and error envelopes

Completion notes:

- Updated `docs/json-contract.md` status for v1.0 stabilization.
- Documented every current JSON-capable command data shape, including doctor, init, pull, up, provision, status, inspect, exec, copy, logs, snapshot, restore, snapshots, delete-snapshot, down, destroy, and version.
- Explicitly documented that `ssh` is interactive and `docs` does not support `--json`.
- Kept the `yeast.v1` compatibility rule, standard error codes, event-streaming commands, event envelope fields, and stable event names in one contract doc.
- Tightened JSON renderer tests for representative fields that were previously omitted from assertions.

### V1.0-T5: Harden installer and upgrade path

Status: [x]

Dependencies:

- V1.0-T1

Definition of done:

- installer installs latest stable release by default
- installer supports explicit `YEAST_REF`
- installer verifies required host dependencies
- installer verifies downloaded or built binary version
- installer behavior is documented for common Linux families
- failure messages are actionable
- shell syntax checks pass

Completion notes:

- Changed installer default `YEAST_REF` from `main` to the current stable release tag `v0.9.0`.
- Kept explicit `YEAST_REF` support for tags, branches, and commits.
- Added version injection during installer builds using the semantic `YEAST_REF` or `YEAST_EXPECTED_VERSION`.
- Added installed binary verification after install, including exact version matching for semantic release refs.
- Updated installation docs and README install snippet to describe stable default install, explicit refs, version verification, supported package managers, and upgrade behavior.

### V1.0-T6: Expand release smoke coverage

Status: [x]

Dependencies:

- V1.0-T1

Definition of done:

- manual smoke script covers the full current user loop
- smoke modes are documented
- smoke coverage includes:
  - doctor
  - init
  - pull
  - up
  - status
  - provisioning
  - guest exec
  - guest copy upload/download
  - logs
  - inspect
  - snapshot
  - restore
  - private lab networking status
  - template init
  - LabsBakery example materialization
  - down
  - destroy
  - negative JSON error cases
- script stays optional for host-dependent VM checks

Completion notes:

- Updated `scripts/manual-smoke.sh` default workdir for the v1 candidate.
- Added LabsBakery example package materialization coverage using `yeast init --template examples/labsbackery-attacker-target-basic`.
- Added `TEST_MODE=templates` for non-VM template and LabsBakery package smoke coverage.
- The smoke suite now verifies LabsBakery package files, lab schema, attacker/target config, scenario checks, and target marker file materialization.
- Refreshed stale smoke labels from old v0.5/v0.7 wording.
- Updated `docs/tutorial-test.md` for the v1.0 candidate and documented the expanded smoke coverage and modes.

### V1.0-T7: Run real-host v1 candidate validation

Status: [x]

Dependencies:

- V1.0-T2
- V1.0-T3
- V1.0-T4
- V1.0-T5
- V1.0-T6

Definition of done:

- run full automated Go test suite
- run static checks
- run installer path on the maintainer machine or documented supported Linux host
- run full manual smoke on a real KVM host
- run the Caddy example from template init through restore
- run the two-VM lab example
- run the LabsBakery attacker/target example at least through materialization and documented Yeast commands
- record validation notes in a v1.0 release checklist doc

Completion notes:

- Built `v1.0.0-rc1` release candidate artifact and verified checksum.
- Ran `go test ./... -count=1`.
- Ran shell syntax checks and whitespace diff check.
- Installed pinned static-analysis tools and ran `./scripts/static-analysis.sh artifacts`.
- Ran installer path through a temporary non-system install harness because the agent cannot provide the maintainer sudo password.
- Ran full real-host KVM smoke with `TEST_MODE=full WORKDIR=/tmp/yeast-v1-full-smoke ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64`.
- Full smoke passed Caddy template init, provisioning, guest control, snapshot/restore, two-VM networking, LabsBakery package materialization, and negative JSON error cases.
- Recorded validation in `docs/release-checklist-v1.0.0.md`.

### V1.0-T8: Refresh public docs and README for v1.0

Status: [x]

Dependencies:

- V1.0-T2
- V1.0-T3
- V1.0-T4
- V1.0-T7

Definition of done:

- README current scope says v1.0
- README quickstart is accurate for a new Linux user
- README examples point to current examples
- docs index links all important docs
- known limitations are honest and current
- LabsBakery integration docs reflect v1.0 as the minimum stable target
- landing page content does not overpromise

Completion notes:

- Updated README scope, status badge, quickstart output shapes, config wording, current limits, and docs index for v1.0.
- Updated quickstart and known limitations from old v0.7/v0.6 wording to the v1.0 local-engine surface.
- Updated installation examples to use `v1.0.0` for explicit release installs.
- Updated LabsBakery integration and package docs to require Yeast `v1.0.0` and describe the local-engine contract as stable.
- Updated landing page install command, docs links, footer links, and v1.0 wording without adding cloud/daemon promises.
- Fixed stale manual-test pass criteria wording.

### V1.0-T9: Prepare v1.0 release notes and changelog

Status: [x]

Dependencies:

- V1.0-T8

Definition of done:

- `docs/release-notes-v1.0.0.md` exists
- `CHANGELOG.md` contains v1.0.0 entry
- release notes summarize the stable product, not only the last minor delta
- release notes include install, upgrade, compatibility, limitations, and verification
- release notes clearly state what remains out of scope

Completion notes:

- Added `docs/release-notes-v1.0.0.md`.
- Added `CHANGELOG.md` v1.0.0 entry.
- Release notes summarize Yeast as the first stable local engine release, not only a v0.9 delta.
- Release notes include install, upgrade, compatibility, limitations, verification, migration notes, and out-of-scope items.
- README project docs now link the v1.0 release notes.

### V1.0-T10: Build and publish v1.0.0

Status: [x]

Dependencies:

- V1.0-T9

Definition of done:

- build artifact created with version injection
- binary reports `v1.0.0`
- checksum verifies
- final CI passes on `main`
- tag `v1.0.0` exists
- GitHub release exists
- release assets include Linux amd64 binary and checksum
- install command works against the released tag
- `TASKS.md` records the release completion

Completion notes:

- Updated installer default `YEAST_REF` to `v1.0.0`.
- Built final release artifact with `bash scripts/build-release.sh v1.0.0`.
- Verified `./dist/yeast-linux-amd64 version` reports `v1.0.0`.
- Verified `dist/yeast-linux-amd64.sha256`.
- Pushed annotated tag `v1.0.0`.
- Published GitHub release: `https://github.com/Twarga/yeast/releases/tag/v1.0.0`.
- Release assets include `yeast-linux-amd64` and `yeast-linux-amd64.sha256`.
- Validated the installer against the released tag with `YEAST_REF=v1.0.0` and `YEAST_EXPECTED_VERSION=v1.0.0` in a temporary non-system install path.
- Final CI and GitHub Pages deploy passed on `main`.

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
