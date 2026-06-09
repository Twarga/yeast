---
layout: home

hero:
  name: "Yeast"
  text: "Turn a folder into real VMs."
  tagline: Linux-first local VM orchestration for QEMU/KVM. One YAML file. One command. Real machines.
  image:
    src: /logo.svg
    alt: Yeast Logo
  actions:
    - theme: brand
      text: Get Started
      link: /docs/quickstart
    - theme: alt
      text: GitHub
      link: https://github.com/Twarga/yeast

features:
  - title: YAML-Driven
    details: One yeast.yaml defines your entire lab. No GUI, no clicks. Version control your infrastructure.
  - title: Snapshots
    details: Stop VMs, take snapshots, restore to any state. Safe lab reset in seconds.
  - title: Private Networking
    details: Private lab networks for multi-VM environments. VM-to-VM communication out of the box.
  - title: Cloud-Init Provisioning
    details: VMs are provisioned with cloud-init on first boot. No manual configuration required.
  - title: JSON Automation
    details: Stable JSON output and event streams for scripting and integration with other tools.
  - title: Real VMs
    details: Full hardware virtualization with QEMU/KVM. Not containers. Real Linux machines.
---

<div style="text-align: center; padding: 2rem 0;">
  <h2>Install Yeast</h2>
  <div style="background: #1a1a1a; border-radius: 8px; padding: 1rem; display: inline-block; font-family: monospace;">
    <code>curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/main/install.sh | bash</code>
  </div>
</div>

<div style="text-align: center; padding: 2rem 0;">
  <h2>Quick Start</h2>
  <div style="background: #1a1a1a; border-radius: 8px; padding: 1rem; display: inline-block; font-family: monospace; text-align: left;">
    <div><span style="color: #22c55e;">$</span> mkdir my-lab && cd my-lab</div>
    <div><span style="color: #22c55e;">$</span> yeast init</div>
    <div><span style="color: #22c55e;">$</span> yeast pull ubuntu-24.04</div>
    <div><span style="color: #22c55e;">$</span> yeast up</div>
    <div><span style="color: #22c55e;">$</span> yeast ssh</div>
  </div>
</div>
