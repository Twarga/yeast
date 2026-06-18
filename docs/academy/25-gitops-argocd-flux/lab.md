# Lab 25 — GitOps With Argo CD

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate to advanced |
| Estimated time | 75-120 minutes |
| VMs | 1 |
| Minimum VM RAM | 4096 MB |
| SSH ports | 2240 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- A GitHub account and `gh` authentication when the lab uses GitHub
- Comfort creating SSH tunnels from `ACCESS.md` for browser-based tools
- Patience for Kubernetes startup time and enough host RAM for multi-VM labs

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether the lab asked you to open an SSH tunnel first.
- Run `kubectl` commands inside the Kubernetes control-plane VM unless the lab says otherwise.

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
- Opening Grafana, Prometheus, Jaeger, or Argo CD before the tunnel is running.
- Running Kubernetes commands before all nodes are Ready.

---

## The Story

You have a Kubernetes cluster. You want to deploy an application to it. The naive approach: `kubectl apply -f deployment.yaml`. Done.

But now: who ran that `kubectl apply`? When? Was it the same YAML that is in the repo? What if someone edited the live deployment directly? How do you roll back? How do you know what version is running in production right now?

GitOps solves all of this. The principle: Git is the only source of truth for what runs in the cluster. You never apply manifests manually. Instead, a GitOps controller (Argo CD) watches a Git repository and continuously reconciles the cluster to match what is in Git.

Want to deploy a new version? Commit the updated manifest to Git. Want to roll back? Revert the commit. Want to know what is in production? Look at the Git repo. The cluster always matches the repo, automatically, continuously.

---

## Before You Start — Understanding The Concepts

### What Is GitOps?

GitOps is an operational model where:
1. The desired state of your system is stored in Git
2. Changes to the desired state are made through Git (commits, PRs)
3. An automated agent continuously reconciles the actual state to match Git
4. Divergence (drift) is detected and corrected automatically

GitOps makes your infrastructure auditable (git log), reviewable (pull requests), and self-healing (automatic reconciliation).

### What Is Argo CD?

Argo CD is a declarative, GitOps continuous delivery tool for Kubernetes. It:
- Watches one or more Git repositories
- Compares the Kubernetes manifests in Git to the live cluster state
- Shows you what is out of sync
- Syncs the cluster to match Git (manually or automatically)
- Has a web UI and CLI for visibility

### What Is Reconciliation?

Reconciliation is the process of comparing desired state (Git) to actual state (cluster) and making changes to close the gap. Argo CD reconciles continuously — if someone manually edits a deployment in the cluster, Argo CD will detect the drift and revert it to what Git says.

### What Is An Argo CD Application?

An Argo CD Application is a custom resource that tells Argo CD:
- Which Git repo and path to watch (`repoURL`, `path`)
- Which Kubernetes cluster and namespace to deploy to (`destination`)
- How to sync: manual or automatic

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: myapp
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/you/your-manifests.git
    targetRevision: HEAD
    path: apps/myapp
  destination:
    server: https://kubernetes.default.svc
    namespace: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

`automated.prune: true` — delete resources removed from Git
`automated.selfHeal: true` — revert manual changes to match Git

---

## What You Are Building

```
Your Laptop
    │  SSH  2240 → gitops-cluster port 22
    │  HTTPS 8080 → gitops-cluster port 443 (Argo CD UI)
    ▼
┌──────────────────────────────────────────────────────┐
│  gitops-cluster VM                                   │
│                                                      │
│  k3s (lightweight Kubernetes)                        │
│    └── argocd namespace                              │
│          └── Argo CD controllers + UI                │
│    └── default namespace                             │
│          └── your app (deployed by Argo CD from Git) │
└──────────────────────────────────────────────────────┘
        ↑ watches
  GitHub repo with Kubernetes manifests
```

---

## Starting The Lab

```bash
cd 25-gitops-argocd-flux
yeast up
yeast ssh gitops-cluster
```

---

## Step 1 — Install k3s

k3s is a lightweight Kubernetes distribution. It runs as a single binary and is perfect for labs:

```bash
curl -sfL https://get.k3s.io | sh -

# Wait for k3s to start
sleep 20
sudo k3s kubectl get nodes
```

Configure kubectl for the ubuntu user:

```bash
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown ubuntu:ubuntu ~/.kube/config
sed -i 's/127.0.0.1/127.0.0.1/' ~/.kube/config

# Install kubectl alias
echo 'alias kubectl="k3s kubectl"' >> ~/.bashrc
source ~/.bashrc

kubectl get nodes
```

Expected:

```
NAME             STATUS   ROLES                  AGE   VERSION
gitops-cluster   Ready    control-plane,master   30s   v1.x.x
```

---

## Step 2 — Install Argo CD

```bash
kubectl create namespace argocd

kubectl apply -n argocd -f \
  https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for pods to start (takes 1-2 minutes)
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server \
  -n argocd --timeout=120s

kubectl get pods -n argocd
```

---

## Step 3 — Access The Argo CD UI

Get the initial admin password:

```bash
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d
echo  # print newline after the password
```

Inside the `gitops-cluster` VM, expose the Argo CD UI on the VM loopback interface:

```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443 --address=127.0.0.1 &
```

From your laptop, create a tunnel to that VM-local port:

```bash
ssh -N -L 8080:127.0.0.1:8080 -p 2240 ubuntu@127.0.0.1
```

Keep that tunnel terminal open. Then open: `https://localhost:8080` (accept the self-signed cert)

Login: `admin` / (the password from above)

---

## Step 4 — Create A Git Repository With Manifests

You need a GitHub repository that Argo CD will watch. Create one from your laptop:

```bash
# From your laptop
gh repo create gitops-lab25 --public --clone
cd gitops-lab25

mkdir -p apps/sample-app

cat > apps/sample-app/deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-app
  labels:
    app: sample-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sample-app
  template:
    metadata:
      labels:
        app: sample-app
    spec:
      containers:
      - name: app
        image: nginx:1.25
        ports:
        - containerPort: 80
EOF

cat > apps/sample-app/service.yaml << 'EOF'
apiVersion: v1
kind: Service
metadata:
  name: sample-app
spec:
  selector:
    app: sample-app
  ports:
  - port: 80
    targetPort: 80
  type: ClusterIP
EOF

git add .
git commit -m "feat: initial sample app manifests"
git push origin main
```

---

## Step 5 — Create An Argo CD Application

Back on the cluster VM, create the Argo CD Application pointing to your repo:

```bash
# In the gitops-cluster VM
cat > /home/ubuntu/argocd-app.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: sample-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/$(gh api user --jq .login 2>/dev/null || echo "YOUR_USERNAME")/gitops-lab25.git
    targetRevision: HEAD
    path: apps/sample-app
  destination:
    server: https://kubernetes.default.svc
    namespace: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
EOF
```

Edit the file to replace `YOUR_USERNAME` with your actual GitHub username, then apply:

```bash
kubectl apply -f /home/ubuntu/argocd-app.yaml

# Watch Argo CD sync it
kubectl get application -n argocd sample-app
sleep 30
kubectl get application -n argocd sample-app
```

The application should show `Synced` and `Healthy`.

Verify the deployment:

```bash
kubectl get pods
kubectl get deployments
```

---

## Step 6 — GitOps In Action: Update Via Git

From your laptop, change the deployment:

```bash
cd gitops-lab25
sed -i 's/replicas: 1/replicas: 2/' apps/sample-app/deployment.yaml
git add apps/sample-app/deployment.yaml
git commit -m "feat: scale to 2 replicas"
git push origin main
```

Watch Argo CD detect and apply the change (within the sync interval, usually 3 minutes):

```bash
# In the gitops-cluster VM
watch kubectl get pods  # press Ctrl+C when you see 2 pods
```

You did not run `kubectl apply`. You committed to Git. Argo CD reconciled the cluster. This is GitOps.

---

## Step 7 — GitOps Rollback

Something is wrong with the 2-replica deployment. Roll back by reverting the Git commit:

```bash
# From your laptop
git revert HEAD --no-edit
git push origin main
```

Argo CD detects the new commit (which reverted to 1 replica) and reconciles. The cluster goes back to 1 pod — no `kubectl` needed.

```bash
# In the cluster VM
sleep 60
kubectl get pods  # back to 1 pod
```

---

## Step 8 — Self-Healing: Argo CD Corrects Drift

Test `selfHeal: true` by manually changing the live deployment:

```bash
# In the cluster VM
kubectl scale deployment sample-app --replicas=5
kubectl get pods  # briefly shows 5 pods
```

Wait 1-2 minutes. Argo CD detects the drift and reverts it to match Git (1 replica):

```bash
kubectl get pods  # back to 1
```

The cluster always converges to what Git says. Manual changes are automatically reverted.

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
# Optionally delete the GitHub repo
# gh repo delete gitops-lab25 --yes
```

---

## Quick Recap

In Lab 25 — GitOps With Argo CD, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What GitOps is: Git as the source of truth, automated reconciliation to match
- Argo CD: what it is, how it watches a repo and syncs the cluster
- The Argo CD Application resource: repoURL, path, destination, syncPolicy
- `automated.prune`: delete resources removed from Git
- `automated.selfHeal`: revert manual cluster changes to match Git
- k3s: lightweight Kubernetes for labs
- GitOps workflow: change Git → Argo CD detects → cluster updates
- GitOps rollback: `git revert` → cluster rolls back
- Drift detection: manual change detected and reverted automatically

---

## What Is Next

**Lab 26 — End-To-End Delivery Platform**

Every piece is in place. Lab 26 assembles them: proxy, app, database, monitoring, logging — all on one platform, all wired together, deployed with automation. This is the VM-platform capstone before you move to Kubernetes.
