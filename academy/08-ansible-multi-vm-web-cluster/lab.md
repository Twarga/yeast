# Lab 08 — Ansible For Multi-VM Web Cluster

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner to intermediate |
| Estimated time | 60-90 minutes |
| VMs | 4 |
| Minimum VM RAM | 4096 MB |
| SSH ports | 2211, 2212, 2213, 2214 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Ansible installed on your laptop
- Comfort using forwarded Yeast host URLs from `ACCESS.md` for browser-based tools

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you. If not, the lab will tell you when to use a manual SSH tunnel.
- Run Ansible commands from your laptop; Ansible connects to the VMs over SSH.

### Expected Checkpoints

- After `yeast up`, `yeast status` should show the expected VM or VMs as running.
- After the main setup steps, the service, tool, or workflow introduced by the lab should respond to the verification commands.
- After `bash assets/validate.sh`, the script should report all checks passed.
- After `yeast destroy`, the lab should be cleaned up before you start the next one.

### Common Mistakes To Avoid

- Running a VM command on your laptop, or a laptop command inside the VM.
- Ignoring the forwarded port shown by `yeast up` or `yeast status`, or opening a tunnel when the lab already gave you a forwarded host port.
- Skipping validation because the final page or command "looked fine".
- Forgetting to run `yeast destroy` before moving to the next lab.
- Forgetting that Ansible runs from your laptop but manages the VM over SSH.

---

## The Story

One server is running fine. The team wants to scale. Instead of one web node, you need three — so traffic can be distributed and if one goes down, the others keep serving. In front of them, a load balancer that routes requests round-robin.

The manual approach would be: SSH into each web node, configure Nginx, copy the site files, make sure it matches the others. Three servers, three times the work, three chances to make them slightly different.

With Ansible, you write the playbook once. It runs against all three web nodes in parallel, applies identical configuration from a template, and when you change something — a new config option, a new page — you run the playbook again and all three update at the same time.

---

## Before You Start — Understanding The Concepts

### What Is A Load Balancer?

A load balancer distributes incoming traffic across multiple backend servers. When a request arrives, the load balancer picks a backend and forwards the request to it. When the next request arrives, it might go to a different backend.

The benefits:
- **Capacity** — ten servers handle ten times the traffic of one
- **Availability** — if one backend dies, the load balancer stops sending it traffic and the rest keep serving
- **Zero-downtime deploys** — take backends out of rotation one at a time, update them, put them back

Nginx can act as a load balancer using its `upstream` block with multiple server entries.

### What Is Round-Robin Load Balancing?

Round-robin is the simplest load balancing algorithm. Requests go to backend 1, then backend 2, then backend 3, then back to backend 1, and so on. Every backend gets an equal share of traffic.

Other algorithms exist (least connections, IP hash for sticky sessions, weighted round-robin) but round-robin is the right default to start with.

### What Is An Ansible Template?

You saw variables in Lab 07. A template takes variables further — it is a file with placeholders that Ansible fills in when deploying.

The load balancer needs to know the IP addresses of all web nodes. Instead of hardcoding them in the config, the Nginx template pulls them from the Ansible inventory using a loop:

```
upstream webservers {
{% for host in groups['webservers'] %}
    server {{ hostvars[host]['ansible_host'] }}:80 fail_timeout=10s;
{% endfor %}
}
```

When Ansible runs this template, it loops over every host in the `[webservers]` group and inserts their IP addresses. Add a fourth web node to the inventory and re-run the playbook — the load balancer config updates automatically.

### What Is `group_vars`?

In Lab 07 you put variables directly in the playbook. For multi-server setups, Ansible has a better pattern: `group_vars/` — a directory where you put YAML files named after inventory groups.

```
inventory.ini
group_vars/
  webservers.yml    ← variables for the [webservers] group
  loadbalancer.yml  ← variables for the [loadbalancer] group
```

Every host in a group gets the variables defined for that group. This separates configuration from logic cleanly.

### What Is `host_vars`?

Some variables are specific to a single host. For example, each web node needs to know its own name to include in the HTML page it serves. `host_vars/` lets you define variables per host:

```
host_vars/
  web1.yml    ← {node_name: web1}
  web2.yml    ← {node_name: web2}
  web3.yml    ← {node_name: web3}
```

### What Is An Ansible Role?

When a playbook gets large, you split it into roles. A role is a directory with a standard structure:

```
roles/
  webserver/
    tasks/main.yml      ← the tasks
    templates/          ← Jinja2 templates
    vars/main.yml       ← variables
    handlers/main.yml   ← handlers
```

In this lab you will write a simple role structure so you can see how it works before Ansible gets more complex.

---

## What You Are Building

```
Your Laptop
    │
    │  SSH tunnel 9081 → lb port 80
    │  SSH  port 2211 → lb port 22
    │  SSH  port 2212 → web1 port 22
    │  SSH  port 2213 → web2 port 22
    │  SSH  port 2214 → web3 port 22
    ▼
┌──────────────────────────────────────────────────────────────┐
│  Private Network: 192.168.30.0/24                            │
│                                                              │
│  ┌──────────────┐    ┌────────┐ ┌────────┐ ┌────────┐       │
│  │  lb          │───▶│  web1  │ │  web2  │ │  web3  │       │
│  │  .10         │    │  .21   │ │  .22   │ │  .23   │       │
│  │              │──▶─┤        ├─┤        ├─┤        │       │
│  │  Nginx       │    │ Nginx  │ │ Nginx  │ │ Nginx  │       │
│  │  upstream:   │    │ :80    │ │ :80    │ │ :80    │       │
│  │  web1,2,3    │    └────────┘ └────────┘ └────────┘       │
│  └──────────────┘                                            │
└──────────────────────────────────────────────────────────────┘
```

Four VMs, one Ansible playbook that configures all of them.

---

## Starting The Lab

```bash
cd 08-ansible-multi-vm-web-cluster
yeast up
```

Four VMs boot. Check:

```bash
yeast status
```

Expected: lb, web1, web2, web3 all running.

---

## Project Structure

Create this directory layout in the lab folder on your laptop:

```bash
mkdir -p group_vars host_vars roles/webserver/tasks roles/webserver/templates
```

Your lab folder should now look like:

```
08-ansible-multi-vm-web-cluster/
  assets/
    yeast.yaml
    validate.sh
  group_vars/
  host_vars/
  roles/
    webserver/
      tasks/
      templates/
  inventory.ini    ← you'll create this
  site.yml         ← you'll create this
```

---

## Inventory

Create `inventory.ini`:

```ini
[loadbalancer]
lb ansible_host=127.0.0.1 ansible_port=2211 ansible_user=ubuntu

[webservers]
web1 ansible_host=127.0.0.1 ansible_port=2212 ansible_user=ubuntu
web2 ansible_host=127.0.0.1 ansible_port=2213 ansible_user=ubuntu
web3 ansible_host=127.0.0.1 ansible_port=2214 ansible_user=ubuntu

[all:vars]
ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'
```

Note `[all:vars]` — variables that apply to every host in the inventory regardless of group.

Test connectivity to all hosts:

```bash
ansible -i inventory.ini all -m ping
```

All four should return `pong`.

---

## Group Variables

Create `group_vars/webservers.yml`:

```yaml
nginx_doc_root: /var/www/html
site_title: "Cluster Web Node"
```

Create `group_vars/loadbalancer.yml`:

```yaml
lb_upstream_port: 80
```

---

## Host Variables

Create `host_vars/web1.yml`:

```yaml
node_name: web1
node_ip: 192.168.30.21
```

Create `host_vars/web2.yml`:

```yaml
node_name: web2
node_ip: 192.168.30.22
```

Create `host_vars/web3.yml`:

```yaml
node_name: web3
node_ip: 192.168.30.23
```

---

## The Webserver Role

Create `roles/webserver/tasks/main.yml`:

```yaml
---
- name: Install nginx
  ansible.builtin.apt:
    name: nginx
    state: present
    update_cache: true
    cache_valid_time: 3600

- name: Enable and start nginx
  ansible.builtin.service:
    name: nginx
    state: started
    enabled: true

- name: Create document root
  ansible.builtin.file:
    path: "{{ nginx_doc_root }}"
    state: directory
    owner: www-data
    group: www-data
    mode: "0755"

- name: Deploy site page
  ansible.builtin.template:
    src: index.html.j2
    dest: "{{ nginx_doc_root }}/index.html"
    owner: www-data
    group: www-data
    mode: "0644"
  notify: reload nginx
```

Create `roles/webserver/templates/index.html.j2`:

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>{{ site_title }}</title>
</head>
<body>
  <h1>{{ site_title }}</h1>
  <p>Served by: <strong>{{ node_name }}</strong></p>
  <p>Node IP: {{ node_ip }}</p>
  <p>Hostname: {{ ansible_hostname }}</p>
</body>
</html>
```

This template uses `{{ node_name }}` and `{{ node_ip }}` from `host_vars` — each web node gets a unique page telling you which node served the request. That is how you verify load balancing is working: make several requests and see different node names in the response.

Create `roles/webserver/handlers/main.yml`:

```yaml
---
- name: reload nginx
  ansible.builtin.service:
    name: nginx
    state: reloaded
```

---

## The Load Balancer Config Template

Create `roles/loadbalancer/tasks/main.yml`:

```bash
mkdir -p roles/loadbalancer/tasks roles/loadbalancer/templates
```

`roles/loadbalancer/tasks/main.yml`:

```yaml
---
- name: Install nginx
  ansible.builtin.apt:
    name: nginx
    state: present
    update_cache: true
    cache_valid_time: 3600

- name: Deploy load balancer config
  ansible.builtin.template:
    src: lb.conf.j2
    dest: /etc/nginx/sites-available/cluster
    mode: "0644"
  notify: reload nginx

- name: Enable cluster site
  ansible.builtin.file:
    src: /etc/nginx/sites-available/cluster
    dest: /etc/nginx/sites-enabled/cluster
    state: link

- name: Disable default site
  ansible.builtin.file:
    path: /etc/nginx/sites-enabled/default
    state: absent
  notify: reload nginx

- name: Enable and start nginx
  ansible.builtin.service:
    name: nginx
    state: started
    enabled: true
```

`roles/loadbalancer/templates/lb.conf.j2`:

```
upstream webservers {
{% for host in groups['webservers'] %}
    server {{ hostvars[host]['node_ip'] }}:{{ lb_upstream_port }} fail_timeout=10s;
{% endfor %}
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass         http://webservers;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
    }

    access_log /var/log/nginx/lb.access.log;
    error_log  /var/log/nginx/lb.error.log;
}
```

The `{% for %}` loop is Jinja2 templating. It iterates over `groups['webservers']` — the list of hosts in the inventory's `[webservers]` group — and inserts each node's IP into the upstream block. Ansible fills this in when it deploys the template.

Create `roles/loadbalancer/handlers/main.yml`:

```yaml
---
- name: reload nginx
  ansible.builtin.service:
    name: nginx
    state: reloaded
```

---

## The Main Playbook

Create `site.yml`:

```yaml
---
- name: Configure web nodes
  hosts: webservers
  become: true
  roles:
    - webserver

- name: Configure load balancer
  hosts: loadbalancer
  become: true
  roles:
    - loadbalancer
```

Clean. Two plays — one for the web nodes, one for the load balancer. The implementation details are in the roles. This is the correct separation: the playbook describes the intent, the roles describe the implementation.

---

## Running The Playbook

```bash
ansible-playbook -i inventory.ini site.yml
```

Ansible runs the webserver role against web1, web2, web3 in parallel. Then it runs the loadbalancer role against lb.

Watch the `PLAY RECAP` at the end. You should see `changed` on the first run. Run it again:

```bash
ansible-playbook -i inventory.ini site.yml
```

Second run: `changed=0` on all hosts. Idempotent.

---

## Verifying Load Distribution

Make six requests to the load balancer:

```bash
for i in 1 2 3 4 5 6; do
  curl -s http://localhost:9081 | grep "Served by"
done
```

Expected output (order may vary):

```
<p>Served by: <strong>web1</strong></p>
<p>Served by: <strong>web2</strong></p>
<p>Served by: <strong>web3</strong></p>
<p>Served by: <strong>web1</strong></p>
<p>Served by: <strong>web2</strong></p>
<p>Served by: <strong>web3</strong></p>
```

Round-robin: web1, web2, web3, web1, web2, web3. Each request goes to a different backend in rotation.

---

## Simulating A Node Failure

Take web2 out of service:

```bash
yeast ssh web2
sudo systemctl stop nginx
exit
```

Now make several requests to the load balancer:

```bash
for i in 1 2 3 4 5 6; do
  curl -s http://localhost:9081 | grep "Served by"
done
```

You should only see web1 and web3. Nginx detected that web2 is not responding (after `fail_timeout=10s` the first time it fails) and stopped sending traffic to it. The cluster is degraded — you lost capacity — but it is still serving.

Check the lb error log:

```bash
yeast ssh lb
sudo tail -10 /var/log/nginx/lb.error.log
```

You will see "connect() failed" errors for web2's IP. Nginx logged every failed upstream connection attempt.

Bring web2 back:

```bash
yeast ssh web2
sudo systemctl start nginx
exit
```

Wait a few seconds, then make more requests — web2 is back in rotation.

---

## Adding A Fourth Web Node

The power of the template-based approach: add `web4` to the inventory and re-run the playbook.

In `inventory.ini`, add under `[webservers]`:

```ini
web4 ansible_host=127.0.0.1 ansible_port=2215 ansible_user=ubuntu
```

Create `host_vars/web4.yml`:

```yaml
node_name: web4
node_ip: 192.168.30.24
```

Running the playbook now would configure web4 and update the load balancer config to include it — all from a single command. The load balancer template re-renders with the new node in the `upstream` block automatically.

(We skip actually booting web4 since it is not in the `yeast.yaml`, but the principle is demonstrated.)

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

In Lab 08 — Ansible For Multi-VM Web Cluster, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- Load balancing: what it is, why it matters, round-robin distribution
- Ansible inventory groups: `[loadbalancer]`, `[webservers]`, `[all:vars]`
- `group_vars/` and `host_vars/`: separating config from logic
- Ansible templates with Jinja2: loops, variable substitution, `groups[]`, `hostvars[]`
- Ansible roles: `tasks/main.yml`, `templates/`, `handlers/main.yml`
- Running a playbook against multiple hosts in parallel
- Failure handling in Nginx upstream: `fail_timeout`, automatic backend removal
- How adding a host to inventory + re-running the playbook is the entire scaling operation

---

## What Is Next

**Lab 09 — Secrets And Configuration Management**

Every lab so far has had credentials hardcoded or in plaintext: database passwords in app code, API keys in config files. Lab 09 teaches you how to separate secrets from configuration, how to avoid leaking credentials into version control, and the basics of tools like Ansible Vault and environment-based secret injection.
