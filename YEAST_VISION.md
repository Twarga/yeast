# Yeast Vision

Status: Draft v1  
Owner: Twarga / TwargaOps  
Phase: 1 - Vision  
Purpose: Define why Yeast exists, what it should become, and what it must never become

## 1. One-Liner

Yeast is a Linux-first local VM orchestrator that turns a project folder into real QEMU/KVM machines, making local infrastructure repeatable, automatable, and understandable.

## 2. Short Vision

Yeast should make real Linux VMs feel as easy to use as a project command.

A user should be able to enter a folder, describe the machines they need, run one command, and get a working environment without manually dealing with image downloads, qcow2 disks, cloud-init files, QEMU flags, SSH ports, or broken state.

The long-term goal is not only to replace parts of the Vagrant workflow. The long-term goal is to make Yeast the infrastructure engine for TwargaOps.

## 3. Founder Thesis

Local infrastructure tooling is still too painful for people who want real machines.

Containers are useful, but they do not replace real VMs. A lot of learning, security practice, OS testing, networking, system service work, and realistic infrastructure experimentation needs actual Linux machines.

The problem is that real VM workflows are usually too heavy:

- old tooling
- too many providers
- painful VirtualBox setup
- manual QEMU commands
- unclear networking
- difficult reset workflows
- weak automation interfaces
- not designed for AI agents
- not designed for cybersecurity labs

Yeast exists because TwargaOps needs a clean, owned foundation for this world.

LabsBackery needs real multi-VM labs.

Yeast MCP needs safe machine-readable control of real VMs.

Twarga Cloud eventually needs workers that can run hosted lab environments.

All of those products become weaker if the VM engine is external, messy, or not understood.

So Yeast is not just a side tool. It is the base layer.

## 4. The Product Belief

The core belief behind Yeast is:

> Real infrastructure should be understandable before it becomes scalable.

Many tools hide complexity by creating more abstraction. Yeast should hide repetitive pain, but not erase the mental model.

The user should still understand:

- there is a project
- the project has a config
- the config describes desired machines
- Yeast creates actual machines
- state records what exists
- QEMU/KVM runs the machines
- cloud-init prepares the guest
- SSH/control lets Yeast operate inside the VM

This is important. Yeast should feel simple, but not magical in a confusing way.

## 5. What Yeast Should Become

Yeast should become the local infrastructure engine that supports:

- solo Linux developers who want real VM environments
- DevOps learners who want repeatable practice environments
- cybersecurity students who need labs they can break and reset
- course creators who want packaged lab templates
- LabsBackery as a visual lab platform
- Yeast MCP as an AI-controlled VM workflow layer
- Twarga Cloud as a hosted version later

The product should grow in this direction:

```text
simple local VM tool
  -> reliable project engine
  -> provisioning system
  -> snapshot/reset engine
  -> multi-VM lab networking
  -> guest control layer
  -> LabsBackery engine
  -> MCP automation layer
  -> cloud worker foundation
```

## 6. What Yeast Should Feel Like

Yeast should feel:

- fast
- direct
- understandable
- terminal-native
- automation-friendly
- boring in the good way
- reliable enough to build other products on

It should not feel like:

- a giant cloud platform
- a fragile wrapper script
- a black box
- a YAML monster
- a Vagrant clone with a different name
- an enterprise hypervisor dashboard

## 7. The First User

The first real user is the founder.

That is acceptable because the founder has real pain:

- Linux infrastructure work
- VM lab needs
- LabsBackery requirements
- desire to replace Vagrant
- need for AI-controlled machines later

But Yeast cannot stay built only for the founder.

The second user should be another Linux builder who wants a simple VM workflow.

The third user should be a lab creator who wants multiple machines, provisioning, and reset.

## 8. The First Successful Workflow

The first workflow Yeast should make excellent is:

```text
create project
define Ubuntu VM
pull trusted image
start VM
wait for SSH
connect to VM
stop VM
destroy VM
```

The second successful workflow should be:

```text
define Ubuntu VM
install Caddy automatically
copy a small web app
start service
verify it works
snapshot clean state
restore clean state
```

The third successful workflow should be:

```text
define attacker VM
define target VM
connect them on private lab network
provision target
snapshot lab baseline
reset lab after breakage
```

If Yeast can do these three workflows well, the product direction is real.

## 9. Why Yeast Matters To TwargaOps

TwargaOps is not only a collection of projects. It needs a foundation.

Yeast can become that foundation because it connects the ecosystem:

- Yeast gives TwargaOps technical credibility.
- LabsBackery uses Yeast to create cybersecurity labs.
- Twarga Academy can teach with labs powered by Yeast.
- Yeast MCP lets AI agents inspect and operate lab machines.
- Twarga Cloud can later host managed Yeast-powered labs.

The open-source product builds trust.

The ecosystem creates revenue:

- teaching
- mentoring
- courses
- support
- consulting
- hosted labs
- managed infrastructure

This makes Yeast strategically important even if Yeast itself is free.

## 10. Product Values

### Own What You Run

Yeast should help people run infrastructure they understand and control.

### Prefer Real Machines When Real Machines Matter

Containers are not bad. But some workflows need real Linux VMs. Yeast should serve those workflows.

### Hide Pain, Not Reality

Yeast should remove repetitive setup pain, but keep the system explainable.

### Be Boring At The Core

The core VM engine should be predictable. LabsBackery and Yeast MCP can be exciting. Yeast itself should be reliable.

### Build For Humans And Tools

Humans need clear terminal output. Tools need stable JSON and events.

### Start Local Before Cloud

The local engine must work before hosted Twarga Cloud exists.

## 11. What Yeast Must Never Become

Yeast must never become a product that hides so much that the user cannot understand what is happening.

Yeast must never become a full cloud platform before the local engine is excellent.

Yeast must never become a generic wrapper around every hypervisor at the cost of being great on Linux/QEMU/KVM.

Yeast must never become dependent on LabsBackery or Twarga Cloud to be useful.

Yeast must never become a collection of random commands without a clear state model.

Yeast must never make JSON automation an afterthought.

Yeast must never chase enterprise features before it serves individual builders and lab creators well.

## 12. Success In 1 Year

In one year, Yeast should:

- install cleanly on Linux
- start Ubuntu VMs reliably
- support project-safe state
- support basic provisioning
- support snapshots/reset
- support at least one private multi-VM lab
- have clear docs
- have example projects
- power an early LabsBackery prototype
- be explainable in one technical article or video

Success in one year is not cloud scale.

Success in one year is trust.

## 13. Success In 3 Years

In three years, Yeast should:

- be a stable local infrastructure engine
- power LabsBackery labs
- support strong template workflows
- expose mature JSON/events
- support Yeast MCP workflows
- have real users outside the founder
- support Twarga Academy courses
- have a growing library of lab templates
- be reliable enough that people recommend it

Success in three years is adoption and ecosystem use.

## 14. Success In 5 Years

In five years, Yeast could:

- power hosted labs through Twarga Cloud
- run remote workers safely
- support teams/classes
- become a known open-source infrastructure tool
- support a paid ecosystem around courses, hosting, support, and templates
- become the technical proof behind TwargaOps as a serious infrastructure company

Success in five years is not only users.

Success in five years is leverage.

## 15. The Vision Sentence To Remember

Yeast exists to make real Linux VM environments project-native, repeatable, automatable, and understandable, so TwargaOps can build labs, AI workflows, education, and hosted infrastructure on a foundation it owns.
