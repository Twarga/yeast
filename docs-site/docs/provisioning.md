---
title: Provisioning
description: Install packages, copy files, and configure VMs automatically
---

# Provisioning

Provisioning is the process of configuring a VM after it boots. Yeast handles provisioning automatically — you declare what you want, and Yeast makes it happen.

## Two Provisioning Phases

### Phase 1: Cloud-Init (First Boot Only)

Cloud-init runs automatically on the VM's first boot. It handles:

- Creating the configured user (default: `yeast`)
- Adding your SSH public key
- Setting the hostname
- Configuring static IP addresses on lab networks
- Setting up sudo policy
- Installing declared packages (via `package_update` and `packages`)

**You don't need to do anything** — cloud-init is generated from `yeast.yaml` automatically.

### Phase 2: Post-Boot Provisioning (Every Boot)

After cloud-init finishes and SSH is ready, Yeast runs the `provision` section from `yeast.yaml`.

This phase handles:
- Installing packages
- Copying files from host to guest
- Running shell commands

## Declaring Provisioning

Add a `provision` section to any instance in `yeast.yaml`:

```yaml
instances:
  - name: web
    image: ubuntu-24.04
    provision:
      packages:
        - nginx
        - curl
        - git
      files:
        - source: ./files/index.html
          destination: /var/www/html/index.html
          permissions: "0644"
        - source: ./files/Caddyfile
          destination: /etc/caddy/Caddyfile
          permissions: "0644"
      shell:
        - sudo systemctl enable nginx
        - sudo systemctl start nginx
        - sudo ufw allow 'Nginx Full'
```

## Packages

Install packages with the system package manager:

```yaml
provision:
  packages:
    - nginx
    - postgresql
    - nodejs
    - npm
```

**How it works:**
- Yeast runs `apt-get update` first
- Then `apt-get install -y <packages>`
- Uses the guest's package manager (apt on Ubuntu/Debian)

**Best practices:**
- List packages in dependency order
- Don't assume packages exist on the base image
- Keep the list focused (install additional packages in shell steps if needed)

## Files

Copy files from the host to the guest:

```yaml
provision:
  files:
    - source: ./site/index.html
      destination: /var/www/html/index.html
      permissions: "0644"
    - source: ./scripts/setup.sh
      destination: /tmp/setup.sh
      permissions: "0755"
    - source: ./config/nginx.conf
      destination: /etc/nginx/nginx.conf
      permissions: "0644"
```

### File fields

| Field | Required | Description |
|---|---|---|
| `source` | Yes | Host path, relative to project root |
| `destination` | Yes | Guest path, absolute |
| `permissions` | No | Octal permissions (default: "0644") |

**Important:**
- `source` paths are resolved relative to the project directory (where `yeast.yaml` lives)
- `destination` paths must be absolute (start with `/`)
- Parent directories are created automatically
- Files are owned by the configured user

**Common permission values:**
- `"0644"` — Readable by everyone, writable by owner (config files)
- `"0755"` — Executable, readable by everyone (scripts, binaries)
- `"0600"` — Private to owner (secrets, keys)

## Shell Commands

Run shell commands during provisioning:

```yaml
provision:
  shell:
    - sudo systemctl enable nginx
    - sudo systemctl start nginx
    - echo "Server ready" > /var/www/html/status.txt
    - curl -fsSL https://example.com/setup.sh | sudo bash
```

**How it works:**
- Commands run as the configured user (default: `yeast`)
- Use `sudo` for commands requiring root
- Commands run in order
- If any command fails, provisioning stops and reports the error

**Best practices:**
- Make commands idempotent (safe to run multiple times)
- Use absolute paths
- Prefer `systemctl enable` + `systemctl start` over `service`
- Test commands in the VM with `yeast exec` before adding to yeast.yaml

## Complete Example

A real-world example: web server with Caddy:

```yaml
version: 1

instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    ssh_port: 2222
    ports:
      - host_port: 8080
        guest_port: 80
    provision:
      packages:
        - caddy
        - curl
      files:
        - source: ./site/index.html
          destination: /var/www/html/index.html
          permissions: "0644"
        - source: ./Caddyfile
          destination: /etc/caddy/Caddyfile
          permissions: "0644"
      shell:
        - sudo systemctl enable caddy
        - sudo systemctl start caddy
        - sleep 2
        - curl -fsS http://localhost || echo "Service not ready yet"
```

Project structure:

```
my-lab/
├── yeast.yaml
├── site/
│   └── index.html
└── Caddyfile
```

## Provisioning Behavior

### First Boot

On first boot:
1. Cloud-init creates user, SSH key, hostname, network
2. Post-boot provisioning runs (packages, files, shell)

### Subsequent Boots

On subsequent boots:
1. Yeast checks if `yeast.yaml` has changed since last boot
2. If changed: re-run post-boot provisioning
3. If not changed: skip provisioning

**Important:** Cloud-init only runs on first boot. If you add users or SSH keys to `yeast.yaml` after first boot, they won't be applied automatically.

### Force Reprovisioning

Force re-run of provisioning without changing `yeast.yaml`:

```bash
yeast up --reprovision
```

Or provision a single running instance:

```bash
yeast provision web
```

## Debugging Provisioning

### Check Provisioning Logs

```bash
yeast logs web
```

Look for lines starting with `[provision]`.

### SSH and Inspect

```bash
yeast ssh web

# Check if packages installed
dpkg -l | grep nginx

# Check file permissions
ls -la /var/www/html/

# Check service status
systemctl status nginx

# Check provisioning script results
cat /var/log/cloud-init-output.log

# Exit
exit
```

### Test Commands First

Before adding to `yeast.yaml`, test commands interactively:

```bash
yeast ssh web
sudo apt update && sudo apt install -y nginx
systemctl status nginx
exit
```

Then add verified commands to `yeast.yaml`.

## Provisioning Idempotency

Make your provisioning safe to run multiple times:

```yaml
# Good — idempotent
shell:
  - sudo systemctl enable nginx || true
  - sudo systemctl start nginx || true

# Better — check before acting
shell:
  - test -f /etc/nginx/nginx.conf || sudo cp /tmp/nginx.conf /etc/nginx/
  - sudo systemctl is-active nginx || sudo systemctl start nginx
```

## Troubleshooting

### "package not found"

The package name might differ between distributions. Ubuntu/Debian use `apt` package names.

```bash
# Find correct package name
apt search <keyword>
```

### "permission denied" on file copy

Use `permissions: "0755"` for scripts and `sudo` in shell commands for system directories.

### "command not found"

The command might not be in the default PATH. Use absolute paths:

```yaml
shell:
  - /usr/bin/systemctl enable nginx
  - /usr/sbin/nginx -t
```

### Slow provisioning

Package installation can be slow. For faster iteration:
1. Start with a minimal package list
2. Test interactively with `yeast ssh`
3. Once working, add to `yeast.yaml`
4. Take a snapshot: `yeast snapshot web provisioned`

## Next Steps

- [Configuration](./configuration) — Full yeast.yaml reference
- [Snapshots](./snapshots) — Save and restore VM state
- [Troubleshooting](./troubleshooting) — Common issues and fixes
