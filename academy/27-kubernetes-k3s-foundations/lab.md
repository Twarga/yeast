# Lab 27 — Kubernetes Foundations With k3s

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate to advanced |
| Estimated time | 90-150 minutes |
| VMs | 3 |
| Minimum VM RAM | 4096 MB |
| SSH ports | 2241, 2242, 2243 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Patience for Kubernetes startup time and enough host RAM for multi-VM labs

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you. If not, the lab will tell you when to use a manual SSH tunnel.
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
- Running Kubernetes commands before all nodes are Ready.

---

## The Story

You have built real platforms on VMs: services, networking, databases, monitoring. You manage them with Ansible and Terraform. You deploy with Docker Compose.

Kubernetes takes all of those concepts and reimplements them at a higher level of abstraction. Instead of configuring Nginx manually and writing systemd units, you describe what you want in YAML manifests and Kubernetes figures out how to schedule, run, restart, and scale your workloads across a cluster of machines.

This lab introduces Kubernetes from first principles. You will build a three-node cluster using k3s, deploy a real application, understand the core objects — Pods, Deployments, Services — and learn `kubectl`, the tool you will use for the rest of the Kubernetes labs.

---

## Kubernetes From Zero — The Mental Model

Before learning object names, slow down and connect Kubernetes to what you already know.

So far in this bootcamp, when you wanted to run an application, you did the work directly:

1. Create a VM.
2. Install packages.
3. Write a config file.
4. Start a service with systemd or Docker.
5. Check logs.
6. Restart or repair the service when it breaks.

Kubernetes changes the workflow.

Instead of manually starting containers on specific machines, you describe the desired state:

> "I want three copies of this app running. Expose them behind a stable service name. Restart them if they die. Roll out new versions safely."

Kubernetes then keeps trying to make reality match that description.

That idea is called **reconciliation**.

You have already seen this pattern:

- Yeast reads `yeast.yaml` and creates the VMs you described.
- Ansible reads playbooks and makes servers match the tasks you described.
- Terraform reads `.tf` files and makes infrastructure match the resources you described.
- Kubernetes reads YAML manifests and makes container workloads match the objects you described.

Kubernetes is not magic. It is a control system for containers.

It constantly asks:

1. What did the user ask for?
2. What is actually running right now?
3. What needs to change to make actual state match desired state?

That loop runs all the time.

### The Same Problems, New Names

Kubernetes can feel scary because the names are new. But the problems are familiar.

| What you already know | Kubernetes version |
|---|---|
| A VM runs workloads | A Node runs Pods |
| A Docker container runs an app | A Pod wraps one or more containers |
| systemd keeps a service running | A Deployment keeps Pods running |
| An Nginx upstream points to app servers | A Service points to matching Pods |
| A reverse proxy routes HTTP traffic | An Ingress routes HTTP traffic |
| `.env` files configure apps | ConfigMaps configure Pods |
| Passwords and tokens stay separate | Secrets configure Pods with sensitive values |
| A disk or Docker volume stores data | PersistentVolumeClaims request storage |
| `docker logs` shows container output | `kubectl logs` shows Pod output |
| `ssh` enters a VM | `kubectl exec` enters a container in a Pod |

Do not try to memorize everything at once. Start with this small chain:

```text
Node → Pod → Deployment → Service
```

That means:

```text
machine → running container wrapper → keeps copies alive → gives stable network access
```

If you understand that chain, the rest of Kubernetes becomes much less strange.

### The Three Ideas To Understand First

**1. Kubernetes runs containers across machines.**

You already ran containers with Docker on one VM. Kubernetes runs containers across a cluster of VMs.

**2. You usually do not run containers directly.**

With Docker, you used `docker run`. With Kubernetes, you usually write YAML and run `kubectl apply -f file.yaml`.

You are not saying:

```bash
run this exact container right now
```

You are saying:

```text
keep this application running in this shape
```

**3. Kubernetes repairs drift.**

If a Pod dies, Kubernetes creates another one. If you ask for three replicas and only two are running, Kubernetes starts a third. If you update the image, Kubernetes rolls out the new version.

That is why Kubernetes matters: it turns "run this container" into "keep this service alive."

### What You Should Not Worry About Yet

Kubernetes is huge. You do not need all of it today.

For this first Kubernetes lab, ignore:

- Helm
- operators
- service meshes
- cloud load balancers
- autoscaling
- RBAC
- production security policies
- multi-region clusters

Today you only need:

- Nodes
- Pods
- Deployments
- Services
- `kubectl`
- the idea of desired state

That is enough to start.

---

## Before You Start — Understanding The Concepts

### What Is Kubernetes?

Kubernetes (K8s) is an open-source container orchestration system. In plain language:

> Kubernetes runs containers across a group of machines and keeps them running the way you described.

Core capabilities:
- **Scheduling** — decides which node runs which container
- **Self-healing** — restarts failed containers, replaces unhealthy ones
- **Scaling** — adjusts the number of running containers up or down
- **Service discovery** — gives services stable DNS names regardless of where the pods are running
- **Rolling updates** — deploys new versions without downtime
- **Config and secrets management** — injects configuration into containers

### What Is A Cluster?

A Kubernetes cluster consists of:

**Control plane** — manages the cluster. Has the API server (all `kubectl` commands go here), the scheduler (decides where to run pods), and the controller manager (runs reconciliation loops).

**Worker nodes** — machines that run your workloads. Each worker runs `kubelet` (the node agent) and a container runtime (containerd or Docker).

In k3s, one VM can be both control plane and worker. For this lab, one VM is control plane and two are workers.

### What Is A Pod?

A Pod is the smallest deployable unit in Kubernetes. It wraps one or more containers that share a network namespace and storage. Containers in the same Pod can communicate over `localhost`.

Pods are ephemeral — when they die, they are replaced by new ones with new IPs. You never directly manage Pod IPs; you use Services for stable addressing.

### What Is A Deployment?

A Deployment manages a set of identical Pods. You tell it: "I want 3 replicas of this container image." It creates 3 Pods, monitors them, and if one dies, it creates a replacement. If you update the image, it rolls out the change Pod by Pod with zero downtime.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-demo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx-demo
  template:
    metadata:
      labels:
        app: nginx-demo
    spec:
      containers:
      - name: nginx
        image: nginx:1.25
        ports:
        - containerPort: 80
```

### What Is A Service?

A Service gives a stable DNS name and IP to a set of Pods. You select Pods with labels. Traffic to the Service is load-balanced across all matching Pods.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-demo
spec:
  selector:
    app: nginx-demo
  ports:
  - port: 80
    targetPort: 80
  type: ClusterIP
```

`ClusterIP` (default) — only reachable inside the cluster.
`NodePort` — maps to a port on every node, reachable from outside.
`LoadBalancer` — provisions a cloud load balancer (cloud providers only).

### What Is kubectl?

`kubectl` is the Kubernetes command-line client. It sends commands to the API server. Core commands:

```bash
kubectl get pods               # list pods
kubectl get nodes              # list nodes
kubectl describe pod <name>    # detailed info about a pod
kubectl logs <pod>             # read pod logs
kubectl exec -it <pod> -- bash # shell inside a pod
kubectl apply -f manifest.yaml # create/update from a file
kubectl delete -f manifest.yaml # delete resources in a file
kubectl scale deployment <name> --replicas=5
```

### What Is k3s?

k3s is a lightweight Kubernetes distribution that runs as a single binary. It removes optional components (some cloud providers, legacy APIs) to produce a ~50 MB binary that boots in seconds. Perfect for edge computing, CI, and labs.

---

## What You Are Building

```
Your Laptop
    │  SSH 2241 → k3s-control port 22
    │  SSH 2242 → k3s-worker1 port 22
    │  SSH 2243 → k3s-worker2 port 22
    ▼
┌──────────────────────────────────────────────────────────────┐
│  Private Network: 192.168.100.0/24                           │
│                                                              │
│  ┌──────────────────────┐                                    │
│  │  k3s-control (.10)   │  ← API server, scheduler          │
│  └──────────────────────┘                                    │
│           │ manages                                          │
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │  k3s-worker1    │  │  k3s-worker2    │                   │
│  │  (.11)          │  │  (.12)          │                   │
│  │  runs pods      │  │  runs pods      │                   │
│  └─────────────────┘  └─────────────────┘                   │
└──────────────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 27-kubernetes-k3s-foundations
yeast up
```

---

## Step 1 — Install k3s Control Plane

```bash
yeast ssh k3s-control

# Install k3s as the control plane
curl -sfL https://get.k3s.io | sh -

# Wait for it to start
sleep 30
sudo systemctl is-active k3s

# Set up kubectl for the ubuntu user
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown ubuntu:ubuntu ~/.kube/config
echo 'export KUBECONFIG=~/.kube/config' >> ~/.bashrc
source ~/.bashrc

kubectl get nodes
```

Expected: one node named `k3s-control` with status `Ready`.

Get the token needed to join worker nodes:

```bash
sudo cat /var/lib/rancher/k3s/server/node-token
```

Copy this token — you will need it for the workers.

Also note the control plane's private IP:

```bash
ip addr show | grep '192.168.100.10'
```

Exit:

```bash
exit
```

---

## Step 2 — Join Worker Nodes

Replace `<TOKEN>` with the token from the control plane.

```bash
yeast ssh k3s-worker1

curl -sfL https://get.k3s.io | K3S_URL=https://192.168.100.10:6443 \
  K3S_TOKEN="<TOKEN>" sh -

sleep 15
sudo systemctl is-active k3s-agent
exit
```

```bash
yeast ssh k3s-worker2

curl -sfL https://get.k3s.io | K3S_URL=https://192.168.100.10:6443 \
  K3S_TOKEN="<TOKEN>" sh -

sleep 15
sudo systemctl is-active k3s-agent
exit
```

Verify all nodes joined:

```bash
yeast ssh k3s-control
kubectl get nodes
```

Expected:

```
NAME           STATUS   ROLES                  AGE
k3s-control    Ready    control-plane,master   3m
k3s-worker1    Ready    <none>                 1m
k3s-worker2    Ready    <none>                 30s
```

---

## Step 3 — Your First Deployment

Create and apply a Deployment manifest:

```bash
# Still on k3s-control

cat > /home/ubuntu/nginx-demo.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-demo
  labels:
    app: nginx-demo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx-demo
  template:
    metadata:
      labels:
        app: nginx-demo
    spec:
      containers:
      - name: nginx
        image: nginx:1.25
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "32Mi"
            cpu: "50m"
          limits:
            memory: "64Mi"
            cpu: "200m"
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-demo
spec:
  selector:
    app: nginx-demo
  ports:
  - port: 80
    targetPort: 80
  type: NodePort
EOF

kubectl apply -f /home/ubuntu/nginx-demo.yaml
```

Watch the Pods being created:

```bash
kubectl get pods -w
```

`-w` watches for changes — press Ctrl+C when all three pods show `Running`.

---

## Step 4 — Explore With kubectl

```bash
# List pods with node assignment
kubectl get pods -o wide

# Describe a pod (look at Events at the bottom — shows the scheduling decision)
kubectl describe pod <pod-name-from-above>

# Read pod logs
kubectl logs <pod-name>

# Shell inside a running pod
kubectl exec -it <pod-name> -- bash
# Inside: ls /etc/nginx, nginx -v, exit
```

`kubectl describe` is one of the most useful commands in Kubernetes. It shows:
- The Pod spec (which image, what resources)
- The Pod's current status (phase, conditions, IP)
- The container status (running, restart count, last state)
- Events: what happened (image pulled, container started, any failures)

When a Pod is misbehaving, `kubectl describe` is always your first diagnostic tool.

---

## Step 5 — The Service And NodePort

Get the NodePort assigned to the nginx-demo service:

```bash
kubectl get service nginx-demo
```

```
NAME         TYPE       CLUSTER-IP     EXTERNAL-IP   PORT(S)        AGE
nginx-demo   NodePort   10.43.x.x      <none>        80:3xxxx/TCP   2m
```

The number after `80:` (e.g., `31234`) is the NodePort. Accessible on any node's IP:

```bash
# Test from control node on the NodePort
NODE_PORT=$(kubectl get service nginx-demo -o jsonpath='{.spec.ports[0].nodePort}')
curl http://192.168.100.10:${NODE_PORT}
curl http://192.168.100.11:${NODE_PORT}  # same app, different node
curl http://192.168.100.12:${NODE_PORT}  # same app, third node
```

All three work — the Service routes traffic to any Pod regardless of which node it is on.

---

## Step 6 — Self-Healing

Delete a Pod and watch Kubernetes replace it:

```bash
# Get a pod name
POD=$(kubectl get pods -l app=nginx-demo -o jsonpath='{.items[0].metadata.name}')
echo "Deleting: $POD"

kubectl delete pod "$POD"

# Watch the replacement appear immediately
kubectl get pods -w
```

The Deployment controller noticed the Pod count dropped to 2, so it created a new Pod immediately. The Service continued routing to the remaining 2 Pods during the brief replacement.

---

## Step 7 — Scaling

```bash
# Scale up to 5 replicas
kubectl scale deployment nginx-demo --replicas=5
kubectl get pods -w  # watch 2 new pods appear

# Scale back down
kubectl scale deployment nginx-demo --replicas=2
kubectl get pods -w  # watch 3 pods terminate

# Or edit the manifest and apply
sed -i 's/replicas: 3/replicas: 3/' /home/ubuntu/nginx-demo.yaml  # restore
kubectl apply -f /home/ubuntu/nginx-demo.yaml
```

---

## Step 8 — Rolling Update

Update the image version:

```bash
kubectl set image deployment/nginx-demo nginx=nginx:1.26

# Watch the rolling update
kubectl rollout status deployment/nginx-demo
kubectl get pods -w
```

Kubernetes replaces Pods one at a time (by default). At no point does it take all Pods offline. The Service routes to the old Pods until new ones are ready, then shifts traffic.

View rollout history:

```bash
kubectl rollout history deployment/nginx-demo
```

Roll back:

```bash
kubectl rollout undo deployment/nginx-demo
kubectl rollout status deployment/nginx-demo
```

---

## Step 9 — Namespaces

Kubernetes uses namespaces to isolate resources logically:

```bash
# List namespaces
kubectl get namespaces

# Create a new namespace
kubectl create namespace staging

# Deploy in the staging namespace
kubectl apply -f /home/ubuntu/nginx-demo.yaml -n staging

# View resources by namespace
kubectl get pods -n staging
kubectl get pods -n default
kubectl get pods --all-namespaces
```

Namespaces give you isolation: dev and prod can coexist in the same cluster without interfering.

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

In Lab 27 — Kubernetes Foundations With k3s, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- Kubernetes cluster architecture: control plane vs worker nodes
- Pods: the smallest deployable unit, ephemeral, IP-per-pod
- Deployments: declarative pod management, self-healing, replicas
- Services: stable DNS/IP for a set of pods, NodePort for external access
- kubectl: `get`, `describe`, `logs`, `exec`, `apply`, `delete`, `scale`, `rollout`
- k3s: lightweight Kubernetes — same API, smaller footprint
- Self-healing: Kubernetes immediately replaces deleted pods
- Rolling updates: zero-downtime image updates
- `kubectl rollout undo`: instant rollback
- Namespaces: logical isolation within a cluster

---

## What Is Next

**Lab 28 — Kubernetes Networking, Config, And Storage**

You have pods and services. Lab 28 adds the real platform layer: Ingress for HTTP routing, ConfigMaps and Secrets for configuration, and PersistentVolumes for stateful storage. These are the concepts that let you run real applications on Kubernetes.
