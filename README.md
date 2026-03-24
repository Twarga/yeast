# Yeast 🍞

> **Local Virtual Machines for the Cloud Native Era.**

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21-cyan.svg)
![Platform](https://img.shields.io/badge/platform-linux-lightgrey.svg)

**Yeast** is a modern, lightweight local VM orchestrator. It is designed to be a faster, simpler alternative to Vagrant for Linux users.

Built on top of **KVM** and **QEMU**, Yeast provides:
- 🚀 **Instant Boot:** Uses Copy-on-Write (CoW) disks for milliseconds creation time.
- 📦 **Zero Bloat:** No 2GB box downloads per project. Base images are shared globally.
- 🔧 **Cloud-Init:** Native support for modern cloud configuration (user-data).
- 🧬 **Declarative Config:** Simple `yeast.yaml` defines your environment.
- 🐹 **Single Binary:** Written in Go, no Ruby/Python dependencies.

## 📋 Prerequisites

Yeast runs on **Linux** and requires hardware virtualization.

1.  **KVM & QEMU:**
    ```bash
    # Ubuntu/Debian
    sudo apt install qemu-system-x86 qemu-utils genisoimage

    # Fedora/RHEL
    sudo dnf install qemu-system-x86 qemu-img genisoimage

    # Arch Linux
    sudo pacman -S qemu-base cdrtools
    ```

2.  **User Permissions:**
    Ensure your user is in the `kvm` group (to run VMs without sudo):
    ```bash
    sudo usermod -aG kvm $USER
    # You may need to log out and log back in.
    ```

## 🚀 Installation

### One-command installer

If you already have the repository locally, run:

```bash
bash install.sh
```

The installer:
- detects the Linux package manager
- installs Yeast dependencies
- installs Go for the source-build path
- builds and installs `yeast` to `/usr/local/bin`
- creates `~/.yeast/cache/`
- generates an SSH key if needed
- adds the user to the `kvm` group when possible

You can override the source repo and branch:

```bash
YEAST_REPO_URL=https://github.com/Twarga/yeast.git YEAST_REF=dev bash install.sh
```

### From Source

```bash
git clone https://github.com/Twarga/yeast.git
cd yeast
go build -o yeast ./cmd/yeast
sudo mv yeast /usr/local/bin/
```

## 🏁 Quick Start

### 1. Run Environment Checks
Before creating VMs, verify host prerequisites:

```bash
yeast doctor
```

If blockers are found, follow the printed `fix:` steps and re-run the command.

### 2. Initialize a Project
Create a new directory and initialize a Yeast project.

```bash
mkdir my-project
cd my-project
yeast init
```

This creates a `yeast.yaml` file:
```yaml
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 1
    user: yeast
    sudo: none
```

You can also generate a starter config with your preferred values:

```bash
yeast init \
  --name api \
  --image ubuntu-24.04 \
  --memory 2048 \
  --cpus 2 \
  --user operator \
  --sudo password
```

### 3. Download Trusted Base Image
List supported images first:

```bash
yeast pull --list
```

Then use Yeast's trusted manifest downloader (URL + pinned SHA256 verification):

```bash
yeast pull ubuntu-22.04
```

Supported images:
- `ubuntu-22.04`
- `ubuntu-24.04`

### 4. Start the VMs
```bash
yeast up
```
You will see output indicating the disk creation, cloud-init generation, and QEMU startup.

### 5. Check Status
```bash
yeast status
```
Output:
```text
NAME    STATUS    PID     IP           SSH PORT
web     running   12345   127.0.0.1    45678
```

Machine-readable mode:

```bash
yeast status --json
```

### 6. SSH Connect
```bash
yeast ssh web
```
You are now inside the VM!

### 7. Stop Everything
```bash
yeast down
```

## ⚙️ Configuration (`yeast.yaml`)

```yaml
version: 1
instances:
  - name: db-server
    image: ubuntu-22.04   # Matches filename in ~/.yeast/cache/[image].img
    memory: 2048          # RAM in MB
    cpus: 2               # vCPU cores
    user: yeast           # Optional, default: yeast
    sudo: none            # Optional: none | password | nopasswd (default: none)
    
    # Optional: User Data for Cloud-Init
    user_data: |
      #cloud-config
      packages:
        - nginx
        - git
      runcmd:
        - systemctl enable --now nginx
```

Starter-config shortcuts:
- `yeast init --name api --image ubuntu-24.04 --memory 2048 --cpus 2`
- `yeast init --user operator --sudo password`

Current config limitation:
- configurable disk size is not yet supported in `yeast.yaml`

Security defaults:
- `sudo: none` is the default for least privilege.
- Use `sudo: password` to require password-protected sudo.
- Use `sudo: nopasswd` only when you explicitly accept that risk.

## 🌐 Networking Modes

Yeast supports explicit networking modes for `up` and `restart`:

- `user` (default): user-mode NAT with SSH host port forwarding
- `private`: restricted user-mode network (`restrict=on`) with SSH host port forwarding
- `bridge`: host bridge attachment plus a restricted management NIC for SSH forwarding

Examples:

```bash
yeast up --network-mode user
yeast up --network-mode private
yeast up --network-mode bridge --bridge br0
yeast restart web --network-mode bridge --bridge br0
```

Design tradeoffs and safety notes:
- `docs/NETWORKING_MODES.md`

## 🧾 JSON Output Contracts

For automation and scripting, major commands support `--json` while keeping default human-readable output unchanged.

Examples:

```bash
yeast up --json
yeast status --json
yeast down --json
yeast halt web --json
yeast restart web --json
yeast destroy web --json
```

Contract details and schema versions:
- `docs/OUTPUT_CONTRACTS.md`

## 📈 Performance Benchmarks

Repeatable benchmark harness:

```bash
scripts/benchmark.sh --iterations 7 --output benchmarks/latest.json
```

Methodology:
- `docs/PERFORMANCE_BENCHMARKS.md`
- Cold-start iteration loop (`up` -> collect metrics -> `down`)
- Deterministic `fake-tools` mode (CI-friendly)

Baseline (fake-tools mode, 7 iterations, captured 2026-03-05):

| Metric | Min | P50 | P95 | Avg | Max |
|---|---:|---:|---:|---:|---:|
| Startup latency (ms) | 518 | 523 | 524 | 522.00 | 524 |
| VM process RSS (KB) | 8728 | 8796 | 10872 | 9668.57 | 10872 |
| Overlay disk bytes | 10 | 10 | 10 | 10.00 | 10 |
| Instance dir bytes | 300 | 300 | 300 | 300.00 | 300 |

Baseline artifact:
- `benchmarks/latest.json`

## 📂 Architecture

Yeast uses a standard layout:
- `~/.yeast/cache/`: Stores read-only base images.
- `~/.yeast/instances/[name]/`: Stores the VM state (CoW disk, logs, seed ISO).
- `yeast.state`: Local JSON file in your project directory tracking PIDs.

## 🪵 Logging and Troubleshooting

VM runtime logs:
- Active log: `~/.yeast/instances/<name>/vm.log`
- Rotated history: `~/.yeast/instances/<name>/vm.<timestamp>.<unixnano>.log`
- Archive retention: default `5` per instance (configure with `YEAST_VM_LOG_RETENTION`)

Docs:
- `docs/LOGGING.md` (format, levels, fields, file naming, retention policy)
- `docs/RUNBOOK_VM_FAILURES.md` (step-by-step debug runbook)

## 🔒 Image Trust Model

Yeast ships a built-in trusted image manifest. Each supported image contains:
- A fixed source URL
- A pinned SHA256 checksum

When you run `yeast pull <image>`:
1. Yeast downloads to a temporary file in `~/.yeast/cache/`
2. It computes SHA256 while downloading
3. It verifies checksum against the trusted manifest
4. It only moves the file into place if verification succeeds

If checksum verification fails, Yeast fails closed and removes the partial file.

Manifest source of truth:
- Ubuntu official cloud image releases
- Corresponding `SHA256SUMS` from `cloud-images.ubuntu.com`

## 🤝 Contributing

Contributions are welcome.

See:
- `CONTRIBUTING.md` for contribution flow, coding/testing standards, release notes format, and versioning policy.
- `CHANGELOG.md` for the release note structure and current `Unreleased` entries.
- `docs/SECURITY_STATIC_ANALYSIS.md` for Go lint/security tooling, severity policy, and suppression standards.

## 📄 License

Distributed under the **MIT License**. See `LICENSE` for more information.

---
*Built with ❤️ and Go.*
