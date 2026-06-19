# Lab 06 — Bash Automation For Server Setup

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner to intermediate |
| Estimated time | 60-90 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2209 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic shell scripting comfort

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

You have set up servers manually five times now. You know the steps: install packages, set hostname, configure the firewall, enable services. You could do it in your sleep.

But your team is growing. Next week you need to set up three more servers — same baseline configuration. The week after that, probably five more. And every time you do it manually, there is a chance you miss a step, make a typo, or forget to enable fail2ban.

Manual processes do not scale. They do not reproduce reliably. They leave no record of what was done. Every senior engineer has a story about "the server nobody knows how to recreate."

The solution is automation. When you automate a task with a script, you get:
- Repeatability — the same steps, every time, in the same order
- A record — the script itself is documentation
- Speed — what takes 10 minutes of careful manual work takes 30 seconds
- Confidence — you know exactly what was done because you can read the script

This lab teaches you to write good Bash automation. Not just scripts that work — scripts that are safe, readable, and idempotent.

---

## Before You Start — Understanding The Concepts

### What Is Bash?

Bash (Bourne Again Shell) is the command-line interpreter you have been using throughout this bootcamp. When you type commands in a terminal, Bash is what reads and runs them.

Bash is also a programming language. You can write files containing bash commands — called shell scripts — and run them as programs. Scripts let you combine sequences of commands, add conditionals, loops, variables, and error handling.

### What Is Idempotency?

Idempotency is the property of an operation that can be applied multiple times and always produce the same result.

Consider two approaches to creating a directory:

**Not idempotent:**

```bash
mkdir /etc/myapp
```

Run this twice and you get an error: `mkdir: cannot create directory '/etc/myapp': File exists`

**Idempotent:**

```bash
mkdir -p /etc/myapp
```

Run this ten times — same result, no errors.

Idempotency is essential for automation scripts because you will run them more than once. Against new servers (expected). Against an existing server to reapply configuration. After a partial failure where only some steps completed. Your script must handle all of these without breaking.

### What Is `set -euo pipefail`?

Almost every good Bash script starts with:

```bash
set -euo pipefail
```

These are options that make Bash safer:

**`-e`** (exit on error) — the script stops immediately if any command returns a non-zero exit code. Without this, a failed command is silently ignored and the script continues.

**`-u`** (undefined variable) — the script stops if you use a variable that was never set. Without this, `echo $TYOP` prints an empty string instead of telling you `TYOP` is not defined.

**`-o pipefail`** — the script fails if any command in a pipeline fails. Without this, `cat nonexistent.txt | wc -l` exits successfully (because `wc -l` succeeded) even though `cat` failed.

These three options together catch the majority of common Bash scripting bugs.

### What Is An Exit Code?

Every command in Linux returns an exit code when it finishes. Exit code `0` means success. Any non-zero exit code means failure.

```bash
# Check exit code of the last command
echo "hello"
echo $?  # prints 0

ls /nonexistent
echo $?  # prints 2 (non-zero = error)
```

With `set -e`, your script stops the moment any command returns non-zero. This is what you want.

### What Is A Function In Bash?

You can group commands into named functions:

```bash
log() {
    echo "[$(date '+%H:%M:%S')] $*"
}

log "Installing packages..."
```

Functions make scripts readable. Long scripts without functions become impossible to understand.

### What Is Logging In A Script?

Good automation scripts log what they are doing:
- It tells you what happened when something goes wrong
- It gives you an audit trail
- It makes debugging much easier

In this lab, every step writes to `/var/log/server-setup.log` as well as the terminal.

---

## What You Are Building

A Bash script that automates the full server baseline setup from Lab 01 — and runs safely whether it is the first time or the tenth time.

The script will:
- Install all baseline packages
- Configure hostname and timezone
- Set up UFW firewall with the right rules
- Enable and start fail2ban and unattended-upgrades
- Log every step with timestamps
- Be safe to run multiple times without errors or side effects

---

## Starting The Lab

```bash
cd 06-bash-automation-server-setup
yeast up
yeast ssh automate
```

The VM starts with minimal provisioning — just the base OS and SSH. You will do the rest via your script.

---

## Writing The Setup Script

Create the script on the VM:

```bash
sudo vim /usr/local/bin/server-setup.sh
```

Write this content:

```bash
#!/usr/bin/env bash
# server-setup.sh — Baseline server setup script
# Idempotent: safe to run multiple times

set -euo pipefail

LOG_FILE="/var/log/server-setup.log"
SCRIPT_VERSION="1.0"

# -- Helpers --

log() {
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] $*"
    echo "$msg"
    echo "$msg" >> "$LOG_FILE"
}

require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        echo "ERROR: This script must be run as root (use sudo)" >&2
        exit 1
    fi
}

package_installed() {
    dpkg -l "$1" 2>/dev/null | grep -q "^ii"
}

service_active() {
    systemctl is-active --quiet "$1"
}

# -- Main --

require_root

log "=== server-setup.sh v${SCRIPT_VERSION} starting ==="

# 1. System updates
log "Step 1: Updating package lists"
apt-get update -qq

# 2. Install packages (idempotent — apt skips already-installed packages)
PACKAGES=(
    curl
    wget
    vim
    htop
    ufw
    fail2ban
    unattended-upgrades
)

log "Step 2: Installing packages: ${PACKAGES[*]}"
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "${PACKAGES[@]}"

# 3. Hostname (only change if different)
DESIRED_HOSTNAME="${SERVER_HOSTNAME:-$(hostname)}"
CURRENT_HOSTNAME="$(hostname)"

if [ "$CURRENT_HOSTNAME" != "$DESIRED_HOSTNAME" ]; then
    log "Step 3: Setting hostname to ${DESIRED_HOSTNAME} (was: ${CURRENT_HOSTNAME})"
    hostnamectl set-hostname "$DESIRED_HOSTNAME"
else
    log "Step 3: Hostname already set to ${DESIRED_HOSTNAME} — skipping"
fi

# 4. Timezone
DESIRED_TZ="${TIMEZONE:-UTC}"
CURRENT_TZ="$(timedatectl | grep 'Time zone' | awk '{print $3}')"

if [ "$CURRENT_TZ" != "$DESIRED_TZ" ]; then
    log "Step 4: Setting timezone to ${DESIRED_TZ} (was: ${CURRENT_TZ})"
    timedatectl set-timezone "$DESIRED_TZ"
else
    log "Step 4: Timezone already ${DESIRED_TZ} — skipping"
fi

# 5. Firewall
log "Step 5: Configuring UFW"

ufw allow OpenSSH

UFW_STATUS="$(ufw status | head -1)"
if echo "$UFW_STATUS" | grep -q "inactive"; then
    log "Step 5: Enabling UFW"
    ufw --force enable
else
    log "Step 5: UFW already active — verifying rules"
    ufw reload
fi

# 6. fail2ban
log "Step 6: Enabling fail2ban"
systemctl enable fail2ban
if ! service_active fail2ban; then
    systemctl start fail2ban
    log "Step 6: fail2ban started"
else
    log "Step 6: fail2ban already running"
fi

# 7. Unattended upgrades
log "Step 7: Enabling unattended-upgrades"
systemctl enable unattended-upgrades
if ! service_active unattended-upgrades; then
    systemctl start unattended-upgrades
    log "Step 7: unattended-upgrades started"
else
    log "Step 7: unattended-upgrades already running"
fi

# 8. Verification
log "Step 8: Verifying setup"
ERRORS=0

verify_service() {
    local svc="$1"
    if service_active "$svc"; then
        log "  OK: ${svc} is active"
    else
        log "  FAIL: ${svc} is NOT active"
        ERRORS=$((ERRORS + 1))
    fi
}

verify_service fail2ban
verify_service unattended-upgrades

UFW_STATUS="$(ufw status | head -1)"
if echo "$UFW_STATUS" | grep -q "active"; then
    log "  OK: UFW is active"
else
    log "  FAIL: UFW is not active"
    ERRORS=$((ERRORS + 1))
fi

if [ "$ERRORS" -gt 0 ]; then
    log "=== Setup completed with ${ERRORS} verification error(s). Check log: ${LOG_FILE} ==="
    exit 1
fi

log "=== Setup complete. All checks passed. ==="
```

Make it executable:

```bash
sudo chmod +x /usr/local/bin/server-setup.sh
```

---

## Understanding Every Part Of The Script

Let's walk through the concepts used.

### The shebang line

```bash
#!/usr/bin/env bash
```

The first line of every script is the "shebang." When the OS sees this line, it knows what interpreter to use to run the file. `/usr/bin/env bash` finds `bash` in the current environment's PATH — more portable than hardcoding `/bin/bash`.

### Error handling

```bash
set -euo pipefail
```

Covered in the concepts section above. Never skip this line.

### The `log()` function

```bash
log() {
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] $*"
    echo "$msg"
    echo "$msg" >> "$LOG_FILE"
}
```

`$*` is all the arguments passed to the function as a single string. `>>` appends to the file. The log goes to both the terminal and the log file simultaneously.

### The `require_root()` function

```bash
require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        echo "ERROR: This script must be run as root (use sudo)" >&2
        exit 1
    fi
}
```

`id -u` returns the numeric user ID. Root is always 0. `-ne 0` means "not equal to zero." `>&2` redirects to stderr (standard error) rather than stdout. Scripts should send error messages to stderr.

### Idempotent hostname check

```bash
if [ "$CURRENT_HOSTNAME" != "$DESIRED_HOSTNAME" ]; then
    hostnamectl set-hostname "$DESIRED_HOSTNAME"
else
    log "Hostname already set — skipping"
fi
```

Instead of blindly running `hostnamectl set-hostname` every time, we check first. If it is already correct, we skip it and log that. This is idempotency in practice.

### Environment variable configuration

```bash
DESIRED_HOSTNAME="${SERVER_HOSTNAME:-$(hostname)}"
```

`${VAR:-default}` means "use `$VAR` if it is set, otherwise use `default`." This lets you override the hostname by setting an environment variable before running the script:

```bash
sudo SERVER_HOSTNAME=web-01 TIMEZONE=America/New_York bash /usr/local/bin/server-setup.sh
```

Without the override, the script uses the current hostname. This makes the script reusable across different servers.

### The `package_installed()` function

```bash
package_installed() {
    dpkg -l "$1" 2>/dev/null | grep -q "^ii"
}
```

`dpkg -l` lists installed packages. `^ii` matches lines that start with `ii` — the status code for "installed and configured." You could use this to skip `apt install` for already-installed packages, but `apt install` itself is already idempotent (it skips packages that are current version), so this function is more useful for conditional logic like "only configure nginx if nginx is installed."

---

## Running The Script

Run it as root (via sudo):

```bash
sudo bash /usr/local/bin/server-setup.sh
```

Watch the output. Every step is logged with a timestamp. You should see it:
1. Update package lists
2. Install packages (fast because some are already installed)
3. Check hostname (already set — skip)
4. Check timezone (already UTC — skip)
5. Configure UFW
6. Enable and start fail2ban
7. Enable and start unattended-upgrades
8. Verify everything

Now run it again:

```bash
sudo bash /usr/local/bin/server-setup.sh
```

Every step should either say "already done" or complete without error. That is idempotency working correctly.

---

## Reading The Log

```bash
cat /var/log/server-setup.log
```

You get a timestamped record of exactly what happened, when, and the result. This is your audit trail.

In a real team, you would ship this log to a centralized logging system (Lab 18 covers this). For now, it lives on the server.

---

## Test Idempotency — Partial Failure Recovery

Simulate a situation where the script ran but failed halfway through:

```bash
# Manually put the system in a partially-configured state
sudo systemctl stop fail2ban
sudo ufw disable
```

Now run the script again:

```bash
sudo bash /usr/local/bin/server-setup.sh
```

The script detects the incomplete state and corrects it:
- Detects UFW is inactive, enables it
- Detects fail2ban is stopped, starts it

Check the log:

```bash
tail -20 /var/log/server-setup.log
```

You can see the decisions the script made. The second run fixed what was broken without touching what was already correct.

---

## Making It Configurable

Run the script with a custom hostname and timezone:

```bash
sudo SERVER_HOSTNAME=web-01 TIMEZONE=UTC bash /usr/local/bin/server-setup.sh
```

The script sets the hostname to `web-01` and confirms the timezone. Same script, different server.

---

## Why Not Just Use yeast.yaml For All Of This?

Good question. You already did this setup via `yeast.yaml` provision blocks in earlier labs.

The answer is: both have their place.

**`yeast.yaml` provisioning** — runs once on first boot, is great for getting a VM to a starting state. You cannot easily run it again.

**A standalone script** — can be run at any time, against any server, whether it was created by Yeast or not. It works on a physical machine, a cloud VM, or a VM created by any other tool.

In production, you will use tools like Ansible (Lab 07) for this kind of repeatable configuration. But Ansible is built on the same principles you practiced here — idempotency, logging, verification. Understanding Bash automation first makes Ansible click immediately when you get there.

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
yeast destroy
```

---

## Quick Recap

In Lab 06 — Bash Automation For Server Setup, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- `set -euo pipefail` — why every Bash script needs this
- Idempotency — what it means and how to implement it with conditionals
- Functions in Bash — how to organize scripts into readable units
- `log()` — writing timestamped log output to file and terminal simultaneously
- `require_root()` — how to validate script preconditions before doing anything
- `${VAR:-default}` — configurable scripts via environment variables
- Exit codes — how scripts signal success or failure
- `dpkg -l` and `systemctl is-active` — querying system state for conditional logic
- Why automation scripts are not just "commands in a file" — they are programs that need error handling, logging, and idempotency

---

## What Is Next

**Lab 07 — Ansible For One Server**

Bash scripts work. But as your infrastructure grows, they get harder to maintain — different scripts for different server types, no central inventory, hard to run against many servers at once. Ansible solves these problems with a structured, declarative approach to configuration management. It also handles all the idempotency for you automatically.
