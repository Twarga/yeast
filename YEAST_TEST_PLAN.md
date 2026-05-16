# Yeast Test Plan

Status: Draft v1  
Owner: Twarga / TwargaOps  
Phase: 10 - Testing / QA  
Purpose: Define how Yeast v2 will be proven reliable before release

## 1. Purpose

Testing for Yeast is not only about passing unit tests.

Yeast controls real virtual machines, real disks, host processes, SSH, cloud-init, and eventually private networks and snapshots. A broken Yeast command can leave dead processes, stale state, corrupted disks, confusing ports, or destroyed files.

The purpose of this file is to define how Yeast will be tested at each level:

- fast unit tests
- integration tests
- host-dependent QEMU/KVM tests
- manual release checklists
- failure tests
- JSON contract tests
- future LabsBackery and MCP compatibility tests

The standard is:

> Yeast is not ready when the code compiles. Yeast is ready when the intended workflow works from a clean machine and failure cases produce clear, safe behavior.

## 2. Testing Principles

### Principle 1: Test The Core Workflow First

The most important test is the user's first success:

```text
init -> pull -> up -> status -> ssh -> down -> destroy
```

If this fails, nothing else matters.

### Principle 2: Status Must Not Lie

Yeast status should reflect reality, not only old state.

If a VM process dies outside Yeast, status must reconcile it.

### Principle 3: Destructive Operations Must Be Path-Safe

Destroy, cleanup, snapshot restore, and file provisioning must never escape Yeast-controlled directories.

### Principle 4: JSON Is A Contract

Human output can change more freely. JSON output is for tools and should be tested as a contract.

### Principle 5: Host-Dependent Tests Are Separate

Some tests need QEMU/KVM and a real Linux host. These should not block fast local unit tests, but they must pass before release.

### Principle 6: Failure Cases Matter

Infrastructure tools earn trust by failing clearly and safely.

## 3. Test Categories

## 3.1 Fast Unit Tests

Purpose:

Test pure logic without starting real VMs.

Should run quickly with:

```text
go test ./...
```

Coverage areas:

- config parsing
- config validation
- default application
- size parsing
- project ID loading/generation
- path safety
- state load/save
- state migration
- lock behavior
- image manifest lookup
- checksum verification
- QEMU command construction
- cloud-init generation
- JSON output shape
- error code mapping

## 3.2 Integration Tests Without Real QEMU

Purpose:

Test layers together using fake runtime/guest implementations.

These tests prove the application workflows coordinate correctly without needing real VMs.

Coverage areas:

- `up` workflow with fake runtime
- `down` workflow with fake runtime
- `status` workflow with fake state/process info
- `destroy` workflow with temporary directories
- output rendering from workflow results
- error propagation from lower layers

## 3.3 Host-Dependent Integration Tests

Purpose:

Test Yeast against real QEMU/KVM on a Linux host.

These tests are slower and environment-dependent.

They should be run before releases and major runtime changes.

Coverage areas:

- pull image
- create qcow2 overlay
- boot Ubuntu VM
- generate cloud-init seed ISO
- wait for SSH
- stop VM
- restart VM
- destroy VM
- reconcile stale PID

## 3.4 Manual Release Checklist

Purpose:

A human runs the product like a new user.

This catches documentation, UX, and environment issues automated tests miss.

Should be required before every public release.

## 3.5 Future LabsBackery Compatibility Tests

Purpose:

Ensure LabsBackery can call Yeast and understand the output.

Coverage areas:

- parse `status --json`
- parse `up --json`
- parse lifecycle errors
- handle reset/snapshot later
- receive SSH connection info

## 3.6 Future Yeast MCP Compatibility Tests

Purpose:

Ensure AI agent operations can use Yeast safely.

Coverage areas:

- structured `exec` result
- stdout/stderr/exit code
- logs output limits
- destructive command separation
- approval boundaries later

## 4. Unit Test Plan

### Config Tests

Test cases:

- valid minimal config loads
- default memory is applied
- default CPU count is applied
- default user is applied
- duplicate instance names fail
- invalid instance name fails
- missing image fails
- invalid memory fails
- invalid CPU count fails
- invalid disk size fails
- invalid sudo policy fails
- invalid env key fails
- env values with unsupported newlines fail

Definition of done:

- config validation prevents bad runtime plans before any VM work starts

### Project Tests

Test cases:

- new project ID is generated
- existing project ID is loaded
- project ID survives folder move
- project runtime path is under `~/.yeast/projects/<project-id>`
- instance runtime path is under project runtime path
- unsafe instance names cannot escape directories
- missing project metadata produces clear behavior

Definition of done:

- project identity prevents name collisions and path escapes

### State Tests

Test cases:

- missing state creates empty state
- valid state loads
- state saves atomically
- corrupt state returns clear error
- schema version is checked
- nil instance map is repaired
- lock file blocks concurrent access
- stale lock can be recovered
- save failure does not leave broken state file

Definition of done:

- state is safe, inspectable, and recoverable

### Image Tests

Test cases:

- supported image names list correctly
- known image resolves
- unknown image fails
- checksum verification passes with correct checksum
- checksum verification fails with wrong checksum
- partial downloads are cleaned up on failure
- existing valid image is reused

Definition of done:

- image cache can be trusted

### Runtime Command Tests

Test cases:

- QEMU command includes KVM flag
- QEMU command includes memory
- QEMU command includes CPUs
- QEMU command includes disk drive
- QEMU command includes cloud-init seed ISO
- QEMU command includes management network and SSH forwarding
- bridge/private args are validated later
- unsafe values are rejected before command construction

Definition of done:

- QEMU commands are predictable and do not rely on unsafe shell strings

### Cloud-init Tests

Test cases:

- generated user-data includes hostname
- generated user-data includes user
- generated user-data includes SSH key
- sudo policy is correct
- env values are quoted safely
- custom user-data header is handled
- meta-data includes instance ID and hostname

Definition of done:

- first boot guest setup is predictable

### Output Tests

Test cases:

- success JSON includes schema, command, ok, data
- error JSON includes schema, command, ok=false, error code/message
- status JSON includes instances
- lifecycle JSON includes per-instance results
- human output does not affect JSON output

Definition of done:

- external tools can parse output safely

## 5. Application Workflow Test Plan

These tests should use fake runtime and fake guest clients.

### `up` Workflow

Test cases:

- starts stopped instance
- skips already running instance
- handles disk preparation failure
- handles runtime start failure
- handles SSH readiness timeout
- saves state after success
- records failure in result
- does not save running state if start fails before PID exists

Definition of done:

- `up` workflow behavior is deterministic without real QEMU

### `status` Workflow

Test cases:

- empty state returns no instances
- running instance with alive process stays running
- running instance with dead process becomes stopped
- status output is sorted by name
- reconciled state is saved

Definition of done:

- status reflects process reality

### `down` Workflow

Test cases:

- stops running instance
- skips stopped instance
- handles missing instance
- handles stop failure
- updates state after stop

Definition of done:

- down is safe and clear

### `destroy` Workflow

Test cases:

- stops running instance before destroy
- removes only project instance directory
- removes state entry
- handles already absent instance
- never removes image cache
- fails safely on unsafe path

Definition of done:

- destroy cannot accidentally remove unrelated files

## 6. Host-Dependent v0.1 Manual Test Checklist

Run on a clean Linux host with KVM available.

### Environment

- Confirm Linux host.
- Confirm `/dev/kvm` exists.
- Confirm user can access KVM.
- Confirm `qemu-system-x86_64` exists.
- Confirm `qemu-img` exists.
- Confirm `genisoimage` or replacement exists.
- Confirm SSH client exists.
- Confirm SSH public key exists.

### Fresh Project Lifecycle

Steps:

1. Create empty test directory.
2. Run `yeast init`.
3. Inspect generated `yeast.yaml`.
4. Run `yeast doctor`.
5. Run `yeast pull ubuntu-24.04`.
6. Run `yeast up`.
7. Confirm VM reaches ready state.
8. Run `yeast status`.
9. Run `yeast ssh <name>`.
10. Inside VM, run `whoami` and `hostname`.
11. Exit SSH.
12. Run `yeast down`.
13. Run `yeast status`.
14. Run `yeast up` again.
15. Confirm same project still works.
16. Run `yeast destroy`.
17. Confirm runtime files removed.
18. Confirm cached image remains.

Pass criteria:

- No command panics.
- Error messages are clear if environment is missing dependencies.
- VM becomes reachable by SSH.
- State matches actual process state.
- Destroy removes only project runtime files.

## 7. Failure Test Plan

### Missing Dependency Tests

Cases:

- QEMU missing
- qemu-img missing
- genisoimage missing
- SSH key missing
- image not pulled
- KVM unavailable

Expected:

- `doctor` reports clear blocker
- runtime command fails with actionable error
- JSON error includes stable code

### Bad Config Tests

Cases:

- invalid YAML
- unsupported version
- duplicate names
- invalid disk size
- invalid username
- unknown image

Expected:

- command fails before runtime work
- no disk created
- no state mutation
- clear error points to config problem

### Runtime Failure Tests

Cases:

- SSH timeout
- QEMU process exits quickly
- port collision
- disk creation failure
- permission denied on runtime directory

Expected:

- state does not lie
- partial resources are cleaned where safe
- logs are available
- human output suggests next step

### State Failure Tests

Cases:

- corrupt state file
- stale lock
- permission denied writing state
- running PID dead

Expected:

- corrupt state gives clear error
- stale lock is recoverable if owner dead
- status reconciles dead process
- no silent state corruption

## 8. JSON Contract Tests

Commands requiring JSON tests in v0.1:

- `init --json`
- `doctor --json`
- `pull --json`
- `up --json`
- `status --json`
- `down --json`
- `destroy --json`

Required JSON properties:

- schema
- command
- ok
- data on success
- error on failure

Error object:

- code
- message
- details optional

Rule:

JSON output must not include human-only formatting, ANSI colors, or progress spinners.

## 9. Provisioning Test Plan Later

For v0.3.

Test cases:

- package install succeeds
- package install failure is visible
- file upload succeeds
- file upload path validation works
- shell command succeeds
- shell command failure returns exit code
- provisioning can be rerun
- provisioning logs are available
- provisioning status is saved

Demo test:

```text
Ubuntu + Caddy + index.html + service verification
```

Pass criteria:

- `curl localhost` inside VM returns expected page.

## 10. Snapshot Test Plan Later

For v0.4.

Test cases:

- create snapshot
- list snapshot
- restore snapshot
- delete snapshot
- restore after file modification
- restore all instances
- refuse or stop before restoring running VM
- snapshot metadata is correct

Pass criteria:

- VM returns to known clean state.
- Disk is not corrupted.

## 11. Networking Test Plan Later

For v0.5.

Test cases:

- create private network
- start two VMs on same network
- static IP assignment works
- attacker can ping target
- target can expose service to attacker
- management SSH still works
- destroy cleans network artifacts

Pass criteria:

- multi-VM lab topology works predictably.

## 12. Guest Control Test Plan Later

For v0.6.

Test cases:

- `exec` returns stdout
- `exec` returns stderr
- `exec` returns exit code
- command timeout works
- copy file to guest
- copy file from guest
- logs command reads VM log
- inspect returns structured info

Pass criteria:

- Yeast MCP can safely use structured guest operations.

## 13. LabsBackery Compatibility Test Later

Test scenario:

LabsBackery calls Yeast through CLI/JSON.

Flow:

1. LabsBackery creates lab folder.
2. LabsBackery writes Yeast config.
3. LabsBackery runs `yeast up --json`.
4. LabsBackery parses progress/result.
5. LabsBackery runs `yeast status --json`.
6. LabsBackery opens terminal using SSH info.
7. LabsBackery runs reset through snapshot/restore later.
8. LabsBackery destroys lab.

Pass criteria:

- LabsBackery never calls QEMU directly.
- LabsBackery never parses human output.
- Yeast JSON is enough for lab lifecycle.

## 14. Release Gates

v0.1 cannot be released unless:

- fast tests pass
- JSON contract tests pass
- manual lifecycle checklist passes
- README quickstart works
- doctor catches missing dependencies
- no known path-safety issue exists
- destroy is tested carefully
- release notes list known limitations

v0.3 cannot be released unless:

- Caddy provisioning demo works
- provisioning failure is debuggable
- rerun behavior is documented

v0.4 cannot be released unless:

- snapshot/restore is tested repeatedly
- restore safety rules are documented
- disk corruption risk is understood

v0.5 cannot be released unless:

- two-VM private network lab works
- static IP docs exist
- host permission requirements are documented

## 15. Test Environments

Minimum environments:

- developer machine
- fresh Linux VM if possible
- clean Debian/Ubuntu host

Host requirements:

- KVM available
- qemu-system-x86_64
- qemu-img
- genisoimage or replacement
- SSH client
- SSH public key

Future environments:

- Fedora
- Arch
- Hetzner bare-metal server

## 16. Test Data And Examples

Example projects:

- `examples/ubuntu-basic`
- `examples/caddy-web`
- `examples/two-vm-lab` later

Each example should include:

- config
- expected behavior
- commands to run
- cleanup command

## 17. Definition Of Tested For v0.1

Yeast v0.1 is tested when:

- unit tests cover config, state, paths, images, output
- application workflows are tested with fake runtime
- manual QEMU lifecycle passes on Linux/KVM
- JSON outputs are parseable
- destroy is path-safe
- docs quickstart has been run exactly as written

## 18. Final QA Rule

Before every release, run Yeast like a stranger.

Do not test only the path you know.

Use the docs.

Start from an empty directory.

Assume nothing.

If the product fails there, the release is not ready.
