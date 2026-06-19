# Lab 12 — Container Build, Scan, And Hardening

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner to intermediate |
| Estimated time | 60-90 minutes |
| VMs | 1 |
| Minimum VM RAM | 2048 MB |
| SSH ports | 2218 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic understanding that Docker commands run inside the VM unless stated otherwise

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether the lab asked you to open an SSH tunnel first.
- Run Docker commands inside the VM unless the lab explicitly says otherwise.

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
- Confusing laptop `localhost`, VM `localhost`, and container `localhost`.

---

## The Story

The Dockerfile from Lab 11 worked. But "works" is not the same as "safe" or "production-ready." That image was built on `python:3.11-slim`, ran as root inside the container, contained more packages than the app needed, and was never scanned for known vulnerabilities.

In real teams, container images go through a build pipeline before they ever reach production. They are scanned for CVEs (Common Vulnerabilities and Exposures). They are rebuilt on a schedule to pick up base image patches. They are tagged precisely — never `latest` in production. They run as non-root users.

This lab teaches you how to build container images correctly: smaller, safer, with a known vulnerability profile.

---

## Before You Start — Understanding The Concepts

### Why Container Images Need To Be Small

A large image is not just slow to pull — it is a larger attack surface. Every package installed in an image is a potential vulnerability. A base image like `ubuntu:22.04` ships with hundreds of packages your app does not use. Each one might have a CVE someday.

Smaller images:
- Pull faster over the network
- Have fewer packages = fewer CVEs
- Are easier to reason about (you know what is in them)
- Use less disk space on the host

### What Is A CVE?

CVE stands for Common Vulnerabilities and Exposures. It is a public database of known security vulnerabilities in software, each with a unique ID (e.g., CVE-2024-12345). Vulnerabilities are rated by severity: CRITICAL, HIGH, MEDIUM, LOW, UNKNOWN.

When a CVE is published for a library in your container image, your image is vulnerable until you rebuild with a patched version.

### What Is Trivy?

Trivy is an open-source vulnerability scanner for container images. It scans the packages in an image against CVE databases and reports what it finds. It also scans filesystems, git repos, Terraform configs, and more.

You point it at an image: `trivy image myapp:latest`. It tells you which packages have known vulnerabilities, how severe, and often what version fixes the problem.

### What Is A Multi-Stage Build?

A multi-stage Dockerfile has multiple `FROM` statements. Each stage is a separate build environment. The final stage copies only what it needs from previous stages.

Use case: build your app in a stage that has the compiler and build tools. Copy only the compiled binary into a minimal final stage. The final image has no compiler, no build tools — just the binary and its runtime dependencies.

```dockerfile
# Stage 1: build
FROM python:3.11 AS builder
RUN pip install ...

# Stage 2: runtime
FROM python:3.11-slim
COPY --from=builder /usr/local/lib/python3.11/site-packages/ /usr/local/lib/python3.11/site-packages/
```

### Why Run As Non-Root?

By default, processes inside a container run as root (UID 0). If the container is compromised — say, an attacker finds a code execution vulnerability in your app — they have root inside the container.

With kernel vulnerabilities or container escape bugs, root inside the container can sometimes become root on the host. Running as a non-root user limits the blast radius.

The fix is simple: create a dedicated user in the Dockerfile and switch to it.

### What Is `latest` And Why Not Use It In Production?

`latest` is the default tag when you do not specify one. `docker pull nginx` pulls `nginx:latest`. The problem: `latest` changes. If the upstream maintainer pushes a new version tagged `latest`, your next pull gets different code. This breaks reproducibility.

In production, always use specific version tags: `nginx:1.25.3`, `python:3.11.7-slim`. Pin to the exact version you tested.

### What Is An SBOM?

SBOM stands for Software Bill of Materials. It is a machine-readable list of every component in a piece of software — every library, version, and license. For a container image, it lists every package installed.

SBOMs are used for compliance, vulnerability tracking, and license auditing. Trivy can generate SBOMs. Some regulated industries require them.

---

## What You Are Building

You will build two versions of the same app image:
- `myapp:fat` — the naive Dockerfile
- `myapp:slim` — the hardened Dockerfile

Then scan both with Trivy and compare the vulnerability counts.

---

## Starting The Lab

```bash
cd 12-container-build-scan-hardening
yeast up
yeast ssh builder
newgrp docker
```

---

## Step 1 — The Fat Image (What Not To Do)

Create the app:

```bash
mkdir -p /home/ubuntu/app && cd /home/ubuntu/app

cat > app.py << 'EOF'
#!/usr/bin/env python3
import os
import json
from http.server import HTTPServer, BaseHTTPRequestHandler

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({"message": "hello", "user": os.getenv("USER", "unknown")}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass

if __name__ == "__main__":
    HTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
EOF
```

The fat Dockerfile — common beginner mistakes:

```bash
cat > Dockerfile.fat << 'EOF'
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    curl \
    wget \
    git \
    vim \
    build-essential

WORKDIR /app
COPY app.py .

EXPOSE 8000
CMD ["python3", "app.py"]
EOF
```

Problems in this Dockerfile:
- `ubuntu:22.04` as base — hundreds of packages you do not need
- Installing `curl`, `wget`, `git`, `vim`, `build-essential` — not needed at runtime
- Running as root (no `USER` directive)
- No specific version pins

Build it:

```bash
docker build -t myapp:fat -f Dockerfile.fat .
docker images myapp:fat
```

Note the image size — it will be several hundred MB.

Check what user it runs as:

```bash
docker run --rm myapp:fat id
```

```
uid=0(root) gid=0(root) groups=0(root)
```

Root. Not good.

---

## Step 2 — Scan The Fat Image

```bash
trivy image myapp:fat | tee /home/ubuntu/scan-results.txt
```

Trivy downloads its vulnerability database (first run takes a minute) then scans the image. You will see a table of vulnerabilities grouped by package, showing CVE IDs, severities, and fixed versions.

Count the critical and high severity issues:

```bash
grep -E "CRITICAL|HIGH" /home/ubuntu/scan-results.txt | wc -l
```

It will be a significant number. These are all known vulnerabilities in packages your app does not even use.

---

## Step 3 — The Slim Image (The Right Way)

```bash
cat > Dockerfile.slim << 'EOF'
FROM python:3.11-slim AS base

# Create a non-root user
RUN groupadd --gid 1000 appuser && \
    useradd --uid 1000 --gid 1000 --no-create-home appuser

WORKDIR /app

# Install only what the app needs
RUN pip install --no-cache-dir psycopg2-binary

# Copy app code
COPY --chown=appuser:appuser app.py .

# Switch to non-root user
USER appuser

EXPOSE 8000

CMD ["python3", "app.py"]
EOF
```

What changed:
- `python:3.11-slim` base — Python with minimal OS packages
- No extra tools (`curl`, `git`, `vim`, etc.) — not needed at runtime
- `groupadd` / `useradd` — create a dedicated user
- `--chown=appuser:appuser` on `COPY` — app code owned by the app user, not root
- `USER appuser` — container process runs as UID 1000, not root
- `--no-cache-dir` on pip — smaller image, no pip cache stored in the layer

Build it:

```bash
docker build -t myapp:slim -f Dockerfile.slim .
docker images myapp:fat myapp:slim
```

Compare the sizes. The slim image should be dramatically smaller.

Check the user:

```bash
docker run --rm myapp:slim id
```

```
uid=1000(appuser) gid=1000(appuser) groups=1000(appuser)
```

Not root.

---

## Step 4 — Scan The Slim Image

```bash
trivy image myapp:slim | tee -a /home/ubuntu/scan-results.txt
```

Compare vulnerability counts:

```bash
echo "=== Fat ===" && grep -E "CRITICAL|HIGH" /home/ubuntu/scan-results.txt | head -20
echo "=== Slim summary ===" && grep -E "Total|CRITICAL|HIGH" /home/ubuntu/scan-results.txt | tail -5
```

The slim image should have significantly fewer vulnerabilities — because it has fewer packages to be vulnerable.

---

## Step 5 — Tagging Correctly

Never use `latest` in production. Use semantic version tags:

```bash
# Tag the slim image with a real version
docker tag myapp:slim myapp:1.0.0

# List all tags
docker images myapp
```

Tagging conventions used in practice:
- `myapp:1.0.0` — exact release version, immutable
- `myapp:1.0` — tracks the latest 1.0.x patch
- `myapp:1` — tracks the latest 1.x.x minor
- `myapp:latest` — only for local development convenience, never production

When you deploy `myapp:1.0.0`, you know exactly what you deployed. If there is a bug, you can trace it to that exact image.

---

## Step 6 — Layer Caching And Build Efficiency

Docker builds images layer by layer. Each instruction in the Dockerfile creates a layer. Layers are cached — if nothing changed since last build, Docker reuses the cached layer instead of rebuilding it.

This matters for build speed. The wrong order:

```dockerfile
# BAD: copying code before installing dependencies
COPY . .
RUN pip install -r requirements.txt
```

Every code change invalidates the `COPY` layer and forces `pip install` to run again — slow.

The right order: **install dependencies before copying code**, because dependencies change rarely:

```dockerfile
# GOOD: dependencies installed first
COPY requirements.txt .
RUN pip install -r requirements.txt

# Code changes only invalidate layers from here down
COPY . .
```

Test this: build `myapp:slim` twice. The second build should say `CACHED` for every layer — milliseconds instead of seconds.

```bash
docker build -t myapp:slim -f Dockerfile.slim .
# Should complete almost instantly — all layers cached
```

---

## Step 7 — Generate An SBOM

```bash
trivy image --format spdx-json --output /home/ubuntu/sbom.json myapp:slim
cat /home/ubuntu/sbom.json | python3 -m json.tool | head -30
```

The SBOM lists every package in the image with its version and license. This is what security and compliance teams ask for when auditing software.

---

## Step 8 — What `latest` Actually Means

Demonstrate why `latest` is dangerous in CI pipelines:

```bash
# Pull the current latest
docker pull python:3.11-slim

# Get the image digest — the cryptographic hash of the exact image
docker inspect python:3.11-slim | grep -i digest

# Six months from now, python:3.11-slim might point to a different digest
# If you use the digest directly, you are pinned:
# FROM python@sha256:<exact-hash>
```

For maximum reproducibility, pin to a digest instead of a tag. This is considered best practice for production base images.

---

## The Dockerfile Best Practices Summary

| Practice | Why |
|---|---|
| Use `slim` or `alpine` base images | Fewer packages = fewer CVEs, smaller size |
| Create a non-root user | Limit blast radius if compromised |
| `COPY --chown=user:user` | Correct file ownership from the start |
| `USER appuser` before `CMD` | Process runs as non-root |
| Install deps before copying code | Better layer cache utilization |
| `--no-cache-dir` with pip | Smaller image layers |
| Pin versions in `FROM` | Reproducible builds |
| No tools you do not need at runtime | Fewer packages = smaller attack surface |
| Scan with Trivy before shipping | Know your vulnerability profile |

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
exit
yeast destroy
```

---

## Quick Recap

In Lab 12 — Container Build, Scan, And Hardening, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- Why image size matters: fewer packages = smaller attack surface
- CVEs: what they are, how they are rated (CRITICAL/HIGH/MEDIUM/LOW)
- Trivy: scanning images for known vulnerabilities, comparing before and after hardening
- Non-root containers: creating a user in the Dockerfile and switching to it
- Multi-stage builds: concept and use case
- Dockerfile best practices: layer ordering for cache efficiency, `--no-cache-dir`
- Version tagging: why `latest` is dangerous in production
- SBOMs: what they are and how to generate one
- The measurable difference between a naive and a hardened image

---

## What Is Next

**Lab 13 — CI With GitHub Actions**

You have been building and running things locally. Lab 13 connects your work to a real CI pipeline. Every push to a GitHub repository automatically runs tests, builds your container image, scans it, and reports the result. You will write your first GitHub Actions workflow from scratch and understand every line of it.
