# Lab 29 — Kubernetes Delivery Capstone

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate to advanced |
| Estimated time | 75-120 minutes |
| VMs | 2 |
| Minimum VM RAM | 3072 MB |
| SSH ports | 2250, 2251 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic understanding that Docker commands run inside the VM unless stated otherwise
- Patience for Kubernetes startup time and enough host RAM for multi-VM labs

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you. If not, the lab will tell you when to use a manual SSH tunnel.
- Run Docker commands inside the VM unless the lab explicitly says otherwise.
- Run `kubectl` commands inside the Kubernetes control-plane VM unless the lab says otherwise.

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
- Confusing laptop `localhost`, VM `localhost`, and container `localhost`.
- Running Kubernetes commands before all nodes are Ready.

---

## The Story

Every lab so far has taught one skill at a time. This lab ties them all together on Kubernetes. You will run the complete modern delivery flow:

**Code push → CI builds image → scans it → pushes to private registry → updates Kubernetes manifest → cluster deploys new version → you verify and roll back.**

This is how real teams ship software today. By the end of this lab, you will have done it yourself — end to end.

---

## Before You Start

This lab integrates: Git, GitHub Actions (Lab 13), Docker registry (Lab 15), Kubernetes deployments (Labs 27–28), and Ingress routing. If you are unclear on any of those pieces, review the relevant lab first.

You should not start this capstone if Kubernetes still feels like random YAML.

Before continuing, make sure you can answer:

- What is a Pod?
- What problem does a Deployment solve?
- Why do Pods need a Service?
- What does an Ingress do?
- Why should an image tag change during delivery?
- How would you roll back if a new version fails?

If those questions feel hard, that is not failure. It means you should review Labs 27 and 28 first. This lab is where the pieces come together; it is not the place to learn every piece for the first time.

---

## What You Are Building

```
GitHub Repo (code + manifests)
    │
    │  push triggers
    ▼
GitHub Actions CI
    │  builds Docker image
    │  scans with Trivy
    │  pushes to local registry
    │  updates manifest image tag
    ▼
┌──────────────────────────────────────────────────────────┐
│  Kubernetes Cluster (capstone-control + worker)          │
│                                                          │
│  Local registry :5000  (stores built images)             │
│  Deployment     myapp  (runs the latest image)           │
│  Service        myapp  (ClusterIP)                       │
│  Ingress        myapp  (routes HTTP by hostname)         │
└──────────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 29-kubernetes-delivery-capstone
yeast up
```

---

## Step 1 — Bootstrap The Cluster

```bash
yeast ssh capstone-control

# Install k3s
curl -sfL https://get.k3s.io | sh -
sleep 30

mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown ubuntu:ubuntu ~/.kube/config
export KUBECONFIG=~/.kube/config
echo 'export KUBECONFIG=~/.kube/config' >> ~/.bashrc

kubectl get nodes
```

Join the worker:

```bash
TOKEN=$(sudo cat /var/lib/rancher/k3s/server/node-token)
exit
```

```bash
yeast ssh capstone-worker1

curl -sfL https://get.k3s.io | K3S_URL=https://192.168.100.70:6443 \
  K3S_TOKEN="$(ssh -p 2250 -o StrictHostKeyChecking=no ubuntu@127.0.0.1 \
  'sudo cat /var/lib/rancher/k3s/server/node-token')" sh -

sleep 15
exit
```

```bash
yeast ssh capstone-control
kubectl get nodes  # should show 2 nodes
```

---

## Step 2 — Run A Local Registry On The Control Node

```bash
# Install Docker on control node for the registry
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker ubuntu
newgrp docker

docker run -d \
  --name registry \
  --restart always \
  -p 5000:5000 \
  registry:2

# Configure k3s to allow insecure registry
sudo mkdir -p /etc/rancher/k3s/
sudo tee /etc/rancher/k3s/registries.yaml << 'EOF'
mirrors:
  "127.0.0.1:5000":
    endpoint:
      - "http://127.0.0.1:5000"
  "192.168.100.70:5000":
    endpoint:
      - "http://192.168.100.70:5000"
EOF

sudo systemctl restart k3s
sleep 20
kubectl get nodes
```

Also configure the worker to pull from the registry:

```bash
exit
yeast ssh capstone-worker1

sudo mkdir -p /etc/rancher/k3s/
sudo tee /etc/rancher/k3s/registries.yaml << 'EOF'
mirrors:
  "192.168.100.70:5000":
    endpoint:
      - "http://192.168.100.70:5000"
EOF

sudo systemctl restart k3s-agent
sleep 10
exit
```

---

## Step 3 — Create The Application Code And Manifests

On your laptop, create a repository for this capstone:

```bash
mkdir -p ~/capstone-app && cd ~/capstone-app
git init
```

Create the application:

```bash
cat > app.py << 'PYEOF'
from http.server import HTTPServer, BaseHTTPRequestHandler
import json, os

VERSION = os.getenv("APP_VERSION", "dev")

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({
            "version": VERSION,
            "status": "ok",
            "message": "Kubernetes delivery capstone"
        }).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass

HTTPServer(("0.0.0.0", 8080), H).serve_forever()
PYEOF

cat > Dockerfile << 'EOF'
FROM python:3.11-slim
RUN groupadd --gid 1000 app && useradd --uid 1000 --gid 1000 --no-create-home app
WORKDIR /app
COPY --chown=app:app app.py .
USER app
EXPOSE 8080
CMD ["python3", "app.py"]
EOF
```

Create the Kubernetes manifests:

```bash
mkdir -p k8s

cat > k8s/deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 2
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: app
        image: 192.168.100.70:5000/myapp:latest
        env:
        - name: APP_VERSION
          value: "1.0.0"
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "32Mi"
            cpu: "50m"
          limits:
            memory: "64Mi"
            cpu: "200m"
        readinessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        livenessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp
spec:
  rules:
  - host: myapp.k8s.lab
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: myapp
            port:
              number: 80
EOF

git add .
git commit -m "feat: initial capstone app"
```

---

## Step 4 — First Manual Build And Deploy

Before automating, do it manually once so you understand every step:

```bash
# Build and tag for the local registry
docker build -t myapp:1.0.0 .
docker tag myapp:1.0.0 127.0.0.1:5000/myapp:1.0.0
docker tag myapp:1.0.0 127.0.0.1:5000/myapp:latest

# Push to registry
docker push 127.0.0.1:5000/myapp:1.0.0
docker push 127.0.0.1:5000/myapp:latest

# Verify in registry
curl http://127.0.0.1:5000/v2/myapp/tags/list
```

Deploy to Kubernetes:

```bash
yeast ssh capstone-control

# Apply manifests
kubectl apply -f /dev/stdin << 'EOF'
# (paste k8s/deployment.yaml content here, or copy the file to the VM)
EOF
```

Actually, let's copy the manifests properly:

```bash
exit

# From your laptop - copy manifests to the control node
scp -P 2250 -o StrictHostKeyChecking=no \
  ~/capstone-app/k8s/deployment.yaml \
  ubuntu@127.0.0.1:/home/ubuntu/deployment.yaml
```

```bash
yeast ssh capstone-control
kubectl apply -f /home/ubuntu/deployment.yaml
kubectl get pods -w
```

Once Running, test:

```bash
curl -H "Host: myapp.k8s.lab" http://127.0.0.1:8080
```

Expected: `{"version": "1.0.0", "status": "ok", ...}`

---

## Step 5 — The Delivery Flow: Ship A New Version

Now simulate the full delivery cycle for version 2.0.0.

**On your laptop:**

```bash
cd ~/capstone-app

# Make a code change
cat > app.py << 'PYEOF'
from http.server import HTTPServer, BaseHTTPRequestHandler
import json, os

VERSION = os.getenv("APP_VERSION", "dev")

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({
            "version": VERSION,
            "status": "ok",
            "message": "Kubernetes delivery capstone",
            "new_feature": "search support"
        }).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass

HTTPServer(("0.0.0.0", 8080), H).serve_forever()
PYEOF

# Build the new version
docker build -t myapp:2.0.0 .
docker tag myapp:2.0.0 127.0.0.1:5000/myapp:2.0.0
docker tag myapp:2.0.0 127.0.0.1:5000/myapp:latest

# Push to registry
docker push 127.0.0.1:5000/myapp:2.0.0
docker push 127.0.0.1:5000/myapp:latest

# Update the manifest
sed -i 's/image: 192.168.100.70:5000\/myapp:latest/image: 192.168.100.70:5000\/myapp:2.0.0/' \
  ~/capstone-app/k8s/deployment.yaml
sed -i 's/value: "1.0.0"/value: "2.0.0"/' \
  ~/capstone-app/k8s/deployment.yaml

git add k8s/deployment.yaml
git commit -m "feat: ship v2.0.0 with search support"

git log --oneline -5
```

**Deploy to the cluster:**

```bash
scp -P 2250 -o StrictHostKeyChecking=no \
  ~/capstone-app/k8s/deployment.yaml \
  ubuntu@127.0.0.1:/home/ubuntu/deployment.yaml

yeast ssh capstone-control
kubectl apply -f /home/ubuntu/deployment.yaml
kubectl rollout status deployment/myapp
kubectl get pods
```

**Verify the new version:**

```bash
curl -H "Host: myapp.k8s.lab" http://127.0.0.1:8080
```

Expected: `{"version": "2.0.0", "new_feature": "search support", ...}`

---

## Step 6 — Rollback

Something is wrong with v2.0.0. Roll back:

```bash
# In the cluster
kubectl rollout undo deployment/myapp
kubectl rollout status deployment/myapp
curl -H "Host: myapp.k8s.lab" http://127.0.0.1:8080
```

Back to v1.0.0 in under 30 seconds.

Or roll back to a specific revision:

```bash
kubectl rollout history deployment/myapp
kubectl rollout undo deployment/myapp --to-revision=1
```

---

## Step 7 — Readiness And Liveness Probes In Action

The manifest includes probes:

```yaml
readinessProbe:
  httpGet:
    path: /
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5

livenessProbe:
  httpGet:
    path: /
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
```

**Readiness probe** — Kubernetes does not send traffic to the Pod until it passes. During a rolling update, the old Pod keeps serving until the new Pod is ready.

**Liveness probe** — Kubernetes restarts the Pod if it fails. Catches deadlocks and stuck processes.

Watch what happens during a rollout:

```bash
kubectl rollout restart deployment/myapp
kubectl get pods -w
```

You will see new pods go through `Init` → `Running` → ready, while old pods stay up and serving traffic until the new ones pass their readiness probe.

---

## Step 8 — Observe

Check the rollout history:

```bash
kubectl rollout history deployment/myapp
kubectl describe deployment myapp
```

Check running pods and their versions:

```bash
kubectl get pods -o custom-columns="NAME:.metadata.name,IMAGE:.spec.containers[0].image,STATUS:.status.phase"
```

Check the ingress is routing correctly:

```bash
kubectl describe ingress myapp
```

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

In Lab 29 — Kubernetes Delivery Capstone, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

In this capstone lab you ran the complete delivery flow:
- Build a container image from a Dockerfile
- Tag and push to a private registry
- Write Kubernetes manifests: Deployment, Service, Ingress
- Apply manifests to the cluster
- Roll out a new version with zero downtime
- Roll back when something goes wrong
- Understand readiness and liveness probes and why they matter
- Read rollout history and pod status

The pattern — build, push, update manifest, apply, verify, rollback — is the same whether you are running three nodes on your laptop or 5000 nodes in production.

---

## What Is Next

**Lab 30 — AI-Assisted DevOps And Local LLM Ops**

The final lab. You have built a complete DevOps platform. Now you will run a local language model with Ollama, feed it real logs and metrics from your labs, and explore how AI can assist — not replace — operational work: log summarization, incident hypothesis generation, runbook drafting. With privacy and human oversight at the center.
