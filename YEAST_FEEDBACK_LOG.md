# Yeast Feedback Log

Status: Living document  
Owner: Twarga / TwargaOps  
Phase: 13 - Feedback  
Purpose: Capture user feedback, bugs, confusion, feature requests, and product decisions after releases

## 1. Purpose

This file is Yeast's memory after it meets real users.

Before release, Yeast is mostly theory, architecture, and founder judgment.

After release, users show what is actually confusing, broken, valuable, missing, or unnecessary.

The goal of this file is to prevent feedback from disappearing into memory, DMs, GitHub notifications, or random notes.

The rule:

> If feedback affects product direction, docs, bugs, roadmap, or trust, record it here.

## 2. Feedback Principles

### Principle 1: Feedback Is Data, Not Identity

If a user says something failed, that is not an insult. It is information.

### Principle 2: Patterns Matter More Than One-Off Comments

One person can be wrong or unusual.

Three people hitting the same problem is a product signal.

### Principle 3: Confusion Is Product Feedback

If users ask you to explain something repeatedly, the docs or product flow are not clear enough.

### Principle 4: Not Every Feature Request Should Be Built

Feature requests must be filtered through Yeast's vision and roadmap.

### Principle 5: Feedback Should Change Documents

Good feedback should update:

- docs
- test plan
- roadmap
- implementation plan
- architecture if needed

## 3. Feedback Categories

Use these categories:

- bug
- install problem
- docs confusion
- UX confusion
- config confusion
- runtime failure
- networking issue
- provisioning issue
- snapshot issue
- JSON/API issue
- feature request
- positive signal
- business signal
- security concern
- roadmap input

## 4. Feedback Entry Template

Use this template for each feedback item.

```text
## YYYY-MM-DD - Short title

Source:
- GitHub issue / DM / user interview / Discord / YouTube / local user / self-test

Category:
- bug / docs confusion / feature request / etc.

User context:
- Who gave this feedback?
- What were they trying to do?

Feedback:
- What did they say or what happened?

Evidence:
- Error logs, screenshots, commands, quotes, reproduction steps

Impact:
- Blocker / major / minor / nice-to-have

Pattern:
- First report / repeated / common

Decision:
- fix now / document / defer / reject / investigate

Action:
- linked issue/task/doc update

Status:
- open / planned / fixed / rejected / watching
```

## 5. Current Feedback

No external user feedback yet.

Current state:

Yeast is still in planning/rebuild phase. Feedback will start after a usable v0.1 or alpha is released.

## 6. Founder Self-Feedback

Use this section when you test Yeast yourself and notice friction.

### 2026-05-16 - Planning clarity improved

Source:
- founder planning session

Category:
- product direction

Feedback:
- Yeast needed a clean product/architecture lifecycle before implementation.
- Current MVP code should be treated as prototype/reference, not final foundation.

Impact:
- major

Decision:
- start v2 clean after planning files are complete

Action:
- created product roadmap, technical discovery, architecture, implementation plan, test plan, release plan, docs plan

Status:
- done

## 7. Pattern Tracker

Use this section to track repeated issues.

| Pattern | Count | Last Seen | Decision |
|---|---:|---|---|
| No external patterns yet | 0 | - | - |

## 8. Feature Request Tracker

Use this section to collect requests without immediately committing to them.

| Request | Source | Strategic Fit | Decision |
|---|---|---|---|
| No external feature requests yet | - | - | - |

Strategic fit options:

- high: supports Yeast core roadmap
- medium: useful but not urgent
- low: distraction
- unknown: needs discovery

## 9. Bug Tracker Summary

This is not a replacement for GitHub Issues.

Use this section to summarize product-level bug patterns.

| Bug Pattern | Severity | Related Issues | Status |
|---|---|---|---|
| No external bugs yet | - | - | - |

## 10. Documentation Confusion Tracker

Use this section when users get confused by docs.

| Doc Area | Confusion | Count | Action |
|---|---|---:|---|
| No external doc confusion yet | - | 0 | - |

## 11. Release Feedback Sections

Create one subsection per release.

## v0.1.0 Feedback

Status:

Not released yet.

Expected things to watch:

- install problems
- KVM permission problems
- missing host dependencies
- image pull failures
- cloud-init/SSH timeout
- status lying
- destroy confusion
- JSON output issues
- unclear docs

## v0.2.0 Feedback

Status:

Not released yet.

Expected things to watch:

- project identity confusion
- project folder moves
- state migration issues
- runtime path confusion

## v0.3.0 Feedback

Status:

Not released yet.

Expected things to watch:

- provisioning failures
- shell command confusion
- file copy paths
- package install time
- rerun behavior

## 12. Decision Log From Feedback

Use this when feedback causes a product decision.

Template:

```text
## YYYY-MM-DD - Decision title

Feedback that caused decision:
- ...

Decision:
- ...

Reason:
- ...

Files updated:
- ...

Status:
- ...
```

## 13. Feedback Review Rhythm

After each release:

1. Wait a few days for real usage.
2. Collect GitHub issues, comments, DMs, and self-test notes.
3. Add entries to this file.
4. Group repeated patterns.
5. Decide what becomes:
   - patch release
   - docs update
   - next roadmap item
   - rejected request
6. Update roadmap, test plan, docs plan, or implementation plan.

Minimum review:

```text
30 minutes after every release
```

Better review:

```text
weekly during active release period
```

## 14. Final Rule

Feedback should not live only in your head.

If users teach you something, write it down.

That is how Yeast improves without losing the story.
