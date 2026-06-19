# Lab 07 — Ansible For One Server

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner to intermediate |
| Estimated time | 60-90 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2210 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Ansible installed on your laptop

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether the lab asked you to open an SSH tunnel first.
- Run Ansible commands from your laptop; Ansible connects to the VMs over SSH.

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
- Forgetting that Ansible runs from your laptop but manages the VM over SSH.

---

## The Story

Your Bash script from Lab 06 works. You can set up a server reliably. But now your team has ten servers — different roles, different configurations. You have a web server setup script, a database setup script, a monitoring agent script. They all do slightly different things. When something changes — say, a new security policy requires a different UFW rule — you update each script separately. You inevitably forget one server.

This is the problem Ansible was built to solve.

Ansible is a configuration management tool. You describe your servers and their desired state in YAML files called playbooks. Ansible connects to each server via SSH, compares the current state to the desired state, and makes only the changes needed. Run the same playbook twice and the second run changes nothing — because the server is already in the right state.

---

## Before You Start — Understanding The Concepts

### What Is Configuration Management?

Configuration management is the practice of tracking and controlling the state of your infrastructure. A configuration management tool lets you:
- Define what every server should look like (packages, services, files, users)
- Apply that definition to any number of servers
- Detect and correct drift — when a server's actual state differs from the desired state
- Have a single source of truth for your infrastructure's configuration

Without configuration management, servers drift over time. Someone SSHes in and installs something manually. A config file gets hand-edited. A service gets disabled. Nobody remembers what changed. Eventually you have a fleet of servers that are all slightly different from each other and nobody knows exactly what is on them.

### What Is Ansible?

Ansible is an agentless configuration management tool. Unlike tools that require a special agent process running on every managed server, Ansible only needs:
- Python on the managed server (almost always already there)
- SSH access from your control machine (your laptop) to the server

You write playbooks in YAML. Ansible connects to servers over SSH and runs modules — small Python programs that perform specific tasks. Modules are idempotent by design: the `apt` module only installs a package if it is not already installed.

### Key Ansible Concepts

**Control node** — the machine running Ansible. In this lab, your laptop.

**Managed node** — the machine Ansible configures. The Yeast VM.

**Inventory** — a file listing which servers Ansible manages, grouped by role or environment.

**Playbook** — a YAML file describing the desired state of one or more servers. Composed of plays.

**Play** — a mapping of a group of hosts to a list of tasks.

**Task** — a single unit of work: install a package, create a file, enable a service.

**Module** — the Ansible component that performs a task. `ansible.builtin.apt`, `ansible.builtin.service`, `ansible.builtin.template`, etc.

**Handler** — a task triggered only when something changes. "Reload Nginx after its config changes."

**Role** — a structured collection of tasks, files, templates, and variables that can be reused across playbooks.

### Idempotency In Ansible

Ansible modules are designed to be idempotent. The `apt` module checks if the package is already installed before installing it. The `service` module checks if the service is already running before starting it. The `copy` module compares file content before writing.

When you run a playbook, each task reports its status:
- `ok` — the state was already correct, nothing changed
- `changed` — Ansible made a change to reach the desired state
- `failed` — something went wrong

A fully applied playbook shows `changed=0` on re-run. That is the goal.

### What Is An Inventory File?

An inventory file tells Ansible which servers to manage and how to connect to them. The simplest form is an INI-style file:

```ini
[webservers]
managed ansible_host=127.0.0.1 ansible_port=2210 ansible_user=ubuntu
```

You can group servers by role (`[webservers]`, `[databases]`, `[monitoring]`) and apply different playbooks or variables to each group.

### What Is A Template?

Ansible templates are files with placeholders that get filled in with variable values when deployed. They use the Jinja2 templating language.

Example Nginx config template:

```
server {
    listen 80;
    server_name {{ server_name }};
    root {{ document_root }};
}
```

When Ansible deploys this template, it replaces `{{ server_name }}` and `{{ document_root }}` with the actual variable values for that host. The same template can configure multiple servers differently based on their variables.

---

## What You Are Building

An Ansible playbook that configures a single server to the same baseline as Lab 01, plus installs Nginx and deploys a simple HTML page. Run it once and the server is configured. Run it again and nothing changes.

You will write the playbook on your laptop and run it against the Yeast VM over SSH.

```
Your Laptop (Ansible control node)
    │
    │  SSH :2210
    │  runs ansible-playbook
    ▼
┌──────────────────────────┐
│  managed (Ubuntu 22.04)  │
│                          │
│  Ansible configures:     │
│  - hostname              │
│  - timezone              │
│  - packages              │
│  - UFW rules             │
│  - fail2ban              │
│  - nginx + site          │
└──────────────────────────┘
```

---

## Installing Ansible

Ansible runs on your laptop (the control node). Install it:

```bash
# Ubuntu/Debian
sudo apt install ansible -y

# Or via pip (gets a newer version)
pip3 install ansible

ansible --version
```

You should see something like: `ansible [core 2.14.x]`

---

## Starting The Lab

```bash
cd 07-ansible-one-server
yeast up
```

The VM boots with just Python installed — the only Ansible requirement on the managed node.

---

## Understanding How Ansible Connects

Yeast connects to the VM with normal system `ssh` on the configured `ssh_port`. Ansible can do the same thing.

Yeast v1.1 does not provide an SSH config export helper. For this lab, the inventory uses:

- `ansible_host=127.0.0.1`
- `ansible_port=2210`
- `ansible_user=ubuntu`
- SSH options matching Yeast's default SSH behavior

---

## Creating The Inventory

Create `inventory.ini` in the lab folder:

```ini
[webservers]
managed ansible_host=127.0.0.1 ansible_port=2210 ansible_user=ubuntu ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'
```

This uses your normal SSH identity, the same kind of connection `yeast ssh managed` uses.

Test Ansible can reach the server:

```bash
ansible -i inventory.ini webservers -m ping
```

Expected output:

```
managed | SUCCESS => {
    "ansible_facts": {
        "discovered_interpreter_python": "/usr/bin/python3"
    },
    "changed": false,
    "ping": "pong"
}
```

`ping` is an Ansible module that tests connectivity. "pong" means it worked.

---

## Creating The Playbook

Create `site.yml`:

```yaml
---
- name: Configure managed server
  hosts: webservers
  become: true

  vars:
    server_hostname: managed
    server_timezone: UTC
    nginx_doc_root: /var/www/html

  tasks:

    - name: Set hostname
      ansible.builtin.hostname:
        name: "{{ server_hostname }}"

    - name: Set timezone
      community.general.timezone:
        name: "{{ server_timezone }}"

    - name: Install required packages
      ansible.builtin.apt:
        name:
          - nginx
          - fail2ban
          - ufw
          - curl
          - vim
          - htop
          - unattended-upgrades
        state: present
        update_cache: true
        cache_valid_time: 3600

    - name: Allow SSH through UFW
      community.general.ufw:
        rule: allow
        name: OpenSSH

    - name: Allow HTTP through UFW
      community.general.ufw:
        rule: allow
        name: "Nginx Full"

    - name: Enable UFW
      community.general.ufw:
        state: enabled

    - name: Enable and start nginx
      ansible.builtin.service:
        name: nginx
        state: started
        enabled: true

    - name: Enable and start fail2ban
      ansible.builtin.service:
        name: fail2ban
        state: started
        enabled: true

    - name: Enable unattended-upgrades
      ansible.builtin.service:
        name: unattended-upgrades
        state: started
        enabled: true

    - name: Create site document root
      ansible.builtin.file:
        path: "{{ nginx_doc_root }}"
        state: directory
        owner: www-data
        group: www-data
        mode: "0755"

    - name: Deploy index.html
      ansible.builtin.copy:
        dest: "{{ nginx_doc_root }}/index.html"
        owner: www-data
        group: www-data
        mode: "0644"
        content: |
          <!DOCTYPE html>
          <html>
          <head><title>Ansible Managed</title></head>
          <body>
            <h1>This server is managed by Ansible</h1>
            <p>Hostname: {{ ansible_hostname }}</p>
          </body>
          </html>
      notify: reload nginx

  handlers:
    - name: reload nginx
      ansible.builtin.service:
        name: nginx
        state: reloaded
```

---

## Reading The Playbook

Let's understand every key concept used here.

### `hosts: webservers`

This play runs against the `[webservers]` group from the inventory. Change this to `all` to run against every server in the inventory.

### `become: true`

Tells Ansible to use privilege escalation — equivalent to running commands with `sudo`. Required for installing packages and modifying system services.

### `vars:`

Variables for this play. Using variables instead of hardcoding values makes the playbook reusable:
- Change `server_hostname` and you can reuse this playbook for a different server
- Later, these variables can come from `host_vars`, `group_vars`, or command-line `-e` flags

### `ansible.builtin.apt`

The apt module manages packages on Debian/Ubuntu systems. `state: present` means "ensure it is installed." `update_cache: true` runs `apt-get update` before installing. `cache_valid_time: 3600` only refreshes if the cache is older than 1 hour — prevents unnecessary network calls on re-runs.

### `community.general.ufw`

The UFW module manages firewall rules. Each task is idempotent — if the rule already exists, it is not added again.

### `ansible.builtin.service`

The service module manages systemd services. `state: started` starts it if stopped. `enabled: true` enables it if disabled. If already started and enabled, the task reports `ok`.

### `ansible.builtin.copy`

Copies content to a file on the remote server. The `content:` key lets you write the file content inline. The module compares content with the current file — if they match, `ok`. If they differ, it updates the file.

### `notify: reload nginx`

When the `copy` task changes the file (content differs), it "notifies" the `reload nginx` handler. Handlers run once at the end of the play, after all tasks complete. Even if five tasks notify the same handler, it only runs once.

This is the correct pattern for config changes: deploy the config, then reload the service. With a handler, the reload only happens when the config actually changed — not every time you run the playbook.

### Handlers

```yaml
handlers:
  - name: reload nginx
    ansible.builtin.service:
      name: nginx
      state: reloaded
```

Handlers are tasks that only run when notified. They run after all regular tasks complete. `reloaded` sends a SIGHUP to Nginx to re-read config without stopping.

---

## Running The Playbook

```bash
ansible-playbook -i inventory.ini site.yml
```

Watch the output carefully. Each task prints its status: `ok` or `changed`.

The first run should show several `changed` items as Ansible configures the server. The last lines summarize:

```
PLAY RECAP ******************************
managed : ok=12  changed=8  unreachable=0  failed=0  skipped=0
```

Now run it again:

```bash
ansible-playbook -i inventory.ini site.yml
```

Expected second run:

```
PLAY RECAP ******************************
managed : ok=12  changed=0  unreachable=0  failed=0  skipped=0
```

`changed=0` — Ansible looked at every task, compared desired state to actual state, and found everything already correct. It made zero changes. This is idempotency.

---

## Verifying The Result

```bash
curl http://localhost:8080
```

You should see the HTML page deployed by Ansible.

SSH in and verify manually:

```bash
yeast ssh managed
sudo systemctl is-active nginx
sudo systemctl is-active fail2ban
sudo ufw status
hostname
```

Everything matches the playbook.

---

## Testing Drift Correction

One of Ansible's most valuable properties is that it corrects drift. Simulate drift by manually changing something on the server:

```bash
yeast ssh managed
sudo systemctl stop fail2ban
exit
```

Now re-run the playbook:

```bash
ansible-playbook -i inventory.ini site.yml
```

The `Enable and start fail2ban` task will show `changed` — Ansible detected the service was stopped and started it again. Everything else shows `ok`.

This is the core value of configuration management: **you can always run the playbook to bring any server back to the correct state**, regardless of what someone did manually.

---

## Adding A Variable — Different Content Per Environment

Change the index.html content for a different environment without modifying the playbook itself:

```bash
ansible-playbook -i inventory.ini site.yml \
  -e "server_hostname=web-staging"
```

The playbook now configures a server named `web-staging` using the same logic. Variables are the mechanism for making one playbook serve many environments.

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

In Lab 07 — Ansible For One Server, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What configuration management is and why it beats shell scripts at scale
- Ansible's architecture: control node, managed nodes, agentless via SSH
- Inventory files: how Ansible knows which servers to manage
- Playbooks: the YAML description of desired server state
- Modules: `apt`, `service`, `copy`, `file`, `hostname`, `ufw` — and that they are all idempotent
- `become: true` — privilege escalation in Ansible
- Variables and `{{ template_syntax }}` — making playbooks reusable
- Handlers: run only when something changes, only once per play
- The `changed=0` test for idempotency
- Drift correction: running the playbook against a drifted server brings it back

---

## What Is Next

**Lab 08 — Ansible For Multi-VM Web Cluster**

One server is straightforward. The real power of Ansible becomes visible when you manage multiple servers — a load balancer and three web nodes — using the same playbook, with templated configs that pull from each host's inventory variables. Lab 08 does exactly that.
