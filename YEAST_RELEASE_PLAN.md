# Yeast Release Plan

Status: Draft v1  
Owner: Twarga / TwargaOps  
Phase: 12 - Release  
Purpose: Define how Yeast versions will be packaged, tested, documented, and published

## 1. Purpose

A release is not just pushing code to GitHub.

A release is the moment Yeast becomes consumable by someone outside the founder's head and local machine.

This file defines the release process for Yeast so each public version is:

- understandable
- testable
- installable
- documented
- honest about limitations
- safe enough for the version's promise

Yeast is an infrastructure tool. If the release is messy, users lose trust quickly.

## 2. Release Philosophy

Yeast should release early, but not carelessly.

The goal is not to wait until Yeast is perfect. The goal is to release versions that make clear promises and keep those promises.

For example:

v0.1 should not promise provisioning, snapshots, LabsBackery integration, or cloud support.

v0.1 should promise:

```text
You can install Yeast, create a project, pull a trusted image, start one VM, SSH into it, stop it, and destroy it.
```

If v0.1 does that reliably, it is a good release.

## 3. Versioning Policy

Yeast should use semantic versioning:

```text
MAJOR.MINOR.PATCH
```

Examples:

```text
0.1.0
0.2.0
0.2.1
1.0.0
```

Meaning:

- `0.x`: early product, behavior may change, but releases should still be usable.
- `0.1.0`: first clean lifecycle release.
- `0.2.0`: project safety and architecture foundation.
- `0.3.0`: provisioning release.
- `0.4.0`: snapshots/reset release.
- `1.0.0`: stable public release with compatibility expectations.

Patch releases:

Use patch versions for:

- bug fixes
- docs fixes
- small compatibility fixes
- security fixes

Do not use patch versions for major behavior changes.

## 4. Release Types

### Development Builds

Used while building.

Not announced publicly.

Can be broken.

### Alpha Releases

Used for early testers.

Expect rough edges.

Good for:

- founder testing
- trusted friends
- internal LabsBackery experiments

Example:

```text
0.1.0-alpha.1
```

### Public v0.x Releases

Public but still early.

Should have:

- working core flow
- docs
- known limitations
- release notes

### Stable v1.0 Release

Public stable contract.

Should have:

- stable CLI commands
- stable config schema
- stable JSON output
- tested lifecycle
- docs
- examples
- known limitations

Do not call Yeast v1.0 too early.

## 5. Release Artifacts

Each serious release should include:

- Git tag
- GitHub release
- Linux binary
- checksum file
- changelog/release notes
- install instructions
- documentation links

Recommended artifacts:

```text
yeast-linux-amd64
yeast-linux-amd64.sha256
```

Later:

```text
yeast-linux-arm64
```

Do not prioritize Windows/macOS binaries before Linux is strong.

## 6. Build Process

Initial build target:

```text
Linux amd64
```

Build requirements:

- Go installed
- clean working tree
- tests passing
- version embedded if possible

Release build should produce:

```text
dist/
  yeast-linux-amd64
  yeast-linux-amd64.sha256
```

Future build automation:

- GitHub Actions
- automatic release artifacts
- checksum generation
- install script compatibility check

## 7. Install Strategy

v0.1 install options:

1. One installation script.
2. Build from source manually.
3. Download binary from GitHub release after artifacts exist.

Required v0.1 install promise:

```text
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash
```

The script should prepare common Linux hosts smartly. It should detect the package manager, install required host dependencies, build Yeast, install the binary, create Yeast directories, generate an SSH key if needed, handle KVM group membership when possible, and run `yeast doctor`.

Supported package managers for v0.1:

- `apt`
- `dnf`
- `yum`
- `pacman`
- `zypper`
- `apk`

Required installer qualities:

- readable output
- clear logs on failure
- non-interactive package installation where possible
- safe overrides for repo URL, ref, install directory, verbosity, and log retention
- honest warning when logout/login is required for KVM group changes

Manual source build remains documented, but the primary user path should be one script.

```text
curl -fsSL https://.../install.sh | bash
```

But the install script must be honest and safe.

## 8. Pre-Release Checklist

Before any public release:

- working tree is clean
- version number chosen
- changelog written
- tests pass
- manual lifecycle checklist passes
- README quickstart tested
- known limitations written
- release notes written
- one-script installer verified or explicitly blocked
- binary builds successfully
- checksum generated
- Git tag created
- GitHub release drafted

For v0.1 specifically:

- `yeast init` works
- `yeast doctor` works
- `yeast pull ubuntu-24.04` works
- `yeast up` works
- `yeast status` works
- `yeast ssh` works
- `yeast down` works
- `yeast destroy` works
- JSON output works for core commands
- destroy does not remove cached images
- project-safe paths work

## 9. Manual Release Test For v0.1

Run this from a clean project directory on a Linux/KVM host.

Checklist:

1. Create new directory.
2. Run `yeast init`.
3. Inspect generated config.
4. Run `yeast doctor`.
5. Run `yeast pull ubuntu-24.04`.
6. Run `yeast up`.
7. Wait for VM ready.
8. Run `yeast status`.
9. Run `yeast ssh`.
10. Confirm shell works.
11. Exit SSH.
12. Run `yeast down`.
13. Run `yeast status`.
14. Run `yeast up` again.
15. Run `yeast destroy`.
16. Confirm runtime files removed.
17. Confirm image cache remains.

If this checklist fails, do not release.

## 10. Release Notes Structure

Every release should have clear notes.

Template:

```text
# Yeast vX.Y.Z

## Summary

One short paragraph describing what this release does.

## What's New

- Feature 1
- Feature 2

## Fixed

- Fix 1
- Fix 2

## Known Limitations

- Limitation 1
- Limitation 2

## Upgrade Notes

- Any migration or compatibility notes

## Install

Commands or link to install docs

## Verification

How to confirm it works
```

## 11. v0.1 Release Notes Draft

Draft summary:

```text
Yeast v0.1.0 is the first clean lifecycle release. It supports initializing a project, pulling trusted Ubuntu cloud images, starting a QEMU/KVM VM, waiting for SSH, showing status, connecting over SSH, stopping the VM, and destroying project runtime files.
```

Known limitations:

- Linux only.
- QEMU/KVM only.
- No provisioning yet beyond generated cloud-init basics.
- No snapshots/reset yet.
- No private multi-VM networking yet.
- No LabsBackery integration contract yet.
- No Yeast MCP integration yet.
- Ubuntu images only at first.

This is good. Early releases should be honest.

## 12. Changelog Policy

Maintain:

```text
CHANGELOG.md
```

Each release section should include:

- Added
- Changed
- Fixed
- Removed
- Security
- Known limitations if needed

Keep it user-facing.

Do not write only commit messages.

## 13. Tagging Policy

Tags should match versions:

```text
v0.1.0
v0.2.0
v1.0.0
```

Before tagging:

- tests pass
- release checklist completed
- release notes ready

Tag command:

```text
git tag v0.1.0
git push origin v0.1.0
```

Do not tag broken releases casually.

## 14. Branching Policy

Simple early policy:

```text
main = latest working development
tags = releases
```

Optional later:

```text
release/v0.1
release/v0.2
```

Do not overcomplicate branching early.

Use small commits and clean tags.

## 15. Documentation Required Before Release

v0.1 required docs:

- README
- install/build from source
- quickstart
- doctor/troubleshooting
- config reference for v0.1
- known limitations

v0.3 required docs:

- provisioning guide
- Caddy example
- provisioning troubleshooting

v0.4 required docs:

- snapshot guide
- reset guide
- disk safety notes

v0.5 required docs:

- networking guide
- static IP guide
- two-VM lab example

v1.0 required docs:

- full config reference
- JSON schema reference
- command reference
- examples
- architecture overview
- LabsBackery integration notes

## 16. Announcement Plan

Do not over-announce before v0.1 is usable.

For v0.1:

Announce softly:

- GitHub README
- personal Twitter/X
- small LinkedIn post
- maybe one blog post

Message:

```text
Yeast v0.1 is out: a Linux-first local VM tool built on QEMU/KVM and cloud-init. It is early, but the first lifecycle is working: init, pull, up, status, ssh, down, destroy.
```

For v0.3 provisioning:

Announce stronger because it becomes more useful.

For v0.5 multi-VM labs:

Announce to cybersecurity/devops communities.

For v1.0:

Full announcement:

- blog post
- demo video
- GitHub release
- Hacker News maybe
- Reddit Linux/devops/cybersecurity communities
- YouTube demo

## 17. Release Risk Checklist

Before release, ask:

- Can a new user install it?
- Can a new user run first success from docs?
- Can a user recover if `up` fails?
- Can `destroy` remove anything outside project runtime?
- Does status tell the truth?
- Are known limitations documented?
- Is JSON output valid?
- Are errors understandable?
- Does this release promise too much?

If any answer is bad, fix before release or document honestly.

## 18. Rollback And Recovery

For v0.x, rollback mostly means:

- users can keep old binary
- old projects may need state migration later
- destructive changes must be avoided

Before introducing state migrations:

- document state format changes
- backup state before migration
- provide clear error if migration unsupported

Do not introduce automatic destructive migrations casually.

## 19. Security Release Policy

If a release fixes a security issue:

- publish patch release quickly
- mention affected versions
- explain mitigation
- avoid over-disclosing exploit details before users can update

Security-sensitive areas:

- path traversal
- destroy cleanup paths
- file provisioning paths
- SSH key handling
- shell command execution
- cloud-init content
- future remote worker mode

## 20. Definition Of A Good Release

A good Yeast release:

- makes a clear promise
- keeps that promise
- has working install path
- has docs
- has release notes
- has known limitations
- passes tests
- passes manual workflow
- does not hide broken behavior

## 21. Release Timeline For v0.1

Expected release path:

1. Finish v2 skeleton.
2. Implement project identity.
3. Implement config/state/images/runtime/cloud-init.
4. Implement core commands.
5. Add JSON output.
6. Add tests.
7. Add docs.
8. Run manual release checklist.
9. Build binary.
10. Tag v0.1.0.
11. Publish GitHub release.
12. Soft announce.

## 22. Post-Release Process

After release:

- collect issues
- update `YEAST_FEEDBACK_LOG.md`
- patch critical bugs
- update docs for confusion points
- decide v0.1.1 or v0.2.0 next
- do not jump immediately to huge new features without feedback

## 23. Next File

After release planning, the next useful file is:

```text
YEAST_DOCS_PLAN.md
```

Then:

```text
YEAST_FEEDBACK_LOG.md
```

Before implementation starts, the minimum planning set is now strong enough:

- vision
- roadmap
- technical discovery
- architecture
- implementation plan
- test plan
- release plan
