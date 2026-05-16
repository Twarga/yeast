# Yeast Prototype Inventory

Status: M0-T2 reference notes

Branch: `v2-rebuild`

Purpose:

This file captures what the current Yeast MVP teaches us before the v2 rebuild removes or rewrites old implementation files. The prototype is not the architecture for v2. It is evidence. It shows which ideas already worked, which implementation shortcuts should not be copied, and which tests are worth porting.

The rule for the rebuild is simple:

- reuse concepts
- port small utilities when they still fit the v2 architecture
- do not copy tangled command-level workflow code directly
- do not keep global project assumptions
- do not preserve behavior only because the prototype happened to do it

## 1. Existing Prototype Shape

The prototype is a Go CLI using Cobra.

Current top-level implementation areas:

- `cmd/yeast`: command handlers, CLI rendering, lifecycle orchestration, process control, network flags, JSON output structs.
- `pkg/config`: `yeast.yaml` model, YAML loading, validation, defaults.
- `pkg/state`: JSON state file, process reconciliation, file lock.
- `pkg/vm`: QEMU disk creation, cloud-init seed preparation, QEMU launch, network argument generation, VM log rotation.
- `pkg/cloudinit`: user-data/meta-data generation, SSH key loading, seed ISO creation.
- `pkg/images`: trusted image manifest, download, checksum verification, progress hooks.
- `pkg/util`: port discovery, SSH readiness, byte-size parsing.
- `scripts/static-analysis.sh`: static analysis helper.

The prototype already contains enough behavior to prove the product direction is technically realistic. The v2 rebuild should not ignore it. But the responsibilities are in the wrong places for the long-term architecture.

## 2. Existing Commands

Current commands found in `cmd/yeast`:

- `yeast init`
- `yeast doctor`
- `yeast pull`
- `yeast up`
- `yeast status`
- `yeast ssh`
- `yeast down`
- `yeast halt`
- `yeast restart`
- `yeast destroy`

Useful command lessons:

- `init` already proves a starter `yeast.yaml` can be generated and validated before being moved into place.
- `doctor` already proves preflight checks are valuable before users hit confusing QEMU/KVM failures.
- `pull` already proves image download and verification should be first-class instead of hidden inside `up`.
- `up` already proves the lifecycle sequence: load config, lock state, reconcile state, create disk, generate cloud-init, start QEMU, wait for SSH, save state.
- `status` already proves users and tools need clear visibility into VM state.
- `ssh` already proves Yeast should hide host-forwarded port details from users.
- `down`, `halt`, `restart`, and `destroy` show useful lifecycle verbs, but v2 should decide final naming carefully before freezing commands.

Avoid copying:

- Command handlers currently own too much application logic.
- Commands directly coordinate config, state, runtime, output, and errors.
- This shape makes LabsBackery and Yeast MCP integrations harder because the CLI is the main product boundary.

v2 direction:

- Keep Cobra if desired, but make CLI thin.
- Move workflows into `internal/app`.
- Commands should parse input, call application services, and render results.
- Command behavior should map to stable app-level results, not ad hoc CLI structs.

## 3. Config Behavior

Current config model:

```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    user: yeast
    sudo: none
    user_data: |
      #cloud-config
    env:
      KEY: value
```

Useful ideas:

- Versioned config is correct and must remain.
- Instance names are validated against a safe pattern.
- Duplicate instance names are rejected.
- Image is required.
- Memory and CPU have defaults.
- Disk size is normalized.
- Linux username is validated.
- Sudo policy is explicit: `none`, `password`, `nopasswd`.
- Environment variable keys are validated.

Avoid copying:

- Config version `1` in the prototype does not need to become v2's final schema.
- Config currently lacks project-level networks, provisioning, templates, snapshots, and stable extension points.
- `user_data` is too raw as the main provisioning interface for the long-term product.

v2 direction:

- Keep the validation discipline.
- Design a v2 schema from the architecture document.
- Add versioned config migrations only after the v2 schema stabilizes.
- Keep raw cloud-init escape hatches possible, but do not make them the main user experience.

## 4. State And Lock Behavior

Current state behavior:

- State file path is `yeast.state` in the current working directory.
- State stores instances by name.
- Instance state includes name, PID, status, IP, and SSH port.
- Save writes a temp file then renames it.
- Lock file uses `<state>.lock`.
- Lock acquisition uses `O_CREATE|O_EXCL`.
- Lock metadata includes PID and creation time.
- Stale locks are detected when the owner process no longer exists or malformed metadata becomes old.
- In-process locking prevents same-process concurrent state mutation.
- Reconciliation checks whether a stored PID still looks like the expected QEMU process.

Useful ideas:

- State locking is non-negotiable and should be kept conceptually.
- Atomic save by temp file + rename is good.
- Stale lock detection is good.
- Reconciliation is correct product thinking: do not trust old state blindly.
- PID checks are useful for local runtime state.

Avoid copying:

- State is not project-safe enough.
- State is stored directly in the project folder as `yeast.state`, while architecture now calls for project identity and `~/.yeast/projects/<project-id>/state.json`.
- State keys by instance name alone are not enough for multi-project safety.
- Reconciliation currently depends on `~/.yeast/instances/<name>`, which causes collisions between projects.
- State has no schema version.
- State has no runtime paths, image/disk metadata, provisioning status, snapshot metadata, or network information.

v2 direction:

- Keep lock semantics, but move them under the state layer.
- Add state schema version.
- Store state under project-specific runtime paths.
- Make reconciliation a first-class service, not a helper hidden behind commands.

## 5. QEMU Runtime Behavior

Current runtime behavior:

- Creates qcow2 overlay disks with `qemu-img create -f qcow2 -F qcow2 -b <base> <overlay>`.
- Supports optional disk resize when requested size is larger than current virtual size.
- Starts `qemu-system-x86_64`.
- Uses `-enable-kvm`.
- Uses `-m <memory>`.
- Uses `-smp <cpus>`.
- Attaches instance disk as `if=virtio`.
- Attaches cloud-init seed ISO as raw cdrom.
- Uses `-nographic`.
- Detaches the QEMU process with its own process group.
- Writes QEMU stdout/stderr to `vm.log`.
- Rotates previous logs and prunes archives.

Useful ideas:

- Direct QEMU/KVM approach is viable for v1.
- qcow2 overlay over cached base image is the correct baseline.
- Disk resizing should be supported carefully.
- QEMU logs per instance are important for troubleshooting.
- Process group handling is important for clean stop behavior.

Avoid copying:

- Runtime code currently lives in `pkg/vm` and combines disk, runtime, cloud-init, networking, paths, and logs.
- Runtime path is `~/.yeast/instances/<name>`, which is not project-safe.
- QEMU command building is not separated enough for tests.
- Runtime interface does not exist yet.
- Start behavior is tied directly to SSH readiness and state creation through command helpers.

v2 direction:

- Create a runtime interface in `internal/runtime`.
- Implement direct QEMU in `internal/runtime/qemu`.
- Split disk preparation, command building, process start, process stop, and log handling.
- Unit-test QEMU command construction without launching real VMs.

## 6. Networking Behavior

Current network modes:

- `user`: QEMU user networking with SSH host forwarding.
- `private`: QEMU user networking with `restrict=on` and SSH host forwarding.
- `bridge`: bridge network plus separate restricted management network for SSH forwarding.

Useful ideas:

- Separating management access from lab traffic is the correct long-term idea.
- `user` networking is a good v0.1 default because it avoids host setup.
- Network flag validation is useful.
- Bridge mode should remain an advanced feature later.

Avoid copying:

- Networking is controlled by command flags, not project config.
- Network model is too global: all instances receive the same command-level network option.
- No project-level network definitions.
- No static private IP model.
- `private` mode does not yet equal the LabsBackery private topology model; it is only restricted user networking.

v2 direction:

- v0.1 should keep simple user-mode management networking.
- Multi-VM private lab networking belongs in a later milestone.
- The config schema should reserve room for project-level networks.

## 7. Cloud-init Behavior

Current cloud-init behavior:

- Generates `user-data`.
- Generates `meta-data`.
- Creates seed ISO using `genisoimage`.
- Loads SSH public key from `~/.ssh/id_rsa.pub`, falling back to `~/.ssh/id_ed25519.pub`.
- Creates a default user named `yeast` if no user is provided.
- Supports sudo policy.
- Supports writing environment variables to `/etc/profile.d/yeast-env.sh`.
- Allows raw user-provided cloud-init content and adds `#cloud-config` if missing.

Useful ideas:

- cloud-init is the correct first-boot mechanism.
- Seed ISO strategy is practical and understandable.
- SSH key injection is required for `yeast ssh` and guest control.
- User and sudo policy should stay explicit.
- Raw user-data escape hatch can be useful for advanced users.

Avoid copying:

- SSH key discovery is too limited and not configurable.
- User-data generation is too close to runtime code.
- There is no clear separation between first-boot bootstrap and future post-boot provisioning.
- `genisoimage` is assumed as the only ISO builder.

v2 direction:

- Put cloud-init in `internal/provision/cloudinit`.
- Make SSH key path configurable or clearly discovered.
- Keep seed ISO generation, but `doctor` must explain missing ISO tooling.
- Separate bootstrap from provisioning in docs and code.

## 8. Image Download Behavior

Current image behavior:

- Trusted manifest has pinned Ubuntu 22.04 and 24.04 cloud image URLs.
- URLs use immutable release paths.
- SHA256 checksums are pinned.
- Downloader uses temp `.part` file names.
- Download verifies checksum before moving into place.
- Download retries retryable errors.
- HTTP 5xx and 429 are retryable.
- Progress sink supports attempt start, bytes transferred, retry scheduled, and attempt finished.
- Existing image manifest test checks immutable URLs and SHA256 format.

Useful ideas:

- Built-in trusted image manifest is correct for v1.
- Pinned immutable URLs are important for trust.
- SHA256 verification is mandatory.
- Temp file + checksum + rename is correct.
- Progress hooks should survive in v2, but behind output/event abstractions.

Avoid copying:

- Manifest is hardcoded only in package code; v2 may still start hardcoded, but should leave room for official manifest updates later.
- Cache path behavior should move under the image layer and project/global path resolver.
- Progress should emit app events, not talk directly to CLI presentation.

v2 direction:

- Keep `TrustedImage` concept.
- Keep checksum verification.
- Move cache path decisions into `internal/images`.
- Add schema-aware result types for JSON and UI use.

## 9. Output And JSON Behavior

Current output behavior:

- Global `--json` flag.
- JSON command envelope contains schema, command, ok, data, and error.
- Error object has code and message.
- Command-specific data structs exist for init, status, lifecycle commands, pull, doctor.
- Human output has colored sections, warnings, errors, key/value lines, and progress.

Useful ideas:

- One command should support both human and JSON output.
- Stable error codes are valuable.
- Envelope-style JSON is a good starting point.
- Human output should stay readable and concise.

Avoid copying:

- Output types currently live in the CLI package.
- JSON schema names are not centrally versioned.
- Human and JSON output are not yet rendered from shared lifecycle events.
- Error codes are useful but not designed as a formal contract.

v2 direction:

- Move result types and output rendering under `internal/output`.
- Define stable envelopes early.
- Keep human and JSON output generated from the same app results/events.
- Add tests for JSON shapes before LabsBackery or MCP depend on them.

## 10. Doctor Behavior

Current doctor checks:

- `qemu-system-x86_64`
- `qemu-img`
- `genisoimage`
- SSH client
- `/dev/kvm`
- KVM group membership
- SSH keys
- cache directory

Useful ideas:

- Doctor is important and should be kept early.
- Remediation text is user-facing product quality, not a small detail.
- Doctor should support both human output and JSON.

Avoid copying:

- Some fix messages are broad and duplicated.
- Doctor should eventually be split into testable checks under an app/service layer.
- Host capability checks should avoid being buried in the command file.

v2 direction:

- Make doctor an application workflow.
- Keep checks modular.
- Return structured check results with severity and fixes.

## 11. Process Control Behavior

Current stop behavior:

- Checks if PID is valid and running.
- Sends SIGTERM first.
- Waits for a grace timeout.
- Sends SIGKILL if process does not exit.
- Waits again and reports if process is still running.

Useful ideas:

- Graceful termination before forced kill is correct.
- Stop behavior should be centralized in runtime/process layer.
- Process checks need to avoid killing unrelated processes.

Avoid copying:

- Current process control relies mostly on PID and broad process checks.
- v2 needs stronger runtime identity verification through runtime paths and command-line checks.

v2 direction:

- Keep SIGTERM/SIGKILL sequence.
- Couple stop operations with runtime identity checks.
- Store runtime metadata in state.

## 12. Tests Worth Porting

Existing test files:

- `pkg/config/loader_test.go`
- `pkg/images/manifest_test.go`
- `pkg/util/size_test.go`

Tests worth porting:

- Config defaults.
- Missing image rejection.
- Invalid instance name rejection.
- Invalid env key rejection.
- Invalid sudo policy rejection.
- Linux username validation.
- Disk size normalization.
- Unsupported config version rejection.
- Empty instance list rejection.
- Duplicate instance names rejection.
- Invalid CPU count rejection.
- Invalid disk size rejection.
- Trusted manifest immutable URL check.
- Trusted manifest SHA256 format check.
- Byte-size parse and normalize tests.

Missing tests v2 should add:

- Project ID creation/load.
- Runtime path resolver.
- State schema version and migration behavior.
- State lock behavior.
- Reconciliation behavior.
- QEMU command builder.
- Cloud-init user-data generation.
- JSON output contract.
- Application workflow tests with fake runtime.

## 13. Code That Should Not Be Copied Directly

Do not directly copy these shapes into v2:

- `cmd/yeast/up.go` as the owner of lifecycle logic.
- `cmd/yeast/lifecycle_helpers.go` as the app layer.
- `pkg/vm/vm.go` as one large VM abstraction.
- `stateFilePath = "yeast.state"` as the long-term state model.
- `~/.yeast/instances/<name>` as runtime storage.
- Command-level network flags as the final networking model.
- CLI-owned JSON structs as the integration contract.
- Raw `user_data` as the main provisioning interface.

These files are useful references, not foundations.

## 14. Reusable Ideas

Strong ideas to preserve:

- Cobra CLI can stay if it remains thin.
- Versioned YAML config.
- Strict config validation and defaults.
- `doctor` before lifecycle debugging.
- Trusted image manifest with pinned SHA256.
- Image download to temp file, verify, then rename.
- qcow2 overlay disks over cached base images.
- cloud-init seed ISO.
- SSH key injection.
- Host SSH forwarding for v0.1.
- Wait for SSH before declaring a VM ready.
- JSON envelope with stable error codes.
- Human output separate from machine output.
- Atomic state save.
- File lock with stale-lock recovery.
- State reconciliation before status or lifecycle actions.
- Graceful stop before forced kill.
- VM log rotation.
- Tests around parsing, validation, and manifest trust.

## 15. Rebuild Decision

The prototype has enough working concepts to guide v2, but the structure does not match the target architecture.

Final decision for M0:

- Start v2 with a clean internal architecture.
- Keep the old implementation temporarily until inventory and skeleton are complete.
- Port only small, well-understood utilities after they are placed in the correct v2 package.
- Recreate lifecycle workflows through `internal/app`, not through copied command handlers.
- Do not delete prototype files until `M0-T5`.

