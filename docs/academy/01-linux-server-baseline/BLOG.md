# Lab 01: Linux Server Baseline — The Long Version

## Before You Start: Why This Lab Exists

Most DevOps courses start with Docker, or Kubernetes, or some shiny CI/CD pipeline. They skip the thing that makes all of that work: a properly configured Linux server underneath.

You are going to spend a lot of your career SSHing into Linux machines. When something breaks at 2 AM — and it will — your ability to orient yourself quickly on an unknown system is what separates a ten-minute fix from a two-hour outage. That skill does not come from reading about it. It comes from building servers the right way, over and over, until the checks are automatic.

This first lab is deliberately unglamorous. You are going to build a single server, configure it correctly, verify it carefully, break it on purpose, and fix it. No application, no web traffic, no database. Just the operating system and the operational layer on top of it.

Do it carefully. Every lab that follows builds on this foundation.

---

## What "Operational" Actually Means

When an engineer says a server is "operational," they do not mean it booted. They mean:

- The hostname identifies what the machine is and where it lives
- The timezone is consistent with the rest of the fleet (almost always UTC)
- Access is controlled — only the right ports are open, only the right people can log in
- The system patches itself for security vulnerabilities
- Suspicious login attempts are detected and blocked
- You can verify all of the above with commands, not assumptions

A server that booted but has none of these properties is not operational. It is a liability.

---

## The Scenario

You joined a small engineering team. They have a new server — freshly provisioned Ubuntu 22.04. Before anyone deploys software onto it, your job is to baseline it. The team has a checklist. Your job is to work through it, verify each step, and hand it off ready.

This is real work. Teams actually do this. Some use Ansible (we will get to that in Lab 07). For now, you will do it manually so you understand what the automation is actually doing when you get there.

---

## The Yeast Config

Open `yeast.yaml` and read it before running anything.

```yaml
version: 1

instances:
  - name: baseline
    hostname: baseline
    image: ubuntu-22.04
    memory: 1024
    cpus: 2
    disk_size: 20G
    ssh_port: 2201
    user: ubuntu
    sudo: nopasswd

    provision:
      packages:
        - curl
        - wget
        - vim
        - htop
        - ufw
        - unattended-upgrades
        - fail2ban
      shell:
        - hostnamectl set-hostname baseline
        - timedatectl set-timezone UTC
        - ufw allow OpenSSH
        - ufw --force enable
        - systemctl enable --now fail2ban
        - systemctl enable --now unattended-upgrades
```

This file is the desired state of your environment. It describes a single VM called `baseline`, running Ubuntu 22.04, with 1 GB RAM and 2 CPUs. Yeast will boot it, install the listed packages, and run the shell commands in order.

**`ssh_port: 2201`** — Inside the VM, SSH listens on port 22. Yeast exposes that SSH management connection on port 2201 on your laptop. You never connect to port 22 directly. `yeast ssh baseline` handles the connection for you.

**`sudo: nopasswd`** — The `ubuntu` user can run `sudo` without entering a password. This is necessary for Yeast's provisioning to work without interactive prompts. In a real production environment you would lock this down further, but for a local lab it is fine.

**`provision.packages`** — These get installed via `apt` during first boot. Notice `ufw`, `unattended-upgrades`, and `fail2ban` — these are the security baseline.

**`provision.shell`** — These commands run after packages are installed, in order. The `ufw --force enable` flag prevents UFW from prompting "this may disrupt existing connections, continue?" — which would hang in a non-interactive SSH session.

---

## Booting The Lab

```bash
yeast up
```

What happens under the hood:

1. Yeast checks if the Ubuntu 22.04 image is cached. If not, it downloads it.
2. It creates a QEMU/KVM virtual machine using that image.
3. Cloud-init runs on first boot — it sets up the `ubuntu` user and installs your SSH public key.
4. Yeast SSHes into the VM and runs the provision block.
5. It reports the VM as ready.

The first boot takes 60–90 seconds. After that the VM state persists on disk until you `yeast destroy` it.

---

## Verifying Hostname And Timezone

These sound trivial. They are not.

A wrong hostname means your logs say `ubuntu` or `localhost` instead of `baseline`. When you are correlating logs from ten servers, unidentifiable hostnames make incident diagnosis much harder.

```bash
hostname
```

Expected: `baseline`

A wrong timezone means your log timestamps are in a local timezone that may differ between servers, between engineers, and between the monitoring system. UTC eliminates all of that ambiguity. Every server in your fleet should log in UTC.

```bash
timedatectl
```

Look for: `Time zone: UTC (UTC, +0000)`

If it says anything else, fix it:

```bash
sudo timedatectl set-timezone UTC
```

---

## Understanding UFW

UFW stands for Uncomplicated Firewall. It is a front-end for `iptables` — Linux's kernel-level packet filtering.

The mental model is simple: by default, UFW denies everything incoming and allows everything outgoing. You then explicitly allow what you need.

```bash
sudo ufw status verbose
```

You should see:

```
Status: active

To                         Action      From
--                         ------      ----
OpenSSH                    ALLOW IN    Anywhere
OpenSSH (v6)               ALLOW IN    Anywhere (v6)
```

Only SSH is allowed. Nothing else can reach this VM from the network.

In later labs you will add rules for HTTP (port 80) and HTTPS (port 443) when you deploy web servers. But the default posture is: deny everything, allow only what is explicitly needed.

---

## Understanding Fail2Ban

Fail2ban watches log files for patterns that indicate attacks. For SSH, it watches `/var/log/auth.log` for repeated failed login attempts. When a source IP fails too many times in a short window, fail2ban adds a temporary `iptables` rule to block that IP.

This does not prevent a determined attacker. But it does eliminate the constant background noise of automated bots scanning for weak SSH credentials — which every internet-exposed server deals with.

Check the SSH jail:

```bash
sudo fail2ban-client status sshd
```

Output:

```
Status for the jail: sshd
|- Filter
|  |- Currently failed: 0
|  |- Total failed:     0
|  `- File list:       /var/log/auth.log
`- Actions
   |- Currently banned: 0
   |- Total banned:     0
   `- Banned IP list:
```

No bans yet — the VM just started and nobody has tried to break in. The important thing is the jail is active and watching the right log file.

---

## Understanding Unattended-Upgrades

Security vulnerabilities in packages get patched regularly. If you manually run `apt upgrade` once and never again, your server accumulates unpatched vulnerabilities over time.

`unattended-upgrades` runs automatically and installs security updates without manual intervention. It only applies security patches by default — not every package update. That keeps the risk of an upgrade breaking something low while keeping the security posture current.

Verify it is configured:

```bash
cat /etc/apt/apt.conf.d/20auto-upgrades
```

Expected:

```
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
```

The `"1"` means "do this every 1 day."

---

## The Firewall Failure Drill

This is the most important exercise in the lab. Do it.

The scenario: you are hardening a server and you decide to add a stricter firewall rule. You apply it without testing. It blocks SSH. You lose access.

This happens. It has happened to every engineer at least once. Understanding what it feels like — and how to prevent it — before it happens to you on a real server is worth more than any tutorial.

**Step 1:** Open a second terminal. Keep your current SSH session open.

**Step 2:** In your current session, deny SSH:

```bash
sudo ufw deny OpenSSH
sudo ufw reload
```

**Step 3:** From your second terminal, try to SSH in:

```bash
yeast ssh baseline
```

It will hang or refuse. You are locked out.

**Step 4:** Go back to your first session — the one still open. Fix it:

```bash
sudo ufw delete deny OpenSSH
sudo ufw allow OpenSSH
sudo ufw reload
```

**Step 5:** Try the second terminal again. It works.

**The rule:** Always have a second session open when making firewall changes. Always test from a second session before closing the first. On a cloud VM where you cannot get console access, getting this wrong means rebuilding the machine.

---

## Snapshots: Your Safety Net

Yeast supports VM snapshots. A snapshot captures the entire disk state of the VM at a point in time. You can restore to it instantly.

Take a snapshot of the clean baseline state now:

```bash
yeast snapshot baseline clean-baseline
```

Now if you break something in a future experiment, you can get back to this exact state:

```bash
yeast restore baseline clean-baseline
```

Get into the habit of snapshotting before any risky change. In later labs — especially the chaos engineering lab — you will rely on this heavily.

---

## Real-World Reflection

The pattern you followed in this lab is the same pattern used by every team that runs servers seriously:

1. Start from a known base image
2. Apply configuration as code (here it is Yeast provisioning — later it will be Ansible)
3. Verify the state with commands, not assumptions
4. Know how to recover from failure

The specific tools matter less than the habit. Whether you use UFW or nftables, fail2ban or CrowdSec, unattended-upgrades or a scheduled Ansible playbook — the underlying practice is the same: define what "correct" looks like, verify it is true, and have a recovery path when it is not.

You just did all three of those things. That is what DevOps engineers do.

---

## What You Learned

- How to define a VM in `yeast.yaml` and provision it automatically
- How Yeast boot and provisioning work under the hood
- UFW: enable, allow, deny, status, reload
- Fail2ban: what it does, how to check the sshd jail, why it matters
- Unattended-upgrades: configuration and verification
- Systemd: `is-active`, `enable`, `start` — and why all three can be different states
- Why hostname and timezone are operational settings, not cosmetic ones
- The firewall testing discipline: always test from a second session
- Yeast snapshots: take them before risky changes, restore to recover

---

## Next Lab

**Lab 02: Static Site With Nginx**

You have a clean server. Time to put it to work. In the next lab you will install Nginx, deploy a static website, configure the virtual host, and learn to debug web server problems from the command line. You will also deliberately break the web server in several ways and trace each failure through the logs.
