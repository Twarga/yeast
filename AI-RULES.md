# AI Rules For Yeast v2

These rules apply to Codex, Claude, and any AI coding agent working on Yeast.

## Core Rules

- Do not start a task unless its dependency is done.
- Do not edit unrelated files.
- Keep each task small.
- Update `TASKS.md` after finishing.
- Run required tests before marking a task done.
- If architecture and code disagree, stop and update docs or ask.
- Never implement future milestone features early.

## Operating Rules

- Read the relevant planning docs before implementation:
  - `YEAST_TECHNICAL_ARCHITECTURE.md`
  - `YEAST_V2_IMPLEMENTATION_PLAN.md`
  - `YEAST_TEST_PLAN.md`
  - current section of `TASKS.md`
- Work on one task at a time.
- Keep changes scoped to the task's listed files/modules.
- Do not refactor unrelated code.
- Do not rename public commands, files, or concepts unless the task explicitly says so.
- Do not silently change architecture decisions.
- If a task reveals a planning problem, update the relevant planning file or ask before continuing.
- After finishing, report:
  - what changed
  - files changed
  - tests run
  - remaining risks

## Rebuild Rule

Yeast v2 starts from a clean implementation.

The current codebase is prototype/reference material. Do not copy old code blindly. Reuse only ideas or small utilities after understanding them.

## Definition Of Done For AI Tasks

A task is done only when:

- implementation matches the task scope
- required tests pass or the reason they cannot run is documented
- `TASKS.md` is updated
- no unrelated files were changed
- the final response lists changed files and verification
