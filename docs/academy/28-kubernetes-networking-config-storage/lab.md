# Lab 28 — Kubernetes Networking, Config, And Storage

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate to advanced |
| Estimated time | 75-120 minutes |
| VMs | 2 |
| Minimum VM RAM | 3072 MB |
| SSH ports | 2248, 2249 |
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
- Running Kubernetes commands before all nodes are Ready.

---

## The Story

In Lab 27 you deployed Pods and exposed them with a NodePort Service. That is enough to prove Kubernetes works, but it is not how real applications are deployed. Real applications need:

- A single entry point that routes HTTP by hostname or path (not random port numbers)
- Configuration injected from the outside — not baked into the image
- Persistent storage that survives Pod restarts
- Secrets managed separately from configuration

This lab adds those layers: Ingress, ConfigMaps, Secrets, and PersistentVolumes. After this lab, you will know how to deploy a real application on Kubernetes, not just a demo.

---

## Before You Go Deeper

This lab assumes Lab 27 makes sense at a basic level.

Before continuing, you should be able to explain this chain in your own words:

```text
Node → Pod → Deployment → Service
```

Simple version:

- A **Node** is a machine in the cluster.
- A **Pod** is where containers run.
- A **Deployment** keeps the right number of Pods alive.
- A **Service** gives those Pods a stable network address.

If that still feels blurry, re-read the "Kubernetes From Zero" section in Lab 27 before starting this lab. Lab 28 adds more pieces, so the foundation matters.

This lab adds the next layer:

| Need | Kubernetes object |
|---|---|
| HTTP routing from outside the app | Ingress |
| Non-secret app settings | ConfigMap |
| Passwords and sensitive values | Secret |
| Data that survives Pod replacement | PersistentVolumeClaim |

The mental model stays the same:

> describe what the app needs, then let Kubernetes reconcile the cluster toward that desired state.

---

## Before You Start — Understanding The Concepts

### What Is An Ingress?

An Ingress is a Kubernetes object that routes HTTP/HTTPS traffic into the cluster based on hostname and path rules. It is the Kubernetes equivalent of an Nginx reverse proxy config.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp
spec:
  rules:
  - host: myapp.lab
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: myapp
            port:
              number: 80
```

An Ingress Controller is the actual component that reads Ingress objects and configures a real load balancer or proxy. k3s ships with Traefik as the default Ingress Controller.

### What Is A ConfigMap?

A ConfigMap stores non-sensitive configuration data as key-value pairs. You inject it into Pods as environment variables or as files.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  DB_HOST: "postgres-service"
  LOG_LEVEL: "info"
  MAX_CONNECTIONS: "100"
```

ConfigMaps decouple configuration from container images. The same image runs with different configs in dev and prod.

### What Is A Secret?

A Secret is like a ConfigMap but for sensitive data — passwords, tokens, certificates. Kubernetes base64-encodes Secret values (this is not encryption — it is encoding). For real encryption at rest, you configure etcd encryption or use an external secrets manager.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secret
type: Opaque
stringData:
  DB_PASS: "mysecretpassword"
  API_KEY: "sk-abc123"
```

### What Is A PersistentVolume (PV) And PersistentVolumeClaim (PVC)?

Pods are ephemeral. When a Pod is replaced, its local filesystem is gone. PersistentVolumes solve this.

A **PersistentVolume (PV)** is a piece of storage provisioned in the cluster — could be a local disk, NFS, or a cloud disk.

A **PersistentVolumeClaim (PVC)** is a request for storage by a Pod. You say "I need 5 GB of ReadWriteOnce storage" and Kubernetes binds a matching PV to your PVC.

k3s includes a local-path provisioner that automatically creates PVs from the node's local disk — no cloud required.

### What Are Resource Requests And Limits?

Every container in Kubernetes should declare:
- **Requests** — the minimum resources Kubernetes guarantees (used for scheduling)
- **Limits** — the maximum resources the container can use

```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "128Mi"
    cpu: "500m"
```

`100m` CPU = 0.1 CPU core. `64Mi` = 64 mebibytes of RAM. Without limits, a runaway container can starve other workloads.

---

## What You Are Building

A two-node k3s cluster running:
- A Python app reading config from ConfigMap and Secrets
- PostgreSQL with a PersistentVolume for data
- Ingress routing HTTP traffic to the app

---

## Starting The Lab

```bash
cd 28-kubernetes-networking-config-storage
yeast up
yeast ssh k8s-control
```

Install k3s:

```bash
curl -sfL https://get.k3s.io | sh -
sleep 20

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
CONTROL_IP="192.168.100.60"
exit
```

```bash
yeast ssh k8s-worker

curl -sfL https://get.k3s.io | K3S_URL=https://192.168.100.60:6443 \
  K3S_TOKEN="$(ssh -p 2248 -o StrictHostKeyChecking=no ubuntu@127.0.0.1 \
  'sudo cat /var/lib/rancher/k3s/server/node-token')" sh -

sleep 15
exit
```

Verify:

```bash
yeast ssh k8s-control
kubectl get nodes
```

---

## Step 1 — ConfigMap And Secret

```bash
mkdir -p /home/ubuntu/k8s && cd /home/ubuntu/k8s

cat > configmap.yaml << 'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  DB_HOST: "postgres"
  DB_PORT: "5432"
  DB_NAME: "appdb"
  DB_USER: "appuser"
  LOG_LEVEL: "info"
EOF

cat > secret.yaml << 'EOF'
apiVersion: v1
kind: Secret
metadata:
  name: app-secret
type: Opaque
stringData:
  DB_PASS: "k8slab28secret"
EOF

kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml

# Verify
kubectl get configmap app-config
kubectl describe configmap app-config
kubectl get secret app-secret
```

Note: `kubectl describe secret` does NOT show the values — by design.

To verify a secret value (for debugging):

```bash
kubectl get secret app-secret -o jsonpath='{.data.DB_PASS}' | base64 -d
echo
```

---

## Step 2 — PostgreSQL With PersistentVolume

```bash
cat > postgres.yaml << 'EOF'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_DB
          valueFrom:
            configMapKeyRef:
              name: app-config
              key: DB_NAME
        - name: POSTGRES_USER
          valueFrom:
            configMapKeyRef:
              name: app-config
              key: DB_USER
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: app-secret
              key: DB_PASS
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
  type: ClusterIP
EOF

kubectl apply -f postgres.yaml
kubectl get pvc  # should show Bound after a moment
kubectl get pods -l app=postgres -w
```

Wait for the postgres pod to be Running. The PVC is automatically bound by k3s's local-path provisioner.

---

## Step 3 — Application With ConfigMap And Secret Injection

```bash
cat > app.yaml << 'EOF'
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
      initContainers:
      - name: wait-for-db
        image: busybox
        command: ['sh', '-c',
          'until nc -z postgres 5432; do echo waiting for postgres; sleep 2; done']
      containers:
      - name: app
        image: python:3.11-slim
        command: ["python3", "-c"]
        args:
          - |
            import os, json, time
            from http.server import HTTPServer, BaseHTTPRequestHandler
            import psycopg2

            DB_CONFIG = {
                "host": os.environ["DB_HOST"],
                "port": int(os.environ["DB_PORT"]),
                "dbname": os.environ["DB_NAME"],
                "user": os.environ["DB_USER"],
                "password": os.environ["DB_PASS"],
            }

            def init_db():
                with psycopg2.connect(**DB_CONFIG) as c:
                    with c.cursor() as cur:
                        cur.execute("CREATE TABLE IF NOT EXISTS items (id SERIAL PRIMARY KEY, name TEXT)")
                        cur.execute("INSERT INTO items (name) SELECT 'item-' || i FROM generate_series(1,3) i ON CONFLICT DO NOTHING")
                    c.commit()

            for i in range(10):
                try:
                    init_db()
                    break
                except Exception as e:
                    print(f"DB not ready: {e}"); time.sleep(3)

            class H(BaseHTTPRequestHandler):
                def do_GET(self):
                    with psycopg2.connect(**DB_CONFIG) as c:
                        with c.cursor() as cur:
                            cur.execute("SELECT id, name FROM items ORDER BY id")
                            rows = [{"id": r[0], "name": r[1]} for r in cur.fetchall()]
                    body = json.dumps({"items": rows, "log_level": os.environ["LOG_LEVEL"]}).encode()
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.send_header("Content-Length", str(len(body)))
                    self.end_headers()
                    self.wfile.write(body)
                def log_message(self, *a): pass

            HTTPServer(("0.0.0.0", 8080), H).serve_forever()
        envFrom:
        - configMapRef:
            name: app-config
        env:
        - name: DB_PASS
          valueFrom:
            secretKeyRef:
              name: app-secret
              key: DB_PASS
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "500m"
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
  type: ClusterIP
EOF

kubectl apply -f app.yaml
kubectl get pods -l app=myapp -w
```

Key concepts in this manifest:
- `envFrom.configMapRef` — injects ALL ConfigMap keys as environment variables
- `env[].valueFrom.secretKeyRef` — injects a specific Secret key
- `initContainers` — runs before the main container; here it waits for PostgreSQL to be ready before starting the app

---

## Step 4 — Ingress

k3s ships with Traefik as the Ingress Controller:

```bash
kubectl get pods -n kube-system | grep traefik

cat > ingress.yaml << 'EOF'
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
spec:
  rules:
  - host: myapp.lab
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

kubectl apply -f ingress.yaml
kubectl get ingress
```

Test with a Host header (since we do not have real DNS):

```bash
NODE_IP="192.168.100.60"
curl -H "Host: myapp.lab" http://${NODE_IP}
```

Expected: `{"items": [...], "log_level": "info"}`

---

## Step 5 — Verify PVC Persistence

Prove the PVC persists data across Pod restarts:

```bash
# Get the app pod name and check items
POD=$(kubectl get pods -l app=myapp -o jsonpath='{.items[0].metadata.name}')
kubectl exec "$POD" -- python3 -c "
import psycopg2, os
with psycopg2.connect(host='postgres', port=5432, dbname='appdb',
                       user='appuser', password='k8slab28secret') as c:
    with c.cursor() as cur:
        cur.execute('INSERT INTO items (name) VALUES (%s) RETURNING id', ('persistent-item',))
        print('Inserted id:', cur.fetchone()[0])
    c.commit()
"

# Delete the postgres pod — Kubernetes will recreate it, data on PVC is preserved
kubectl delete pod -l app=postgres

# Wait for new postgres pod
kubectl get pods -l app=postgres -w

# Verify the data is still there
curl -H "Host: myapp.lab" http://192.168.100.60 | python3 -m json.tool
```

The `persistent-item` you inserted is still in the database — because the data lives on the PVC, not inside the Pod.

---

## Step 6 — Update ConfigMap And Rollout

Change the LOG_LEVEL in the ConfigMap and trigger a rollout:

```bash
kubectl patch configmap app-config \
  --type merge \
  -p '{"data": {"LOG_LEVEL": "debug"}}'

# Trigger a rollout to pick up the new config
# (Pod env vars are set at start time, not live-updated)
kubectl rollout restart deployment/myapp
kubectl rollout status deployment/myapp

curl -H "Host: myapp.lab" http://192.168.100.60 | grep log_level
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

In Lab 28 — Kubernetes Networking, Config, And Storage, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- Ingress: HTTP routing by hostname/path, Ingress Controllers (Traefik in k3s)
- ConfigMap: injecting non-sensitive config as env vars with `envFrom` and `env[].valueFrom`
- Secret: same pattern for sensitive data, base64-encoded, never shown in describe
- PersistentVolumeClaim: requesting storage, PVC binding to PV
- Local-path provisioner: k3s automatic PV creation from node disk
- PVC persistence: data survives Pod deletion and recreation
- initContainers: startup ordering — wait for dependencies before main container
- Resource requests and limits: `cpu` and `memory`, `m` for millicores, `Mi` for mebibytes
- Rolling restart after ConfigMap change: env vars are set at container start

---

## What Is Next

**Lab 29 — Kubernetes Delivery Capstone**

You know the building blocks. Lab 29 runs the complete delivery flow on Kubernetes: push code, CI builds the image, pushes to the registry, updates the Kubernetes manifest, and the cluster deploys the new version — all automatically.
