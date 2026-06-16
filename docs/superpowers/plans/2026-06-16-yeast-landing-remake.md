# Yeast Landing Remake Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the Yeast GitHub Pages landing page to match the approved dark neon design, keep the existing Yeast logo and colors, remove the YAML hero panel, and leave the docs site unchanged.

**Architecture:** The landing page stays a single static HTML entrypoint at `landing/index.html` that is copied into the GitHub Pages root by the existing docs workflow. The implementation should preserve the current brand assets and content categories, but simplify the hero, tighten section hierarchy, and remove redundant visual noise so the page feels polished and intentional on desktop and mobile.

**Tech Stack:** Static HTML, CSS, vanilla JavaScript, GitHub Pages, MkDocs Material docs site, GitHub Actions.

## Global Constraints

- Keep the Yeast logo unchanged.
- Keep the black/green Yeast color system unchanged.
- Do not modify the docs site content or structure unless needed to preserve navigation.
- Remove the YAML preview card from the hero; keep only the animated terminal.
- Preserve a single-page landing experience at the GitHub Pages root.
- Maintain Linux/QEMU/KVM positioning and the existing CTA targets.

---

### Task 1: Simplify the hero and terminal area

**Files:**
- Modify: `/home/twarga/Projects/yeast/landing/index.html`

**Interfaces:**
- Consumes: existing hero markup, terminal animation script, Yeast logo asset, current CTA URLs.
- Produces: a cleaner hero with only the animated terminal proof panel and stronger CTA hierarchy.

- [ ] **Step 1: Remove the YAML snippet panel from the hero layout**

Delete the code panel that renders the `yeast.yaml` preview and keep only the animated terminal proof panel on the right side of the hero.

- [ ] **Step 2: Tighten the hero layout**

Update the hero grid, spacing, and typography so the headline, subhead, buttons, and terminal feel balanced without the removed panel.

- [ ] **Step 3: Keep the terminal animation but polish the copy**

Retain the existing command animation structure, but ensure it reads as a short command sequence instead of a crowded console transcript.

- [ ] **Step 4: Verify the hero still works visually on desktop and mobile**

Run a local preview and confirm the hero does not collapse awkwardly at narrow widths.

### Task 2: Refine the remaining homepage sections

**Files:**
- Modify: `/home/twarga/Projects/yeast/landing/index.html`

**Interfaces:**
- Consumes: current feature, workflow, and use-case sections.
- Produces: a shorter, cleaner page with less repetition and better section rhythm.

- [ ] **Step 1: Reduce repeated visual weight**

Tune section spacing, card density, and divider treatment so each section feels distinct rather than stacked with equal weight.

- [ ] **Step 2: Keep the content categories but sharpen presentation**

Preserve install, what-it-is, features, how-it-works, and use-cases blocks, but present them with a cleaner hierarchy and fewer competing accents.

- [ ] **Step 3: Align footer and nav with the simplified landing page**

Keep docs and GitHub links obvious, but remove any extra visual clutter that makes the page feel busy.

- [ ] **Step 4: Preserve docs navigation targets**

Make sure homepage links still point to the docs and tutorials paths used by the current GitHub Pages setup.

### Task 3: Verify, build, and publish

**Files:**
- Inspect: `/home/twarga/Projects/yeast/landing/index.html`
- Inspect: `/home/twarga/Projects/yeast/.github/workflows/deploy-docs.yml`

**Interfaces:**
- Consumes: final landing page HTML and the existing GitHub Pages workflow.
- Produces: a validated static site change ready for push and deployment.

- [ ] **Step 1: Validate the HTML structure locally**

Open the landing page in a browser and confirm the hero, CTA row, terminal, section spacing, and footer render correctly.

- [ ] **Step 2: Run a build or lint check if available**

Confirm the change does not break the docs workflow assumptions and does not require workflow edits.

- [ ] **Step 3: Commit and push the landing page remake**

Create a focused commit with the homepage-only changes and push it so GitHub Pages can deploy the updated landing page.

