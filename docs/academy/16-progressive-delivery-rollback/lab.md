# Lab 16 — Progressive Delivery And Rollback

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 3 |
| Minimum VM RAM | 3072 MB |
| SSH ports | 2223, 2224, 2225 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether the lab asked you to open an SSH tunnel first.

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

---

## The Story

Your team ships a new version. You stop the old service and start the new one. During the restart — even if it is 10 seconds — users see errors. And if the new version has a bug, you just took down production and your rollback means another 10 seconds of downtime.

There is a better way. Progressive delivery means releasing a new version gradually, without downtime, with the ability to instantly roll back if something goes wrong. The two most common patterns are **blue/green deployment** and **canary releases**.

This lab teaches both patterns using Nginx as the traffic controller and two application VMs as deployment targets.

---

## Before You Start — Understanding The Concepts

### What Is Blue/Green Deployment?

Blue/green deployment uses two identical environments: blue (current production) and green (the new version). You deploy the new version to green while blue keeps serving all traffic. When green is verified, you flip the load balancer to send all traffic to green. Blue becomes the standby.

Benefits:
- **Zero downtime** — the flip is instant (a config reload)
- **Instant rollback** — flip back to blue if green has problems
- **Full environment** — green is a complete, production-like environment that you can test before switching

Cost: you need double the infrastructure (two full environments running simultaneously).

### What Is A Canary Release?

A canary release sends a small percentage of traffic to the new version while the majority still goes to the old version. If the new version looks healthy (low error rates, good latency, no alerts), you gradually increase the percentage until it gets 100%.

Named after "canary in a coal mine" — you put a small number of users on the new version first, so if it has a fatal bug, only a small percentage of users are affected.

Benefits:
- Risk-limited: only a small subset sees the new version at first
- Real traffic validation: you test with real users, not synthetic tests
- Gradual: you can stop at any percentage if something looks wrong

### What Is A Health Check?

A health check is an endpoint your application exposes specifically to indicate whether it is ready to serve traffic. Typically `/healthz` or `/health` returning a 200 status when healthy, a non-200 status when not ready.

The load balancer polls the health check before sending traffic. If a backend fails its health check, the load balancer stops sending it requests — automatically, without manual intervention.

### What Is A Weighted Upstream In Nginx?

Nginx lets you assign weights to upstream servers:

```nginx
upstream backend {
    server 192.168.50.21:8000 weight=9;   # 90% of traffic
    server 192.168.50.22:8000 weight=1;   # 10% of traffic
}
```

This is how you implement a canary: 90% old, 10% new. Increase green's weight over time.

---

## What You Are Building

```
Your Laptop
    │  HTTP 8880 → lb port 80
    │  SSH  2223 → lb port 22
    │  SSH  2224 → blue port 22
    │  SSH  2225 → green port 22
    ▼
┌─────────────────────────────────────────────────────────┐
│  Private Network: 192.168.50.0/24                       │
│                                                         │
│  ┌──────────────┐    ┌──────────────┐ ┌──────────────┐ │
│  │  lb          │───▶│  blue (.21)  │ │  green (.22) │ │
│  │  Nginx       │    │  v1.0.0      │ │  v2.0.0      │ │
│  │  :80         │    │  :8000       │ │  :8000       │ │
│  └──────────────┘    └──────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 16-progressive-delivery-rollback
yeast up
```

---

## Step 1 — Deploy V1 To Blue

SSH to blue and run a v1 app:

```bash
yeast ssh blue

cat > /home/ubuntu/app.py << 'EOF'
from http.server import HTTPServer, BaseHTTPRequestHandler
import json

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/healthz":
            b = json.dumps({"status": "ok"}).encode()
            code = 200
        else:
            b = json.dumps({"version": "1.0.0", "host": "blue"}).encode()
            code = 200
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)
    def log_message(self, *a): pass

HTTPServer(("0.0.0.0", 8000), H).serve_forever()
EOF

# Run as a background process
nohup python3 /home/ubuntu/app.py > /home/ubuntu/app.log 2>&1 &
echo $! > /home/ubuntu/app.pid

curl http://localhost:8000
exit
```

---

## Step 2 — Deploy V2 To Green (But Not Live Yet)

SSH to green and run a v2 app:

```bash
yeast ssh green

cat > /home/ubuntu/app.py << 'EOF'
from http.server import HTTPServer, BaseHTTPRequestHandler
import json

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/healthz":
            b = json.dumps({"status": "ok"}).encode()
            code = 200
        else:
            b = json.dumps({"version": "2.0.0", "host": "green", "feature": "new"}).encode()
            code = 200
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)
    def log_message(self, *a): pass

HTTPServer(("0.0.0.0", 8000), H).serve_forever()
EOF

nohup python3 /home/ubuntu/app.py > /home/ubuntu/app.log 2>&1 &
echo $! > /home/ubuntu/app.pid

curl http://localhost:8000
exit
```

Both apps are running. Green is deployed but gets no traffic yet.

---

## Step 3 — Configure Load Balancer (Blue Only)

SSH to lb:

```bash
yeast ssh lb

sudo tee /etc/nginx/sites-available/app << 'EOF'
upstream backend {
    server 192.168.50.21:8000;
}

server {
    listen 80;
    server_name _;

    location /healthz {
        proxy_pass http://backend;
    }

    location / {
        proxy_pass         http://backend;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/app /etc/nginx/sites-enabled/app
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx

curl http://localhost
exit
```

All traffic goes to blue (v1.0.0).

---

## Step 4 — Blue/Green Switch

V2 on green has been tested and validated. Switch all traffic from blue to green:

```bash
yeast ssh lb

sudo tee /etc/nginx/sites-available/app << 'EOF'
upstream backend {
    server 192.168.50.22:8000;
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
EOF

sudo nginx -t
sudo systemctl reload nginx

# Traffic now goes to green
curl http://localhost
exit
```

All requests now return `{"version": "2.0.0"}`. Blue is still running — just not receiving traffic.

Test from your laptop:

```bash
curl http://localhost:8880
```

Expected: `{"version": "2.0.0", "host": "green", "feature": "new"}`

---

## Step 5 — Rollback To Blue

Something is wrong with v2. Roll back instantly:

```bash
yeast ssh lb

sudo tee /etc/nginx/sites-available/app << 'EOF'
upstream backend {
    server 192.168.50.21:8000;
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://backend;
        proxy_set_header Host $host;
    }
}
EOF

sudo nginx -t
sudo systemctl reload nginx
curl http://localhost
exit
```

Back to v1.0.0. Rollback time: under 5 seconds.

---

## Step 6 — Canary Release

Now implement a canary: send 10% of traffic to green, 90% to blue:

```bash
yeast ssh lb

sudo tee /etc/nginx/sites-available/app << 'EOF'
upstream backend {
    server 192.168.50.21:8000 weight=9;
    server 192.168.50.22:8000 weight=1;
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://backend;
        proxy_set_header Host $host;
    }
}
EOF

sudo nginx -t
sudo systemctl reload nginx
exit
```

Test distribution from your laptop:

```bash
for i in $(seq 20); do curl -s http://localhost:8880 | grep '"host"'; done | sort | uniq -c
```

Expected: roughly 18 requests to blue, 2 to green (10%).

Increase to 50/50:

```bash
yeast ssh lb
# Change weight=9/weight=1 to weight=1/weight=1
sudo sed -i 's/weight=9/weight=1/' /etc/nginx/sites-available/app
sudo nginx -t && sudo systemctl reload nginx
exit
```

And then to 100% green (full rollout):

```bash
yeast ssh lb
sudo tee /etc/nginx/sites-available/app << 'EOF'
upstream backend {
    server 192.168.50.22:8000;
}
server {
    listen 80;
    server_name _;
    location / {
        proxy_pass http://backend;
    }
}
EOF
sudo nginx -t && sudo systemctl reload nginx
exit
```

---

## Step 7 — Health Checks

Add automatic health checking so Nginx removes a failed backend automatically:

```bash
yeast ssh lb

sudo tee /etc/nginx/sites-available/app << 'EOF'
upstream backend {
    server 192.168.50.21:8000 max_fails=2 fail_timeout=10s;
    server 192.168.50.22:8000 max_fails=2 fail_timeout=10s;
}

server {
    listen 80;
    server_name _;
    location / {
        proxy_pass http://backend;
        proxy_next_upstream error timeout;
    }
}
EOF

sudo nginx -t && sudo systemctl reload nginx
exit
```

`max_fails=2 fail_timeout=10s` — if a backend fails 2 requests in 10 seconds, remove it from rotation for 10 seconds. `proxy_next_upstream` retries failed requests on the next backend.

Test by killing blue:

```bash
yeast ssh blue
kill $(cat /home/ubuntu/app.pid)
exit

# All requests now go to green automatically
curl http://localhost:8880
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

In Lab 16 — Progressive Delivery And Rollback, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- Blue/green deployment: two environments, instant traffic flip, instant rollback
- Canary release: weighted upstreams in Nginx, gradual traffic shifting
- Health checks: `max_fails`, `fail_timeout`, `proxy_next_upstream`
- The critical difference between deploy (put new version on infrastructure) and release (send traffic to new version)
- Rollback by config change: no rebuild, no redeploy — just a different upstream address
- Why blue always stays running until green is proven: the safety net

---

## What Is Next

**Lab 17 — Prometheus And Grafana Monitoring**

You can deploy. You can roll back. But how do you know when something is going wrong before users complain? Lab 17 installs Prometheus to collect metrics and Grafana to visualize them, and teaches you to define what "healthy" looks like for your services.
