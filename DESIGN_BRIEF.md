# Yeast Website Design Brief

## 1. What is Yeast?

Yeast is a **Linux-first local VM orchestration tool** for QEMU/KVM. It turns a project folder into real virtual machines.

Think of it as "Vagrant done right for Linux" — but the vision is bigger. Yeast is meant to become the infrastructure foundation for TwargaOps, powering cybersecurity labs (LabsBackery), AI-controlled VM workflows (Yeast MCP), and eventually hosted cloud environments.

**The one-liner:** *"Turn a folder into real VMs."*

**The full pitch:** Define machines in a single `yeast.yaml` file, run `yeast up`, and get working Linux VMs with cloud-init provisioning, private networking, SSH access, snapshots, and a stable JSON/events contract for automation.

## 2. Target Audience

| Persona | Who They Are | What They Want |
|---|---|---|
| **Linux Developer** | Writes code, needs local VMs for testing | Fast, repeatable dev environments |
| **DevOps Learner** | Learning infrastructure, needs practice labs | Clean lab environments they can break and reset |
| **Security Student** | Practicing cybersecurity, needs multi-VM labs | Attacker/target labs with networking |
| **Course Creator** | Building courses with hands-on labs | Packaged, distributable lab templates |
| **Automation Engineer** | Building CI/CD or tooling | Stable JSON output and event streams |

## 3. Brand Personality

From our vision document, Yeast should feel:

- **Fast** — VMs boot in seconds, not minutes
- **Direct** — one YAML file, one command
- **Understandable** — not magical in a confusing way
- **Terminal-native** — CLI-first, not GUI-first
- **Automation-friendly** — JSON and events for tools
- **Boring in the good way** — reliable, predictable core
- **Trustworthy** — reliable enough to build other products on

**It should NOT feel like:**
- A giant cloud platform
- A fragile wrapper script
- A black box
- An enterprise dashboard

## 4. The Logo

**Current Logo:** A stylized letter "Y" inside a rounded square border.

**Description:**
- Outer shape: A rounded square (rect with rx/ry ~40px on 200x200 canvas) with a stroke border
- Inner shape: The letter "Y" formed by three rounded rectangles:
  - Left diagonal stroke: angled left
  - Right diagonal stroke: angled right  
  - Vertical stem: straight down connecting the two diagonals
- The strokes have rounded caps (rx=7 on the rects)
- There's a subtle glow effect behind the border

**Logo file:** `/home/twarga/Projects/yeast/assets/logo.svg` — SVG format, can scale to any size.

**Note:** The current logo is simple/minimal. Feel free to suggest refinements or a complete redesign if you think it fits the brand better. Just keep the "Y" letter concept or the name association.

## 5. Brand Color

**Primary brand color:** Green. The current hex is `#22c55e` (a bright, energetic green).

**Note:** This is your domain — feel free to adjust the exact shade, saturation, and complementary palette. The green should feel:
- Technical but approachable
- Not "enterprise corporate" green
- More "developer tool" than "eco-friendly"
- Could lean toward neon/cyber if that fits your vision, or toward forest/organic if that works better

## 6. What the Landing Page Must Communicate

### Above the fold (Hero):
- **What Yeast is** in one sentence: "Turn a folder into real VMs."
- **The sub-headline:** Something about Linux-first VM orchestration with YAML
- **Primary CTA:** Get Started / Install / Quick Start
- **Secondary CTA:** View on GitHub / Read Docs
- **Install command:** `curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash`
- **Quick start snippet** (5 commands: mkdir, init, pull, up, ssh)

### Key value propositions (Features section):
1. **YAML-Driven** — One `yeast.yaml` defines your entire lab. No GUI, no clicks.
2. **Real VMs** — Full hardware virtualization with QEMU/KVM. Not containers.
3. **Snapshots** — Stop, snapshot, restore. Safe lab reset in seconds.
4. **Private Networking** — Multi-VM lab networks out of the box.
5. **Cloud-Init Provisioning** — Automatic first-boot setup.
6. **JSON Automation** — Stable output and event streams for scripting.

### Social proof / trust signals:
- GitHub stars (if available)
- "Built by TwargaOps"
- MIT License
- Links to documentation

### Sections to include:
1. **Hero** — The "what" and "get started" CTA
2. **Features** — 6 key selling points (could be 2x3 grid or similar)
3. **How it works** — A visual workflow showing: YAML → `yeast up` → Running VMs → SSH in
4. **Install** — One-liner install command + package manager options
5. **Quick start** — Terminal-style code block with the 5-command workflow
6. **Who uses it** / Use cases — Developers, DevOps learners, Security students
7. **Footer** — GitHub link, license, TwargaOps branding

## 7. Design Direction (Your Domain)

This is entirely your call. Some possible directions:

- **Dark terminal aesthetic** — Green-on-black, monospace fonts, terminal windows as UI elements
- **Clean modern SaaS** — White/light with green accents, professional but approachable
- **Cyberpunk/tool aesthetic** — Dark mode with neon green accents, grid lines, technical feel
- **Minimal Swiss design** — Clean typography, lots of whitespace, grid layouts

The docs site uses VitePress with a default dark/light toggle. The landing page can:
- Be completely custom (separate from VitePress theme)
- Use VitePress's home page layout (the `layout: home` frontmatter)
- Or be a hybrid — custom CSS on top of VitePress

## 8. Assets Available

- Logo SVG: `/assets/logo.svg`
- Current website: https://twarga.github.io/yeast/
- GitHub: https://github.com/Twarga/yeast
- Brand color reference: `#22c55e`

## 9. Must-Have Elements

- [ ] Logo visible (can be redesigned)
- [ ] Product name "Yeast" prominently displayed
- [ ] Tagline "Turn a folder into real VMs."
- [ ] Install command visible
- [ ] "Get Started" or "Read Docs" CTA button
- [ ] GitHub link
- [ ] Feature highlights (6 points)
- [ ] Terminal/code block for quick start
- [ ] Footer with license info (MIT, TwargaOps)

## 10. Nice-to-Have Elements

- [ ] Animated terminal demo (typing commands, showing output)
- [ ] Dark/light mode toggle
- [ ] Smooth scroll animations
- [ ] Interactive feature cards
- [ ] Diagram showing the architecture (YAML → VM)
- [ ] Testimonials or use cases
- [ ] Performance stats ("Boot in 30 seconds", etc.)

---

**Questions for you, the designer:**

1. What's your vision for the overall aesthetic?
2. Do you want to keep the current logo concept or redesign it?
3. What's your proposed color palette (including the green)?
4. Should the landing page be fully custom HTML/CSS or work within VitePress?
5. Any interactive elements or animations you want to include?
6. What's the layout structure you're thinking?

Please provide:
- A design description / concept
- Any CSS/styling you'd want me to implement
- Layout wireframe (text description is fine)
- Color palette with hex codes
- Typography choices

I'll implement everything in the VitePress docs-site.
