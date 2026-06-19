# Lab 01 — Linux Server Baseline

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner |
| Estimated time | 45-75 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2201 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether the lab asked you to open an SSH tunnel first.

### Expected Checkpoints

- After `yeast up`, `yeast status` should show the expected VM or VMs as running.
- After the main setup steps, the service, tool, or workflow introduced by the lab should respond to the verification commands.
- After `bash assets/validate.sh`, the script should report all checks passed.
- After `yeast destroy`, the lab should be cleaned up before you start the next one.

### Common Mistakes To Avoid

- Running a VM command on your laptop, or a laptop command inside the VM.
- Closing an SSH tunnel and then wondering why `localhost:<port>` stopped working.
- Skipping validation because the final page or command "looked fine".
- Forgetting to run `yeast destroy` before moving to the next lab.

---

## The Story

It is your first week at a small startup. The CTO tells you: "We just got a new server. Set it up properly before we deploy anything on it."

You ask: what does "properly" mean?

She says: "Hostname is set. Timezone is UTC. SSH is the only thing reachable from the network. Automatic security patches are running. No unpatched vulnerabilities from day one. I want to be able to SSH in and know within 60 seconds that the server is healthy."

That is your task for today.

It sounds simple. But every senior engineer you will ever meet has a story about the production incident that happened because someone skipped this step. A server with the wrong timezone makes log correlation a nightmare. A server with no firewall is exposed to the entire internet. A server that never patches itself will get compromised within months.

The baseline is not optional. It is the thing everything else is built on.

---

## Before You Start — Understanding The Tools

You are about to use several things you may never have touched before. Let's understand each one before using it.

### What Is Linux?

Linux is an operating system — the software that manages a computer's hardware and lets other programs run on top of it. When you hear "Linux server," it means a computer running the Linux operating system, usually without a graphical interface. You interact with it entirely through a terminal.

Almost every server in the world — every web server, every database, every cloud VM — runs Linux. Learning to operate Linux servers is not optional for a DevOps engineer. It is the foundation.

### What Is A Virtual Machine?

A virtual machine (VM) is a computer that runs inside another computer. Using software called a hypervisor, your laptop can pretend to be multiple separate computers at the same time. Each VM has its own operating system, its own disk, its own network interface. From the inside, it looks and behaves exactly like a real physical machine.

You use VMs because:
- You can create and destroy them in seconds, without buying hardware
- You can snapshot them (freeze their exact state) and restore to that snapshot later
- You can experiment freely — if you break something, you destroy it and start again
- Multiple VMs let you build realistic multi-machine architectures on one laptop

### What Is QEMU/KVM?

QEMU is a piece of software that creates and runs virtual machines. KVM (Kernel-based Virtual Machine) is a feature built into the Linux kernel that lets QEMU run VMs at near-native speed by using your CPU's hardware virtualization support.

When you run `yeast up`, Yeast uses QEMU/KVM to create a real virtual machine on your laptop. The VM gets its own CPU threads, its own chunk of RAM, and its own virtual disk.

You need `/dev/kvm` to exist on your system, which means your CPU supports hardware virtualization and the kernel module is loaded. Run `yeast doctor` to check this.

### What Is Yeast?

Yeast is the tool that manages your virtual machines for this bootcamp. Think of it as a simple, local version of the tools that cloud providers use (like AWS EC2 or DigitalOcean Droplets), but running entirely on your laptop.

You define what you want in a file called `yeast.yaml`. Yeast reads that file and creates the VM for you — the right size, the right operating system, with the right software installed and the right configuration applied.

The key concept is **declarative**: you describe the desired end state, not the steps to get there. Yeast figures out the steps.

### What Is SSH?

SSH stands for Secure Shell. It is a protocol that lets you open a terminal session on a remote computer securely over a network connection.

When you `yeast up`, the VM boots and starts an SSH server. You then connect to it with `yeast ssh <name>`, which opens a terminal on the VM. From that terminal you can run commands on the VM as if you were sitting at its keyboard.

SSH uses public-key cryptography for authentication. Yeast automatically generates a key pair and installs the public key into the VM during provisioning, so you never need to type a password.

### What Is Cloud-Init?

Cloud-init is software that runs during the very first boot of a Linux server. It is how cloud providers (and Yeast) configure a fresh machine: it creates users, sets hostnames, copies SSH keys, and runs setup scripts.

When Yeast boots your VM for the first time, cloud-init runs and sets up the `ubuntu` user with your SSH key. After that, Yeast SSHes in and runs the `provision` block from your `yeast.yaml`.

### What Is Ubuntu?

Ubuntu is one of the most popular Linux distributions (a "distro" is a packaged version of Linux with specific software and tools included). Ubuntu 22.04 LTS (Long Term Support) is a version released in April 2022 that receives security updates until 2027. It is stable, widely used, and well-documented.

In this bootcamp, Ubuntu 22.04 is the base OS for all your VMs.

---

## What You Are Building

One virtual machine. Ubuntu 22.04. Configured to this standard:

```
hostname:              baseline
timezone:              UTC
firewall:              on — SSH only, everything else blocked
automatic updates:     on — security patches apply without manual action
SSH brute-force guard: on — repeated failed logins trigger an IP ban
monitoring tools:      htop, curl, wget, vim installed
```

```
Your Laptop
    │
    │  SSH (port 2201 → VM port 22)
    ▼
┌──────────────────────────┐
│  baseline (Ubuntu 22.04) │
│                          │
│  ufw firewall: on        │
│  fail2ban: active        │
│  unattended-upgrades: on │
└──────────────────────────┘
```

That is the whole architecture. One machine. No external network. No other services. The goal is to get one server into a known, safe, verifiable state before we put anything on it.

---

## The Config File — assets/yeast.yaml

Open `assets/yeast.yaml` and read it now, before running anything. Understanding your config before running it is a habit you need to build.

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

Let's read every field:

**`version: 1`** — This tells Yeast which version of the config format you are using. Always 1 for this bootcamp.

**`name: baseline`** — The name Yeast uses to refer to this VM in commands. `yeast ssh baseline`, `yeast down`, `yeast destroy`. Pick names that describe the machine's role.

**`hostname: baseline`** — The hostname that the Linux OS inside the VM will use. This appears in your shell prompt (`ubuntu@baseline:~$`), in log files, and anywhere the machine identifies itself. It should match the VM's name so you always know which machine you are looking at.

**`image: ubuntu-22.04`** — The base operating system image. Yeast downloads this from a trusted image registry the first time you use it. After that it is cached locally.

**`memory: 1024`** — RAM in megabytes. 1024 MB = 1 GB. That is enough for a baseline server running no applications. Multi-VM labs later in the course will need more.

**`cpus: 2`** — How many virtual CPU cores the VM gets. 2 is fine for most of this bootcamp.

**`disk_size: 20G`** — The VM's virtual disk size. 20 GB is more than enough for everything in this lab.

**`ssh_port: 2201`** — This is the key management networking concept for Yeast. The VM's SSH server listens on port 22 inside the VM. Yeast exposes that SSH management connection on port 2201 on your laptop. So when you run `yeast ssh baseline`, Yeast connects to `localhost:2201`, which reaches the VM's SSH service.

Why not use port 22 directly? Because your laptop might already have an SSH server on port 22, and different labs use different VMs that would all need different ports. Each VM in this bootcamp gets its own SSH port on your laptop.

**`user: ubuntu`** — The default user created by Ubuntu's cloud image. Yeast SSHes in as this user.

**`sudo: nopasswd`** — Allows the `ubuntu` user to run `sudo` (which gives root/admin privileges) without typing a password. Required because Yeast runs provisioning commands non-interactively and cannot type a password. In production you would tighten this.

**`provision.packages`** — A list of packages to install via `apt` (Ubuntu's package manager). Let's look at each one:

- **`curl`** — A tool to make HTTP requests from the command line. You will use this constantly to test if web services are responding.
- **`wget`** — Similar to curl, used for downloading files. Some scripts use one, some use the other.
- **`vim`** — A text editor that works in the terminal. Essential when you need to edit config files on a server.
- **`htop`** — An interactive process viewer. Shows you CPU, RAM, and running processes in real time. Like Task Manager, but in a terminal and much more useful.
- **`ufw`** — Uncomplicated Firewall. The tool we use to manage which network ports are open.
- **`unattended-upgrades`** — Automatically installs security updates without you having to log in and run `apt upgrade` manually.
- **`fail2ban`** — Watches log files for repeated failed login attempts and temporarily bans those IP addresses.

**`provision.shell`** — Commands that run after packages are installed, in order:

- **`hostnamectl set-hostname baseline`** — Sets the system hostname. Without this it defaults to a random cloud-init hostname.
- **`timedatectl set-timezone UTC`** — Sets the timezone to UTC. Every server in a fleet should use UTC so log timestamps are comparable.
- **`ufw allow OpenSSH`** — Tells the firewall to allow SSH connections. We must do this BEFORE enabling the firewall, or we will lock ourselves out.
- **`ufw --force enable`** — Turns the firewall on. The `--force` flag skips the interactive "are you sure?" prompt that would hang a non-interactive script.
- **`systemctl enable --now fail2ban`** — Enables fail2ban (so it starts on every boot) and starts it right now.
- **`systemctl enable --now unattended-upgrades`** — Same pattern: enable and start.

---

## Starting The Lab

Make sure you are in the lab folder:

```bash
cd 01-linux-server-baseline
```

Now run:

```bash
yeast up
```

Here is what Yeast does when you run this:

1. Reads `yeast.yaml` in the current directory
2. Checks if the `ubuntu-22.04` image is cached locally — if not, downloads it (~500 MB, one time only)
3. Creates a QEMU/KVM virtual machine with the specified RAM, CPUs, and disk
4. Boots the VM using cloud-init to set up the user and SSH key
5. Waits for the VM to be reachable over SSH (usually 30–60 seconds)
6. SSHes in as `ubuntu` and runs the provision block: installs packages, runs the shell commands
7. Reports success when everything is done

The first run takes 60–120 seconds. Subsequent runs of the same VM are faster because the image is already cached.

You can watch what Yeast is doing:

```bash
yeast up --events
```

This streams the provisioning output live so you can see each step as it happens.

---

## Getting A Shell On The VM

Once `yeast up` finishes, connect to your new VM:

```bash
yeast ssh baseline
```

Your terminal prompt will change to something like:

```
ubuntu@baseline:~$
```

You are now inside the virtual machine. The `ubuntu` is your username. `baseline` is the hostname. The `~` is your current directory (the home directory). The `$` means you are a regular user (not root).

Everything you type now runs inside the VM, not on your laptop. To get back to your laptop, type `exit` or press `Ctrl+D`.

---

## Verifying The Baseline

This is the most important part of the lab. Do not skip it. Verification is not optional extra credit — it is the job. An engineer who provisions a server and does not verify it has not done the job. They have done half the job.

### Check the hostname

```bash
hostname
```

Expected output:

```
baseline
```

If it says `ubuntu` or something random, the `hostnamectl` command in provisioning failed. Fix it manually:

```bash
sudo hostnamectl set-hostname baseline
```

What is `sudo`? Sudo stands for "superuser do." Some commands require root (administrator) privileges to run. Instead of logging in as root, you prefix the command with `sudo` and it runs with elevated privileges. The `ubuntu` user has `sudo nopasswd` configured, meaning it can do this without a password.

### Check the timezone

```bash
timedatectl
```

Expected output (look for these lines):

```
               Local time: Mon 2026-06-15 12:00:00 UTC
           Universal time: Mon 2026-06-15 12:00:00 UTC
                 RTC time: Mon 2026-06-15 12:00:00
                Time zone: UTC (UTC, +0000)
```

Why does this matter? When you have 10 servers and they all log events, if some are in UTC+1 and some in UTC-5, correlating events across machines becomes a nightmare. A request that hit the web server at 14:00 UTC and the database at 14:00:04 UTC will appear to be 5 hours apart if the database uses EST. UTC everywhere eliminates that ambiguity entirely.

### Check the OS version

```bash
cat /etc/os-release
```

`cat` is a command that prints the contents of a file. `/etc/os-release` is a file that contains information about the operating system. You should see:

```
PRETTY_NAME="Ubuntu 22.04.X LTS"
NAME="Ubuntu"
VERSION_ID="22.04"
...
```

### Check the kernel

```bash
uname -r
```

`uname` prints system information. The `-r` flag means "release" — it prints the kernel version number. The kernel is the core of the operating system. You will see something like `5.15.0-100-generic`. The exact number does not matter right now — just confirm the command works.

### Check who you are and confirm sudo

```bash
whoami
```

Returns: `ubuntu`

```bash
sudo whoami
```

Returns: `root`

`sudo whoami` runs the `whoami` command as root and prints its result. This confirms that:
1. Sudo is working
2. The nopasswd config is in place (it did not ask for a password)
3. You can run privileged commands when needed

---

## Understanding And Verifying The Firewall

### What is a firewall?

A firewall is software that controls which network connections are allowed in and out of a machine.

Every machine connected to a network is reachable on numbered "ports." Port 22 is SSH. Port 80 is HTTP. Port 443 is HTTPS. Port 5432 is PostgreSQL. There are 65,535 possible ports, and without a firewall, any service listening on any port is reachable by anyone on the network.

A firewall's job is to enforce a policy: "only these ports are allowed in, block everything else." On a fresh Ubuntu server with a firewall, the only thing reachable is what you explicitly allow.

### UFW — Uncomplicated Firewall

UFW is Ubuntu's interface for managing firewall rules. Under the hood it configures `iptables` (the Linux kernel's built-in packet filtering), but UFW gives you a much simpler syntax.

Check the status:

```bash
sudo ufw status verbose
```

Expected output:

```
Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), disabled (routed)
New profiles: skip

To                         Action      From
--                         ------      ----
OpenSSH                    ALLOW IN    Anywhere
OpenSSH (v6)               ALLOW IN    Anywhere (v6)
```

Read this carefully:

- **`Status: active`** — the firewall is on
- **`Default: deny (incoming)`** — unless explicitly allowed, all incoming connections are blocked
- **`Default: allow (outgoing)`** — the server can make outbound connections (needed for downloading packages, etc.)
- **`OpenSSH ALLOW IN Anywhere`** — port 22 (SSH) is allowed

Right now this VM is as locked down as it can be while still being accessible. Port 22 is open because you need to connect. Every other port is blocked. That is the correct starting posture.

---

## Understanding And Verifying Fail2ban

### What problem does fail2ban solve?

Every machine connected to the internet with SSH open gets probed constantly. Automated bots scan IP ranges 24 hours a day trying to find machines with weak passwords or default credentials. This is not theoretical — if you run a public-facing SSH server for 24 hours, you will see thousands of failed login attempts in your auth log.

Fail2ban watches `/var/log/auth.log` for repeated failed login attempts. When the same IP fails to authenticate too many times in a short window (default: 5 failures in 10 minutes), fail2ban adds a firewall rule to block that IP for a period of time (default: 10 minutes).

This does not stop a sophisticated attacker. But it eliminates the automated noise and makes brute-force attacks vastly less effective.

### Checking fail2ban

```bash
sudo systemctl is-active fail2ban
```

Expected: `active`

What does `systemctl` do? Systemd is the process manager built into modern Linux. It manages "services" — programs that run in the background. `systemctl` is the command you use to talk to systemd.

- `systemctl start <service>` — starts a service now
- `systemctl stop <service>` — stops a service now
- `systemctl enable <service>` — marks a service to start automatically on boot
- `systemctl disable <service>` — removes the auto-start mark
- `systemctl is-active <service>` — prints "active" if running, "inactive" if not

A service can be "enabled" but "inactive" (it will start on next boot but is not running now), or "active" but "disabled" (running now but will not restart after reboot). For production services you want both enabled AND active.

Now check the SSH jail:

```bash
sudo fail2ban-client status sshd
```

Expected output:

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

"Jail" in fail2ban terms means a set of rules for watching a specific log file for a specific pattern. The `sshd` jail watches for SSH auth failures. Zero bans is expected — the VM just started.

---

## Understanding And Verifying Unattended-Upgrades

### Why automatic updates matter

The moment a security vulnerability is disclosed, attackers start scanning for unpatched systems. The time between "vulnerability announced" and "active exploitation in the wild" is sometimes hours.

If you rely on a human to log into every server and run `apt upgrade` manually, that window can stretch to days or weeks. Unattended-upgrades closes that gap by automatically applying security patches on a schedule (by default, daily).

It only applies *security* updates by default — not every available update. That keeps the risk of an update breaking something low.

### Verifying it

```bash
sudo systemctl is-active unattended-upgrades
```

Expected: `active`

Check the configuration:

```bash
cat /etc/apt/apt.conf.d/20auto-upgrades
```

Expected:

```
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
```

The `"1"` means "do this every 1 day." This file tells Ubuntu's package system to: (1) update the list of available packages every day, and (2) apply security upgrades every day.

---

## Break Something On Purpose

Reading about failure is not the same as experiencing it. We are going to deliberately break the firewall, experience being locked out, and then recover.

**Open a second terminal on your laptop before doing this. Keep your current SSH session open.**

Inside the VM (in your first terminal), add a deny rule for SSH:

```bash
sudo ufw deny OpenSSH
sudo ufw reload
```

What just happened? You told the firewall to explicitly deny SSH connections. The `reload` command makes the firewall apply the new rules immediately.

Now go to your **second terminal** on your laptop and try to connect:

```bash
yeast ssh baseline
```

It will hang or return "Connection refused." You are locked out of the VM from any new session.

This is exactly what happens when a real engineer misconfigures a firewall on a production server. The difference on a real server: you might not have a second session open. You might not have console access. Recovery could require a support ticket to the cloud provider for "out-of-band" access.

Now go back to your **first terminal** — the one that was already connected before you changed the rules. Fix it:

```bash
sudo ufw allow OpenSSH
sudo ufw delete deny OpenSSH
sudo ufw reload
sudo ufw status
```

The order matters here. You allow SSH first, then delete the deny rule. If you did it in the wrong order, deleting the deny rule while it was the only rule would leave an ambiguous state.

From your second terminal, try again:

```bash
yeast ssh baseline
```

It works.

**The lesson to internalize:** Always test firewall changes from a second, already-open session. Never close the first session until you have confirmed the second session can connect. On a machine where you cannot get console access (most cloud VMs), this discipline is the difference between a 30-second fix and a server rebuild.

---

## Taking A Snapshot

Yeast lets you snapshot a VM — freeze its entire disk state so you can return to it later.

Take a snapshot of the clean baseline state right now:

```bash
# Back on your laptop (exit the SSH session first)
exit

yeast snapshot baseline clean-baseline
```

Now if you break something while experimenting — even if the VM becomes completely unusable — you can restore to this exact state:

```bash
yeast restore baseline clean-baseline
```

Get into the habit of snapshotting after every "working state." In the chaos and failure labs later in this course, you will rely on this heavily.

---

## Validate Your Work

Run the automated validation script from your laptop:

```bash
bash assets/validate.sh
```

The script SSHes into the VM and checks every item from this lab automatically. If something fails, it tells you what it checked and what it found. Fix the issue and run it again.

---

## Clean Up

When you are done:

```bash
yeast destroy
```

This stops the VM and removes it. The snapshot you took earlier is preserved.

Every lab ends with `yeast destroy`. Good operational habit: do not leave VMs running when you are not using them.

---

## Quick Recap

In Lab 01 — Linux Server Baseline, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

You spent time on one Linux server today. Not on Kubernetes, not on Docker, not on CI pipelines. Just one machine. That was intentional.

Here is what you did:

- You read a declarative config file (`yeast.yaml`) and understood every field before running it
- You learned what virtual machines are and how Yeast uses QEMU/KVM to create them
- You learned what SSH is and how to use it to get a terminal on a remote machine
- You verified the hostname and timezone — the two things that make fleets of servers manageable
- You verified UFW and understood what a firewall policy means: deny everything, allow only what you need
- You verified fail2ban and understood why SSH brute-force protection matters
- You verified unattended-upgrades and understood why automatic security patching matters
- You deliberately broke the firewall, experienced what being locked out feels like, and recovered safely
- You took a snapshot — your first recovery checkpoint

Every lab that follows adds something on top of this foundation. You will deploy web servers, databases, containers, pipelines. But every one of those things lives on a Linux server that should pass the same checks you ran today.

---

## What Is Next

**Lab 02 — Static Site With Nginx**

You have a clean, secured server. Now put it to work. In the next lab you will install Nginx, deploy a static website, understand how web servers and ports work, read access and error logs, and learn to debug HTTP failures from the command line.
