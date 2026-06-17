# Yeast Installer + Premium CLI UX Remake Plan

Status: approved design direction, not implemented yet
Date: 2026-06-17
Owner: TwargaOps / Yeast
Audience: implementation AI or engineer taking over the installer + CLI UX remake

## 1. Purpose

This document is the full handoff plan for remaking the Yeast installation experience and the human-facing CLI UX.

The target outcome is:

- Yeast is extremely easy to install on a fresh Linux machine.
- The installer feels polished, clear, and trustworthy.
- The CLI keeps its normal command style, but human mode feels premium.
- JSON and automation behavior remain stable and boring.
- WSL and distro support are honest, not hand-wavy.
- The new experience reflects real Yeast v1.1 behavior, not imagined workflows.

This is not a generic "make it prettier" note. It is a concrete product and implementation plan.

## 2. Locked Product Decisions

These decisions are already made and should not be reopened during implementation unless the owner explicitly changes them.

### 2.1 Installer model

- Use a hybrid installer.
- Default experience should work as a one-command install.
- When running in a real TTY, the installer may switch to a more guided polished experience.

### 2.2 Platform support model

- First-class: Ubuntu and Debian.
- Supported but not first-class: Fedora and Arch.
- WSL2: beta, explicitly documented as unpredictable and degraded.
- Best-effort only: other Linux distributions.
- No macOS support.

### 2.3 CLI model

- Yeast remains a premium normal CLI.
- Do not turn Yeast into a TUI-first product.
- Keep commands like `yeast up`, `yeast status`, `yeast doctor`, `yeast ssh`.
- Improve human UX heavily.
- Keep `--json` and event-driven automation safe and stable.

### 2.4 Binary delivery model

- Default install path must be release-binary-first.
- Source build must become fallback or advanced mode only.
- Installing Go is not part of the normal end-user path.

## 3. Current State Audit

This section describes the real problems in the current repo that justify the remake.

### 3.1 Installer problems

- `install.sh` is too large and tries to be bootstrapper, dependency manager, build system, and onboarding script at the same time.
- The default path is source-build-first, which is heavier and more failure-prone than a release binary install.
- The script still contains stale first-run guidance like `yeast pull ubuntu-24.04`, even though `yeast up` can auto-download images.
- The script installs build dependencies for regular users even when they only need the released CLI binary.
- The script has richer environment logic than the Go `doctor` command, which creates product inconsistency.
- The script currently suggests TCG fallback on WSL/container when KVM is missing, but `yeast doctor` still treats missing `/dev/kvm` as a blocker.

### 3.2 CLI UX problems

- Current progress UX is functional but visually thin.
- The TTY progress sink is basically spinner plus one-line updates.
- Human output uses `lipgloss`, but the product does not feel like it has a strong visual system yet.
- Command feedback is inconsistent across workflows.
- There is no unified "Yeast human mode" language for success, warning, remediation, or next steps.
- There is no structured guided remediation path for host setup problems.

### 3.3 Product truthfulness problems

- WSL behavior is not explained consistently enough.
- KVM requirements are not represented the same way in installer, doctor, and docs.
- Release/update/install need one coherent story.
- Host architecture support must be described carefully. The runtime currently expects `qemu-system-x86_64`, so x86_64 native Linux is the safe primary target.

## 4. North Star Experience

The desired user experience should feel like this:

1. User runs one install command from docs or landing page.
2. Installer immediately identifies:
   - distro family
   - package manager
   - native Linux vs WSL2 vs container
   - architecture
   - whether Yeast is already installed
3. Installer clearly shows what it is about to do.
4. Installer installs only what is actually needed.
5. Installer installs the Yeast release binary quickly.
6. Installer hands off to a rich host readiness check and optional remediation flow.
7. The summary is crystal clear:
   - what succeeded
   - what is degraded
   - what still needs manual action
   - the exact next commands to start using Yeast
8. After installation, commands like `yeast doctor`, `yeast up`, `yeast status`, and `yeast init` feel deliberate, premium, and alive.

The experience should feel smooth and modern, but never fake, noisy, or confusing.

## 5. Non-Goals

The implementation should explicitly avoid these traps.

- Do not build a full-screen TUI shell around all commands.
- Do not break JSON output or automation workflows.
- Do not make animations mandatory.
- Do not rely on fake typing effects inside the real CLI.
- Do not silently perform aggressive system mutation like `/dev/kvm` chmod/chown unless the user explicitly opts in.
- Do not keep Go installation in the normal user install path.
- Do not over-promise support on platforms that are only partially validated.
- Do not add random flashy UI if it makes output harder to read.

## 6. Support Matrix and Product Truth

### 6.1 Official support tiers

| Tier | Environment | Support level | Notes |
|---|---|---|---|
| A | x86_64 Ubuntu/Debian native Linux with KVM | first-class | best install path and best test coverage |
| B | x86_64 Fedora native Linux with KVM | supported | package mapping and doctor fix flow should work |
| B | x86_64 Arch native Linux with KVM | supported | package mapping and doctor fix flow should work |
| C | WSL2 | beta | document risk, degraded performance, and unpredictability |
| C | other Linux distros | best-effort | no premium guarantees |

### 6.2 Architecture support truth

- x86_64 Linux is the main supported host target.
- Arm64 packaging may exist for binary distribution, but runtime support must not be marketed as first-class until the runtime and dependency model are explicitly validated for it.
- If the product cannot safely provide the same outcome on arm64 hosts, the installer and doctor must say so honestly.

### 6.3 WSL truth

- WSL is not a first-class Yeast platform.
- Docs should tell users WSL support is beta and potentially unpredictable.
- The installer and doctor must present WSL as degraded mode, not "everything is fine."
- If KVM is unavailable, the product should explain the consequence in plain language:
  - slower emulation
  - potentially unsupported workflows
  - not equivalent to native Linux with KVM

## 7. High-Level Architecture

The current installer is too much Bash and not enough product logic.

The new architecture should move the smart experience into Go where Yeast already has:

- consistent platform behavior
- better terminal rendering
- shared UX components
- easier testing

### 7.1 New split of responsibilities

#### Layer A: bootstrap shell script

`install.sh` becomes a thin bootstrapper.

Responsibilities:

- detect absolute minimum bootstrap prerequisites
- determine OS/arch at a very high level
- download Yeast release artifact
- verify checksums
- place binary in install location
- invoke Yeast for host validation/remediation

Non-responsibilities:

- no full source build by default
- no Go install by default
- no giant dependency intelligence graph
- no long-term ownership of the product UX

#### Layer B: host setup engine inside Yeast

Move the smart host logic into Go.

Responsibilities:

- dependency detection
- environment classification
- readiness checks
- fix plan generation
- fix execution
- user-facing progress and remediation summaries

This is the right layer for the premium UX.

#### Layer C: shared human UI system

Introduce a proper human UX layer used by:

- installer handoff flows
- `yeast doctor`
- `yeast up`
- `yeast status`
- `yeast init`
- image-related operations

#### Layer D: machine output contract

Maintain or improve the current machine-safe contract for:

- `--json`
- `--events`
- `--quiet`
- CI or non-TTY environments

## 8. Recommended Command Strategy

Do not explode the public command surface unless needed.

### 8.1 Keep existing commands

Existing commands that remain important:

- `yeast doctor`
- `yeast init`
- `yeast up`
- `yeast status`
- `yeast inspect`
- `yeast down`
- `yeast destroy`
- `yeast pull`
- `yeast images`
- `yeast logs`
- `yeast ssh`
- `yeast exec`
- `yeast copy`
- `yeast snapshot`
- `yeast snapshots`
- `yeast restore`
- `yeast delete-snapshot`
- `yeast provision`
- `yeast update`
- `yeast docs`

### 8.2 Add capability through flags first

Preferred evolution:

- `yeast doctor` stays the main host readiness command.
- Add guided remediation through flags instead of immediately inventing a brand-new top-level command.

Recommended new behavior:

- `yeast doctor` = read-only check
- `yeast doctor --fix` = attempt guided remediation
- `yeast doctor --fix --yes` = non-interactive fix path where safe
- `yeast doctor --json` = machine-readable output

This keeps the mental model simple.

## 9. Installer Remake Plan

### 9.1 New installer philosophy

The install command should be fast, safe, and honest.

The default user path should be:

1. bootstrap
2. install binary
3. validate host
4. fix host if the user allows it
5. print first steps

That is the right experience for a released CLI product.

### 9.2 New installer modes

#### Mode 1: automatic mode

Used when:

- stdin/stdout are not both TTYs
- CI is detected
- `--yes` is provided
- a plain machine-driven install is desired

Behavior:

- minimal prompts
- clear logs
- no decorative animations
- deterministic and copy-paste friendly

#### Mode 2: interactive mode

Used when:

- a real terminal is present
- the user is running interactively
- no explicit plain/non-interactive override is set

Behavior:

- richer progress stages
- better status cards
- remediation confirmations
- polished summaries
- tasteful motion

### 9.3 Bootstrap shell script responsibilities in detail

The new `install.sh` should only handle these steps:

1. detect shell-safe essentials:
   - `uname`
   - `id`
   - `mktemp`
   - one of `curl` or `wget`
   - `tar`
2. detect platform:
   - distro family
   - arch
   - WSL or not
3. compute release artifact URL
4. fetch artifact and checksums
5. verify checksum
6. install binary to target path
7. optionally run `yeast doctor` or `yeast doctor --fix`
8. print fallback instructions if the handoff stage cannot continue

The script should stop trying to own full host setup logic.

### 9.4 Release artifact behavior

The script should install from GitHub release artifacts by default.

Requirements:

- use versioned release assets
- install binary named `yeast`
- verify checksum against release-provided sums
- fail loudly if the artifact naming or checksum data does not match

The release pipeline, update flow, and install flow must all agree on:

- version format
- artifact names
- embedded binary version
- checksum file naming

### 9.5 Source-build fallback

Source build is allowed only as fallback or advanced path.

Examples:

- unsupported release artifact for the host
- contributor mode
- explicit `YEAST_INSTALL_MODE=source`

Rules:

- source mode should be explicit
- source mode may install Go
- source mode should not be the default documented path

### 9.6 Post-install handoff

After binary placement, the script should hand off to Yeast itself.

Recommended handoff command:

`yeast doctor --fix` in interactive terminals  
`yeast doctor` or `yeast doctor --fix --yes` in non-interactive workflows, depending on explicit user choice

This is where the premium UX should begin.

## 10. Host Dependency and Remediation Design

### 10.1 Core principle

Dependency detection and remediation should live in Go, not mostly in Bash.

### 10.2 Required dependency categories

At minimum the host readiness system must reason about:

- QEMU system binary
- QEMU image tooling
- ISO builder
- SSH client
- SSH public key availability
- KVM device presence and accessibility
- Yeast cache directories

### 10.3 Supported binary alternatives

The current product reality should be reflected accurately:

- QEMU system: `qemu-system-x86_64`
- QEMU image utility: `qemu-img`
- ISO builder alternatives:
  - `genisoimage`
  - `mkisofs`
  - `xorriso` via compatibility wrapper or direct supported path

The doctor logic must stop being narrower than the installer.

### 10.4 Package mapping model

Create a small explicit package mapping layer by distro family.

Ubuntu/Debian target packages:

- `qemu-system-x86`
- `qemu-utils`
- `genisoimage`
- `openssh-client`

Fedora target packages:

- `qemu-system-x86`
- `qemu-img` or distro equivalent
- `genisoimage` or supported ISO-builder equivalent
- `openssh-clients`

Arch target packages:

- `qemu-base` or chosen supported QEMU package set
- supported ISO-builder package
- `openssh`

Important rule:

- Do not keep an unbounded package-name guessing game if a smaller stable supported map is enough.
- Prefer a tested support map over "try everything and hope."

### 10.5 KVM policy

KVM handling must be explicit and careful.

Rules:

- Native Linux without `/dev/kvm` should usually be a blocker.
- WSL2 without `/dev/kvm` should be reported as degraded beta mode, not falsely okay.
- Containers without `/dev/kvm` should also be treated as degraded or unsupported depending on current product guarantees.
- Direct `/dev/kvm` permission mutation must be opt-in.
- Adding the user to the `kvm` group is acceptable when clearly explained and confirmed.
- The UX must explain when a logout/login or shell restart is required for new group membership to apply.

### 10.6 SSH key policy

If no SSH public key exists:

- `doctor` should explain why Yeast needs it
- interactive mode may offer to generate one
- non-interactive mode should print the exact command and reason

### 10.7 WSL policy

If WSL is detected:

- show a beta warning early
- explain likely limitations
- avoid claiming parity with native Linux
- if fix automation is not safe, print manual steps instead of bluffing

## 11. New Internal Package Structure

This section proposes a clean code structure for the remake.

### 11.1 `internal/host`

New package for host environment logic.

Sub-responsibilities:

- environment detection
- distro classification
- package manager abstraction
- dependency checks
- fix plan generation
- fix execution

Suggested files:

- `internal/host/environment.go`
- `internal/host/distro.go`
- `internal/host/checks.go`
- `internal/host/packages.go`
- `internal/host/fixplan.go`
- `internal/host/fixexec.go`

### 11.2 `internal/ui`

New package for shared human-facing UI behavior.

Sub-responsibilities:

- theme tokens
- spacing rules
- headings
- badges
- tables/cards
- standard next-step blocks
- warning/error/success blocks
- motion toggles

Suggested files:

- `internal/ui/theme.go`
- `internal/ui/components.go`
- `internal/ui/layout.go`
- `internal/ui/motion.go`
- `internal/ui/terminal.go`

### 11.3 `internal/output`

Refactor current output package to distinguish:

- plain machine-safe progress
- rich TTY progress
- summary renderers

Suggested direction:

- keep the current event-sink architecture
- introduce a richer renderer without breaking plain sinks

Possible files:

- `internal/output/progress_plain.go`
- `internal/output/progress_rich.go`
- `internal/output/summary.go`
- `internal/output/theme_adapter.go`

### 11.4 `internal/app`

Application workflows should orchestrate, not own rendering details.

Potential additions:

- host doctor fix workflow
- install/fix result summaries
- richer structured result objects for rendering

## 12. CLI UX Design System

### 12.1 Design principles

Human mode should feel:

- premium
- calm
- clear
- intentional
- high-signal
- Linux-native

It should not feel:

- noisy
- cartoonish
- fake
- emoji-driven
- full of random visual effects

### 12.2 Visual language

Use Yeast's existing visual identity:

- dark terminal-friendly aesthetic
- green as the main accent
- restrained gray for secondary information
- amber for warnings
- red for blockers/errors
- clean whitespace and alignment

### 12.3 Motion principles

Motion should communicate state, not decorate emptiness.

Allowed motion:

- smoother spinner behavior
- staged progress transitions
- subtle reveal of status groups
- success/failure transition replacement
- loading indicators with clear meaning

Avoid:

- fake typing animations for normal command output
- long decorative animations that slow real work
- flashing content
- motion that makes copy-paste or log reading worse

### 12.4 Motion control policy

Animations must automatically disable when:

- `--json` is active
- `--events` is active
- `--quiet` is active
- stdout is not a TTY
- `NO_COLOR` is set
- `TERM=dumb`
- CI environment is detected

Optional user controls should be considered:

- `YEAST_UI=auto|rich|plain`
- `YEAST_ANIMATION=auto|on|off`

## 13. Recommended Library Strategy

### 13.1 Keep

- `cobra` for command structure
- `lipgloss` for styling
- current event-driven workflow model

### 13.2 Consider adding

- `bubbletea` for richer transient progress or guided flows only if it improves the experience without turning Yeast into a full-screen app
- `bubbles` for better spinner/progress primitives if they reduce custom renderer complexity

### 13.3 Important rule

Do not adopt libraries just because they are "cool."

Every UI dependency must support this product direction:

- standard command usage remains standard
- output remains script-friendly when needed
- rich mode stays optional and context-aware

## 14. Command-by-Command UX Plan

### 14.1 `yeast doctor`

This command becomes the main host readiness experience.

Read-only mode should show:

- environment summary
- support tier
- blocker/warning counts
- grouped checks
- clear human explanation of each problem
- exact remediation suggestions

Fix mode should support:

- preview of proposed actions
- confirmation before privileged operations
- package install attempts
- SSH key generation offer
- KVM group membership updates
- final "what changed" summary

### 14.2 `yeast init`

Improve onboarding with:

- short success card
- what was created
- what file to edit
- exact next steps
- direct pointer to `yeast.yaml` docs

Do not leave the user wondering where to go next.

### 14.3 `yeast up`

This is the flagship workflow and should feel the best.

Target behavior:

- grouped progress by major stage
- one rich progress area, not scattered lines
- instance-aware status updates
- clear transitions:
  - loading project
  - validating config
  - preparing image
  - creating disks
  - booting VM
  - waiting for SSH
  - provisioning
  - ready
- stronger final summary:
  - instance name
  - management address
  - SSH hint
  - whether provisioning ran or was skipped

### 14.4 `yeast status`

Should feel like a concise control surface.

Show:

- project name/path
- instance list
- state
- management endpoint
- image info if useful
- snapshot count if cheap to show
- standout warnings if any instance is missing or inconsistent

### 14.5 `yeast inspect`

Should present deep details in clear sections instead of a wall of text.

Possible sections:

- identity
- runtime state
- networking
- image/disk
- paths
- timestamps

### 14.6 `yeast pull` and `yeast images`

These commands still matter, but the onboarding language should change.

Rules:

- do not center `yeast pull` in the default first-run path
- explain that `yeast up` can auto-download trusted images
- keep `pull` valuable for prefetching, browsing, or CI preparation

### 14.7 `yeast update`

Should feel safer and clearer.

Show:

- current version
- target version
- downloaded artifact name
- checksum verification result
- success confirmation

### 14.8 `yeast logs`, `ssh`, `exec`, `copy`

Improve small command ergonomics:

- better missing-instance errors
- better "instance not running" errors
- show the resolved management target when relevant
- keep success output short

### 14.9 Snapshot commands

Commands:

- `snapshot`
- `snapshots`
- `restore`
- `delete-snapshot`

Improve with:

- clearer precondition messaging
- cleaner result summaries
- explicit warnings when a VM must be stopped

## 15. Docs and Onboarding Impact

The CLI remake must be reflected in docs immediately.

Required docs outcomes:

- installation docs must use the new installer story
- Linux and WSL docs must be separate and honest
- quickstart must not lead with stale `yeast pull` flows unless intentionally teaching prefetch
- `write-yeast-yaml` and `yeast-yaml` reference must be directly linked from install/init/quickstart
- troubleshooting must match new doctor/fix behavior

## 16. Concrete Implementation Phases

This is the recommended implementation order. Do not attempt the whole remake in one giant patch.

### Phase 1: product truth and install path alignment

Goal:

- Make release/install/update/doctor tell one coherent story.

Tasks:

- shrink `install.sh` scope
- make release-binary path the default
- remove default Go install path from normal user flow
- ensure artifact/checksum/version naming is aligned
- remove stale `yeast pull` guidance from installer summary
- align doctor expectations with installer realities for ISO builders and environment reporting

Done when:

- fresh install uses release binary by default
- install summary points to valid next steps
- doctor no longer contradicts installer on obvious checks

### Phase 2: host diagnostics engine

Goal:

- Move host intelligence into Go.

Tasks:

- create `internal/host`
- model environment and support tiers
- model dependency checks
- model remediation plans
- implement package manager abstraction for supported tiers
- implement `doctor --fix` foundation

Done when:

- host checks are testable in Go
- fix plans are generated from structured data
- package suggestions are no longer hidden in Bash only

### Phase 3: premium UI system

Goal:

- Create the shared Yeast human-mode visual language.

Tasks:

- add `internal/ui`
- define theme tokens and component helpers
- define motion policy and TTY capability checks
- refactor output package to support rich vs plain modes cleanly

Done when:

- renderers use shared components instead of ad hoc styling
- rich mode can be turned off cleanly
- no command is forced into rich mode in automation contexts

### Phase 4: flagship workflow upgrades

Goal:

- Make the most important commands feel premium first.

Priority order:

1. `yeast doctor`
2. `yeast up`
3. `yeast status`
4. `yeast init`
5. `yeast update`

Done when:

- the most common user journey feels meaningfully better end to end

### Phase 5: secondary command polish

Goal:

- Make the rest of the CLI consistent.

Commands:

- `inspect`
- `pull`
- `images`
- `logs`
- `ssh`
- `exec`
- `copy`
- snapshot flows

### Phase 6: docs, smoke tests, and rollout validation

Goal:

- ensure the experience is real, documented, and shippable

Tasks:

- update installation docs
- update quickstart
- update troubleshooting
- run real host smoke tests
- confirm WSL docs are honest

## 17. File-by-File Change Map

This section gives the likely implementation footprint.

### Must change

- `/home/twarga/Projects/yeast/install.sh`
- `/home/twarga/Projects/yeast/cmd/yeast/doctor.go`
- `/home/twarga/Projects/yeast/internal/app/doctor.go`
- `/home/twarga/Projects/yeast/internal/output/progress.go`
- `/home/twarga/Projects/yeast/internal/output/spinner.go`
- `/home/twarga/Projects/yeast/internal/output/human.go`

### Likely add

- `/home/twarga/Projects/yeast/internal/host/environment.go`
- `/home/twarga/Projects/yeast/internal/host/distro.go`
- `/home/twarga/Projects/yeast/internal/host/checks.go`
- `/home/twarga/Projects/yeast/internal/host/packages.go`
- `/home/twarga/Projects/yeast/internal/host/fixplan.go`
- `/home/twarga/Projects/yeast/internal/host/fixexec.go`
- `/home/twarga/Projects/yeast/internal/ui/theme.go`
- `/home/twarga/Projects/yeast/internal/ui/components.go`
- `/home/twarga/Projects/yeast/internal/ui/layout.go`
- `/home/twarga/Projects/yeast/internal/ui/motion.go`
- `/home/twarga/Projects/yeast/internal/output/progress_rich.go`
- `/home/twarga/Projects/yeast/internal/output/progress_plain.go`

### Likely docs updates

- `/home/twarga/Projects/yeast/docs/getting-started/installation-linux.md`
- `/home/twarga/Projects/yeast/docs/getting-started/installation-windows-wsl.md`
- `/home/twarga/Projects/yeast/docs/getting-started/quickstart.md`
- `/home/twarga/Projects/yeast/docs/getting-started/write-yeast-yaml.md`
- `/home/twarga/Projects/yeast/docs/reference/commands.md`
- `/home/twarga/Projects/yeast/docs/troubleshooting/kvm.md`

## 18. Testing Plan

### 18.1 Unit tests

Add tests for:

- environment detection
- distro classification
- package mapping selection
- dependency check classification
- remediation plan generation
- rich renderer fallback behavior
- motion disable conditions

### 18.2 Golden tests

Add golden tests for:

- `doctor` human output
- `status` human output
- `up` final summaries
- installer summary text where practical

### 18.3 Integration tests

Add or strengthen tests for:

- `doctor --json`
- output mode switching
- installer source guard behavior if retained

### 18.4 Real-host smoke matrix

Run manual smoke tests on:

- Ubuntu fresh VM with KVM
- Debian fresh VM with KVM
- Fedora fresh VM with KVM
- Arch fresh VM with KVM if available
- WSL2 fresh setup

Smoke scenarios:

1. install from scratch
2. run `yeast doctor`
3. run fix flow if needed
4. `yeast init`
5. create first VM
6. `yeast up`
7. `yeast status`
8. `yeast ssh`

### 18.5 Regression requirements

Before claiming success:

- `go test ./... -count=1`
- fast suite if maintained
- relevant targeted command tests
- docs build if docs changed

## 19. Acceptance Criteria

The remake is successful only if all of these are true.

### Installer acceptance

- Fresh install does not require Go in the normal path.
- Fresh install uses release artifacts by default.
- Install output clearly shows platform/support status.
- Install summary contains correct next steps.
- WSL messaging is explicit and honest.

### Doctor acceptance

- `yeast doctor` explains the host state better than today.
- `yeast doctor --fix` can remediate supported issues in supported environments.
- Doctor does not contradict installer on KVM or ISO builder logic.

### CLI UX acceptance

- Human mode feels noticeably more polished.
- Rich mode never leaks into JSON/event workflows.
- Animations are subtle, meaningful, and disable correctly.
- The flagship journey `install -> doctor -> init -> up -> status` feels coherent.

### Product truth acceptance

- Docs, install script, doctor, and landing-page instructions tell the same story.
- No stale `yeast pull` requirement remains in the main happy path.

## 20. Risks and Pitfalls

### Risk 1: overbuilding Bash again

Bad outcome:

- installer becomes another giant shell framework

Mitigation:

- keep Bash as bootstrap only
- move intelligence to Go

### Risk 2: turning Yeast into a pseudo-TUI

Bad outcome:

- commands become harder to script or reason about

Mitigation:

- rich mode only for human TTY contexts
- maintain plain renderers and JSON paths

### Risk 3: dishonest platform promises

Bad outcome:

- users on WSL or weaker environments are misled

Mitigation:

- explicit support tier messaging everywhere

### Risk 4: visual noise over clarity

Bad outcome:

- "wow" UI but worse usability

Mitigation:

- motion must explain progress
- no decorative clutter
- no emoji-driven UX

## 21. Recommended Execution Rules for the Next AI

If another AI implements this plan, it should follow these rules:

1. Do not implement everything in one patch.
2. Start with install/release/doctor alignment before visual polish.
3. Do not invent unsupported Yeast features.
4. Do not change public commands unnecessarily.
5. Do not break `--json` or automation flows.
6. Add tests with each phase.
7. Run verification before claiming work complete.
8. Update docs as the command behavior changes.

## 22. First Implementation Slice Recommendation

If the work must begin immediately, the first slice should be:

1. make `install.sh` release-binary-first
2. remove default Go install from happy path
3. clean install summary and stale `yeast pull` guidance
4. align `doctor` ISO-builder checks with real supported tools
5. add environment classification groundwork for WSL/native Linux

Why this first:

- it removes the most user pain
- it reduces product contradiction
- it creates the correct base for later premium UX

## 23. Final Direction Summary

The right remake is not:

- "make Bash fancier"
- "turn Yeast into a TUI"
- "add random animation"

The right remake is:

- thin bootstrap installer
- Go-owned host intelligence
- release-binary-first distribution
- honest support-tier messaging
- premium human CLI rendering
- stable machine output

That is the path most likely to make Yeast feel smooth, serious, and trustworthy.
