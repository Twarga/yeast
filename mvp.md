Understood.
Below is a **PURE PLAN / SPEC DOCUMENT** — **NO CODE**, **NO SNIPPETS**, **NO IMPLEMENTATION DETAILS**.
This is written so **another AI with zero context** can fully understand **what Yeast is**, **why it exists**, **why it’s better than Vagrant**, **what to build**, **what NOT to build**, and **how the final product should feel**.

You can paste this **as-is** to an AI and say:

> “Build this project exactly as described.”

---

# 🧪 YEAST — FULL PRODUCT & BUILD PLAN

*(Modern Vagrant Replacement — KVM-native, Go, cloud-init, UX-first)*

---

## 1. What Yeast Is (Clear Definition)

**Yeast** is a **local virtual machine orchestration tool**.

It allows users to define and run **local VM environments** from a **simple declarative YAML file**, using:

* Direct **KVM / QEMU**
* **Official OS cloud images**
* **cloud-init** for first-boot configuration
* A **clean, calm, professional CLI**

Yeast is **not**:

* a cloud platform
* a virtualization manager
* a GUI
* a provisioning framework

Yeast is a **single-purpose CLI tool** whose only job is:

> *Turn a simple config file into running virtual machines, quickly and predictably.*

---

## 2. Why Yeast Exists (Problem Statement)

### The problem with Vagrant

Vagrant was built for a different era and suffers from structural problems:

* Uses a **Ruby DSL**, which increases cognitive load
* Depends heavily on **VirtualBox**, which is slow and unstable
* Uses **SSH-based provisioning**, which is slow and fragile
* Relies on a **plugin ecosystem**, which often breaks
* Uses **boxes**, which are bloated, outdated, and inconsistent
* Hides complexity behind abstractions that frequently leak

Vagrant optimizes for **provider abstraction**, not **developer experience**.

---

## 3. Yeast’s Core Philosophy

Yeast is built on the following non-negotiable principles:

1. **Direct > Abstract**
2. **Declarative > Imperative**
3. **Boot-time configuration > SSH provisioning**
4. **Few base images > many boxes**
5. **Clarity > features**
6. **Calm UX > noisy output**

Yeast intentionally supports **fewer use cases** than Vagrant to deliver a **better experience** for the majority of users.

---

## 4. How Yeast Is Better Than Vagrant

| Area          | Vagrant            | Yeast                       |
| ------------- | ------------------ | --------------------------- |
| Language      | Ruby DSL           | Simple YAML                 |
| Hypervisor    | VirtualBox default | Direct KVM/QEMU             |
| Provisioning  | SSH-based          | cloud-init (boot-time)      |
| Images        | Many boxes         | Few base images + templates |
| Extensibility | Plugins            | None (by design)            |
| UX            | Noisy, complex     | Calm, minimal               |
| Speed         | Slow boot          | Fast boot                   |
| Failure mode  | Cryptic errors     | Helpful errors              |

Yeast feels closer to **Docker Compose**, but for **virtual machines**.

---

## 5. Mental Model (CRITICAL)

Yeast is built around this single idea:

```
Base OS Image
      +
Configuration Template
      =
Configured Virtual Machine
```

* **Base images** are clean operating systems only
* **Templates** define what the machine becomes
* Yeast does not bake or build images in MVP

This replaces Vagrant’s “box” concept with something simpler and more flexible.

---

## 6. Target User Experience

A first-time user should be able to:

1. Initialize a project
2. Start machines
3. SSH into a machine

…without reading documentation.

The tool should feel:

* predictable
* calm
* professional
* trustworthy

If the user feels confused, Yeast has failed.

---

## 7. Commands (MVP Only)

Yeast supports **exactly** the following commands:

* initialize a project
* start machines
* show status
* connect via SSH
* stop machines

There are **no optional features**, **no sub-modes**, and **no plugins**.

---

## 8. Configuration Model

Each project has a single configuration file.

The configuration:

* is declarative
* describes machines
* never contains logic
* never executes code

Each machine defines:

* which base OS it uses
* which configuration template it applies
* optional CPU and memory limits

Defaults must exist so the user can omit most fields.

---

## 9. Base Images (No “Boxes”)

Yeast **does not use boxes**.

Instead:

* It uses **official upstream cloud images**
* These images are already optimized for virtualization
* They already support cloud-init

Only a **very small number of base images** are supported.

Yeast never modifies base images.

If a required image is missing, Yeast:

* stops
* explains what is missing
* explains where to download it

---

## 10. Templates (Key Innovation)

Templates define **what a machine becomes**.

Templates are:

* simple
* reusable
* OS-level configuration
* executed once at first boot

Templates are written using **cloud-init**, which:

* runs automatically on first boot
* installs packages
* enables services
* creates users
* configures the system before login

Templates are **not scripts run over SSH**.

---

## 11. Why cloud-init (Important Explanation)

cloud-init is an industry-standard system used by:

* cloud providers
* virtualization platforms
* Proxmox
* Kubernetes node bootstrapping

cloud-init allows machines to configure themselves **during boot**, instead of waiting for external tools.

This makes Yeast:

* faster
* more reliable
* more deterministic

cloud-init is **not cloud-specific** — it works perfectly with local VMs.

---

## 12. How Yeast Builds Machines (Conceptual Flow)

For each machine:

1. A project-local disk is created from a base image
2. A cloud-init configuration is generated
3. The VM is started with KVM/QEMU
4. cloud-init configures the machine automatically
5. Yeast records machine state and exits

Yeast never stays running as a daemon.

---

## 13. State Management Philosophy

All runtime state is:

* project-local
* human-readable
* disposable

If state is deleted:

* machines stop
* nothing global is corrupted

Yeast never stores global runtime state.

---

## 14. CLI UX Rules (VERY IMPORTANT)

The CLI must follow these rules:

* Speak like a calm senior engineer
* Show intent, not implementation
* Hide noise
* Never panic
* Never dump raw logs by default

### Visual language is minimal:

* Action indicator
* Success indicator
* Error indicator
* Step indicator

No ASCII art.
No emojis.
No flashy animations.

---

## 15. Logging & Debugging Philosophy

* Default mode: **quiet and clean**
* Debug mode: **explicit and verbose**

Debug output is:

* opt-in
* clearly labeled
* never mixed with normal output

---

## 16. Error Handling Philosophy

Errors must:

1. Explain **what failed**
2. Explain **why**
3. Explain **how to fix it**

Errors must never:

* show stack traces
* expose internal implementation details
* scare beginners

---

## 17. Technology Constraints (Locked)

The MVP is constrained to:

* Linux hosts only
* KVM / QEMU only
* Go language only
* cloud-init only
* YAML configuration only

These constraints are **intentional** and **non-negotiable**.

---

## 18. What Yeast MVP Does NOT Include

Yeast MVP intentionally excludes:

* Windows or macOS support
* Cloud providers
* libvirt
* Ansible
* Docker integration
* Snapshots
* Image building
* Registries
* Advanced networking
* GUIs

This keeps the product focused and reliable.

---

## 19. Success Criteria

Yeast MVP is successful if:

* A user can run a VM in under 2 minutes
* The CLI feels calm and predictable
* The configuration feels obvious
* The user never thinks about providers or plugins
* The user says:
  **“This feels simpler than Vagrant.”**

---

## 20. Final Guiding Principle (MOST IMPORTANT)

> **If a feature improves power but reduces clarity, it must be removed.**

Yeast wins by:

* being small
* being honest
* being boring in the best way

---

## 21. One-Sentence Definition (Use Everywhere)

> **Yeast is a modern, KVM-native local VM orchestrator that replaces Vagrant by using cloud-init, official images, and a clean declarative workflow.**

---

**END OF PLAN**

If you want, next I can:

* compress this into a **single perfect AI prompt**
* rewrite it in **ultra-strict specification language**
* or help you evaluate the AI’s output against this plan

Just say it.