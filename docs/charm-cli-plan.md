# Yeast Charm CLI Plan

Status: Draft v1  
Owner: TwargaOps  
Scope: Human terminal experience only

## Goal

Yeast should feel like a serious modern CLI, not a thin wrapper around QEMU.

The terminal experience should be:

- clear when everything works
- calm when something fails
- structured enough to scan quickly
- visually strong enough to match Yeast as a flagship TwargaOps project
- separate from JSON output so tools remain stable

## Rule

Human output can be beautiful. JSON output must stay boring.

This means Charm libraries belong in the human terminal layer, progress layer, prompts, and docs rendering. They must not leak ANSI styling into `--json`.

## Charm Libraries To Use

### Lip Gloss

Use now.

Purpose:

- styled command summaries
- status badges
- bordered result panels
- aligned tables
- consistent Yeast colors

Where:

- `internal/output`

Current implementation:

- human command output renders through Lip Gloss
- JSON output remains separate

### Glamour

Use after v0.1 docs exist.

Purpose:

- render markdown help pages beautifully inside the terminal
- power future commands like `yeast help quickstart`, `yeast help config`, and `yeast help troubleshooting`

Where:

- `internal/output`
- future `internal/docs`

Good first use:

- render `docs/quickstart.md` from the CLI

### Huh

Use after the plain `yeast init` workflow is stable.

Purpose:

- interactive project creation wizard
- guided config creation
- safer beginner experience

Where:

- `cmd/yeast`
- `internal/app/init.go`

Good first command:

- `yeast init --interactive`

Important:

- keep non-interactive `yeast init` available for scripts
- never make prompts mandatory in automation

### Bubble Tea

Use after lifecycle events are modeled.

Purpose:

- live `yeast up` progress screen
- live `yeast pull` download progress
- future lab dashboard for multiple VMs

Where:

- future `internal/ui`
- future event stream from app workflows

Good first use:

- `yeast up` progress view showing image, disk, seed ISO, QEMU start, SSH readiness

Important:

- do not wire Bubble Tea directly into app workflows
- app workflows should emit events; UI should render events

### Bubbles

Use with Bubble Tea.

Purpose:

- progress bars
- spinners
- tables
- viewport/log panes

Good first components:

- spinner for lifecycle steps
- progress bar for image download
- table for multi-VM status later

### Charm Log

Use when Yeast has more runtime diagnostics.

Purpose:

- beautiful human logs
- clearer debug output
- future `--verbose` and `YEAST_LOG_LEVEL`

Where:

- internal logging package, not random direct calls across runtime code

Important:

- runtime and app packages should not print directly
- logs must not replace structured results

### Harmonica

Optional, later.

Purpose:

- smooth animation inside Bubble Tea views

Decision:

- do not use in v0.1
- only useful if Yeast gets a real interactive TUI

### Wish

Not planned for local Yeast v0.1.

Purpose:

- building SSH apps

Why not now:

- Yeast is currently a local CLI engine
- Wish may become relevant for hosted Twarga Cloud lab terminals later, but not for local VM lifecycle

## Implementation Phases

### Phase C1: Pretty Human Output

Status: started.

Scope:

- add Lip Gloss
- style current command results
- preserve JSON contracts

Done when:

- `doctor`, `init`, `pull`, `up`, `status`, `down`, and `destroy` have styled human output
- JSON tests still pass

### Phase C2: Terminal Docs

Scope:

- add Glamour
- create terminal-rendered docs command

Potential commands:

- `yeast docs quickstart`
- `yeast docs config`
- `yeast docs troubleshoot`

Done when:

- docs render nicely in terminal
- markdown files remain normal GitHub-readable docs

### Phase C3: Interactive Init

Scope:

- add Huh
- create `yeast init --interactive`

Questions:

- instance name
- image
- memory
- CPU count
- disk size
- user
- sudo mode

Done when:

- beginner can create a config without editing YAML
- scripts can still use plain `yeast init`

### Phase C4: Lifecycle Event Model

Scope:

- app workflows emit lifecycle events
- output renderers consume events

Events:

- config loaded
- state locked
- image ready
- disk prepared
- seed ISO created
- QEMU started
- SSH ready
- state saved

Done when:

- human output, JSON output, and future TUI output all use the same event model

### Phase C5: Live Progress TUI

Scope:

- add Bubble Tea and Bubbles
- render live progress for `pull` and `up`

Done when:

- long operations feel alive
- failures show the exact failed step
- non-TTY or `--json` mode never starts the TUI

## Non-Goals

- Do not make Yeast depend on a full TUI to work.
- Do not make interactive prompts mandatory.
- Do not style JSON output.
- Do not hide real errors behind pretty boxes.
- Do not add Charm libraries where the standard library is enough.

## Product Standard

The user should understand three things immediately:

- what Yeast is doing
- what succeeded or failed
- what to do next

Pretty output is only useful if it improves those three things.
