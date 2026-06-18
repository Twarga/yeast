# Lab 15 — Private Container Registry

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 2 |
| Minimum VM RAM | 3072 MB |
| SSH ports | 2221, 2222 |
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

Your CI pipeline builds container images. You scan them. They pass. But then what? Right now they live only on the build machine. When you want to deploy to a different server, you need a way to share the image. Pushing to Docker Hub is one option, but not every team wants their images public, and Docker Hub has rate limits.

A private container registry is the standard solution. It is a server that stores your container images — accessible only to authenticated clients. Your CI pushes to it. Your deploy servers pull from it. Every image version is stored indefinitely. Rolling back is as simple as pulling an older tag.

In this lab you run your own registry on a Yeast VM using the official `registry:2` image, push a tagged image to it, and deploy from it to a second VM.

---

## Before You Start — Understanding The Concepts

### What Is A Container Registry?

A container registry is an HTTP server that implements the OCI Distribution Specification — a standard API for storing and serving container images. `docker push` sends images to a registry. `docker pull` retrieves them.

Docker Hub is the default public registry. But the Docker registry is also open-source software (`registry:2`) that you can run yourself. This is exactly what you will do today.

### What Is The OCI Distribution Spec?

OCI stands for Open Container Initiative. It is an industry standards body that defines:
- The container image format (OCI Image Spec)
- The container runtime behavior (OCI Runtime Spec)
- The image distribution protocol (OCI Distribution Spec — the registry API)

Any tool that implements these specs interoperates. `docker push`, `podman push`, `crane push` — they all speak the same protocol to any compliant registry.

### What Is Image Promotion?

Image promotion is the practice of moving an image through environments (dev → staging → production) without rebuilding it. You build once in CI, scan it, tag it with the version, and push it to the registry. Staging pulls that tag and deploys it. After testing, production pulls the same tag. The same bytes run everywhere — no "it worked in staging" surprises.

### What Is A Registry Tag?

A tag is a mutable pointer to an immutable image digest. `myapp:1.0.0` points to a specific image. `myapp:latest` might point to the same image today but a different one tomorrow.

The digest is the immutable identifier: `myapp@sha256:abc123...`. Production deployments should reference digests, not tags, for true immutability.

### What Is An Insecure Registry?

By default Docker only allows pushes/pulls to registries over HTTPS. Our local registry runs on HTTP (no TLS). To use it, Docker clients must be configured to allow "insecure" registries for that address — a per-host opt-in.

In production you would put a TLS-terminating proxy (Nginx or Caddy with Let's Encrypt) in front of the registry. For this lab we use insecure mode for simplicity.

---

## What You Are Building

```
Your Laptop
    │  SSH 2221 → registry VM port 22
    │  SSH 2222 → deployer VM port 22
    ▼
┌──────────────────────────────────────────────────────┐
│  Private Network: 192.168.40.0/24                    │
│                                                      │
│  ┌──────────────────────────┐                        │
│  │  registry (192.168.40.10)│                        │
│  │  Docker registry:2       │                        │
│  │  listening :5000         │                        │
│  └──────────────────────────┘                        │
│            ↑ push              ↓ pull                 │
│  ┌─────────────────┐  ┌───────────────────────────┐  │
│  │  build (laptop) │  │  deployer (192.168.40.20) │  │
│  └─────────────────┘  └───────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 15-private-container-registry
yeast up
```

---

## Step 1 — Start The Registry

SSH into the registry VM:

```bash
yeast ssh registry
newgrp docker
```

Start the registry using the official `registry:2` image:

```bash
docker run -d \
  --name registry \
  --restart always \
  -p 5000:5000 \
  -v registry-data:/var/lib/registry \
  registry:2
```

- `-p 5000:5000` — map the registry's port 5000 to the host's port 5000
- `-v registry-data:/var/lib/registry` — persist image data to a named volume
- `--restart always` — restart on failure or reboot

Verify it is running:

```bash
docker ps
curl http://localhost:5000/v2/
```

Expected: `{}` — the registry is running and responding to the API.

Exit the registry VM:

```bash
exit
```

---

## Step 2 — Configure Docker To Use The Insecure Registry

Both the build machine (your laptop) and the deployer VM need to know that `192.168.40.10:5000` is allowed as an insecure registry.

**On your laptop:**

```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json << 'EOF'
{
  "insecure-registries": ["192.168.40.10:5000"]
}
EOF
sudo systemctl restart docker
```

**On the deployer VM:**

```bash
yeast ssh deployer
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json << 'EOF'
{
  "insecure-registries": ["192.168.40.10:5000"]
}
EOF
sudo systemctl restart docker
exit
```

---

## Step 3 — Build, Tag, And Push An Image

Create a simple app on your laptop:

```bash
mkdir -p /tmp/registry-lab && cd /tmp/registry-lab

cat > app.py << 'EOF'
from http.server import HTTPServer, BaseHTTPRequestHandler
import json, os

VERSION = os.getenv("APP_VERSION", "unknown")

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        b = json.dumps({"version": VERSION, "status": "ok"}).encode()
        self.send_response(200)
        self.send_header("Content-Type","application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)
    def log_message(self, *a): pass

HTTPServer(("0.0.0.0", 8080), H).serve_forever()
EOF

cat > Dockerfile << 'EOF'
FROM python:3.11-slim
WORKDIR /app
COPY app.py .
EXPOSE 8080
CMD ["python3", "app.py"]
EOF
```

Build version 1.0.0:

```bash
docker build -t myapp:1.0.0 --build-arg APP_VERSION=1.0.0 .
```

Tag it with the registry address:

```bash
docker tag myapp:1.0.0 192.168.40.10:5000/myapp:1.0.0
docker tag myapp:1.0.0 192.168.40.10:5000/myapp:latest
```

Tagging with the registry address tells Docker where to push. `192.168.40.10:5000/myapp:1.0.0` means: registry at `192.168.40.10:5000`, image name `myapp`, tag `1.0.0`.

Push:

```bash
docker push 192.168.40.10:5000/myapp:1.0.0
docker push 192.168.40.10:5000/myapp:latest
```

Verify it arrived in the registry:

```bash
curl http://192.168.40.10:5000/v2/myapp/tags/list
```

Expected: `{"name":"myapp","tags":["1.0.0","latest"]}`

---

## Step 4 — Deploy From The Registry

SSH into the deployer VM:

```bash
yeast ssh deployer
newgrp docker
```

Pull the image from the registry:

```bash
docker pull 192.168.40.10:5000/myapp:1.0.0
```

Run it:

```bash
docker run -d \
  --name app \
  --restart unless-stopped \
  -p 8080:8080 \
  -e APP_VERSION=1.0.0 \
  192.168.40.10:5000/myapp:1.0.0
```

Verify:

```bash
curl http://localhost:8080
```

Expected: `{"version": "1.0.0", "status": "ok"}`

The deployer pulled the image from your private registry — not Docker Hub.

---

## Step 5 — Image Promotion: Deploy A New Version

Build version 2.0.0 on your laptop:

```bash
# Update the app with a new message
cat > /tmp/registry-lab/app.py << 'EOF'
from http.server import HTTPServer, BaseHTTPRequestHandler
import json, os

VERSION = os.getenv("APP_VERSION", "unknown")

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        b = json.dumps({"version": VERSION, "status": "ok", "feature": "new-in-v2"}).encode()
        self.send_response(200)
        self.send_header("Content-Type","application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)
    def log_message(self, *a): pass

HTTPServer(("0.0.0.0", 8080), H).serve_forever()
EOF

cd /tmp/registry-lab
docker build -t myapp:2.0.0 .
docker tag myapp:2.0.0 192.168.40.10:5000/myapp:2.0.0
docker tag myapp:2.0.0 192.168.40.10:5000/myapp:latest
docker push 192.168.40.10:5000/myapp:2.0.0
docker push 192.168.40.10:5000/myapp:latest
```

Verify both versions are in the registry:

```bash
curl http://192.168.40.10:5000/v2/myapp/tags/list
```

Expected: `{"name":"myapp","tags":["1.0.0","2.0.0","latest"]}`

Deploy 2.0.0 on the deployer:

```bash
yeast ssh deployer
docker pull 192.168.40.10:5000/myapp:2.0.0
docker stop app && docker rm app
docker run -d --name app --restart unless-stopped -p 8080:8080 \
  -e APP_VERSION=2.0.0 \
  192.168.40.10:5000/myapp:2.0.0
curl http://localhost:8080
```

Expected: `{"version": "2.0.0", "feature": "new-in-v2", ...}`

---

## Step 6 — Rollback

The new version has a bug. Roll back to 1.0.0:

```bash
# Still on deployer VM
docker stop app && docker rm app
docker run -d --name app --restart unless-stopped -p 8080:8080 \
  -e APP_VERSION=1.0.0 \
  192.168.40.10:5000/myapp:1.0.0
curl http://localhost:8080
```

Expected: `{"version": "1.0.0", "status": "ok"}` — no `feature` field. Rolled back.

This is why keeping old image versions in the registry matters. Rollback is instant — no rebuild required.

---

## Step 7 — Registry API

The registry exposes a REST API. Useful commands:

```bash
# List all repositories in the registry
curl http://192.168.40.10:5000/v2/_catalog

# List tags for an image
curl http://192.168.40.10:5000/v2/myapp/tags/list

# Get the manifest for a specific tag
curl http://192.168.40.10:5000/v2/myapp/manifests/1.0.0 \
  -H "Accept: application/vnd.docker.distribution.manifest.v2+json"
```

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
exit  # if still in a VM
yeast destroy
```

---

## Quick Recap

In Lab 15 — Private Container Registry, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What a private registry is and when to use one vs Docker Hub
- The OCI Distribution Spec: the standard API all registries implement
- `registry:2`: the official self-hosted Docker registry image
- Insecure registry configuration in `daemon.json`
- Image tagging convention: `<registry-host>/<name>:<tag>`
- `docker push` and `docker pull` from a private registry
- Image promotion: build once, push to registry, deploy to multiple environments
- Version tags vs `latest`: why `latest` moves and version tags are stable
- Registry API: `_catalog`, `tags/list`, `manifests/`
- Rollback by tag: no rebuild required, instant

---

## What Is Next

**Lab 16 — Progressive Delivery And Rollback**

You know how to deploy. Lab 16 teaches you how to deploy safely — with health checks, traffic shifting, and fast rollback. Blue/green and canary patterns: how to release a new version without taking down the old one until you are sure the new one works.
