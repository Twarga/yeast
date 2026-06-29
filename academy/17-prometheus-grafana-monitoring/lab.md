# Lab 17 — Prometheus And Grafana Monitoring

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 2 |
| Minimum VM RAM | 3072 MB |
| SSH ports | 2226, 2227 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic understanding that Docker commands run inside the VM unless stated otherwise
- Comfort using forwarded Yeast host URLs from `ACCESS.md` for browser-based tools

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you. If not, the lab will tell you when to use a manual SSH tunnel.
- Run Docker commands inside the VM unless the lab explicitly says otherwise.

### Expected Checkpoints

- After `yeast up`, `yeast status` should show the expected VM or VMs as running.
- After the main setup steps, the service, tool, or workflow introduced by the lab should respond to the verification commands.
- After `bash assets/validate.sh`, the script should report all checks passed.
- After `yeast destroy`, the lab should be cleaned up before you start the next one.

### Common Mistakes To Avoid

- Running a VM command on your laptop, or a laptop command inside the VM.
- Ignoring the forwarded port shown by `yeast up` or `yeast status`.
- Skipping validation because the final page or command "looked fine".
- Forgetting to run `yeast destroy` before moving to the next lab.
- Confusing laptop `localhost`, VM `localhost`, and container `localhost`.
- Opening Grafana, Prometheus, Jaeger, or Argo CD before the forwarded service is ready.

---

## The Story

A server goes down. Nobody knew. Users started complaining at 9 AM. The server had been slowly running out of memory since 6 AM. If someone had been watching a memory graph, they would have caught it before it became an incident.

This is what monitoring is for. Not to admire dashboards — to catch problems before users do, and to understand what "normal" looks like so you recognize "abnormal."

Prometheus is the de facto standard metrics system for Linux infrastructure and cloud-native applications. Grafana is the visualization layer. Together they let you define what healthy looks like, set alerts for when it is not, and give you graphs to understand what happened during and before an incident.

---

## Before You Start — Understanding The Concepts

### What Is A Metric?

A metric is a measurement of something over time. Examples:
- CPU usage on web-01: 23% at 14:00, 41% at 14:01, 38% at 14:02
- HTTP requests per second to the API: 142 at 14:00, 156 at 14:01
- Memory used on db-primary: 4.2 GB at 14:00, 4.3 GB at 14:01

Metrics are numeric, timestamped data points. They answer quantitative questions: "how much?", "how often?", "how long?".

### What Is Prometheus?

Prometheus is a monitoring system that:
1. **Scrapes** metrics from your servers and applications at a regular interval (default: 15s)
2. **Stores** them in a time-series database (each metric is a sequence of (timestamp, value) pairs)
3. **Provides PromQL** — a query language to select, aggregate, and calculate over the stored metrics
4. **Evaluates alerts** — when a metric crosses a threshold, Prometheus fires an alert

Prometheus uses a **pull model**: it polls `/metrics` endpoints on your servers. Each server exposes its metrics via HTTP; Prometheus pulls them.

This is the opposite of a push model (like sending metrics to a central server). Pull means Prometheus controls the scrape interval and can detect when a target is down.

### What Is An Exporter?

An exporter is a program that collects metrics from a system and exposes them at `/metrics` in Prometheus format.

**Node Exporter** — collects OS-level metrics: CPU, RAM, disk, network, filesystem. Install it on every VM you want to monitor.

**Other exporters:**
- `nginx-prometheus-exporter` — Nginx request rates, error rates
- `postgres_exporter` — PostgreSQL query counts, lock waits, cache hit rates
- `redis_exporter` — Redis operations, memory, connections

### What Is PromQL?

PromQL (Prometheus Query Language) lets you query the time-series database. Examples:

```
# CPU usage percentage over last 5 minutes
100 - (avg by(instance)(irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# Memory usage percentage
(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100

# HTTP request rate
rate(http_requests_total[5m])
```

PromQL is how you build dashboards and alerts.

### What Is Grafana?

Grafana is a visualization tool. It connects to Prometheus (and dozens of other data sources) and lets you build dashboards: graphs, gauges, tables, alerts. It has a rich editor and a library of pre-built dashboards for common exporters.

### What Is An Alert?

An alert in Prometheus is a rule: "if this PromQL expression is true for X minutes, fire an alert." Alerts are sent to Alertmanager, which routes them to Slack, PagerDuty, email, etc.

For this lab you define alert rules but send them to a simple webhook — the full Alertmanager setup is in Lab 20.

---

## What You Are Building

```
Your Laptop
    │  HTTP 9090 → monitoring port 9090 (Prometheus UI)
    │  HTTP 3000 → monitoring port 3000 (Grafana)
    │  SSH  2226 → monitoring port 22
    │  SSH  2227 → app port 22
    ▼
┌─────────────────────────────────────────────────────┐
│  Private Network: 192.168.60.0/24                   │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  monitoring (.10)                            │   │
│  │  Prometheus :9090    (scrapes → :9100)       │   │
│  │  Grafana     :3000   (reads from Prometheus) │   │
│  └──────────────────────────────────────────────┘   │
│            ↑ scrapes /metrics                        │
│  ┌──────────────────────────────────┐               │
│  │  app (.20)                       │               │
│  │  node_exporter :9100             │               │
│  │  app :8000                       │               │
│  └──────────────────────────────────┘               │
└─────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 17-prometheus-grafana-monitoring
yeast up
```

---

## Step 1 — Install Node Exporter On The App VM

Node Exporter is a binary that exports OS metrics. Install it on the app VM:

```bash
yeast ssh app

# Download and install node_exporter
NODE_EXPORTER_VERSION="1.7.0"
wget -q "https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
tar xzf node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz
sudo cp node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64/node_exporter /usr/local/bin/
rm -rf node_exporter-*

# Create a system user for node_exporter (security: runs as non-root)
sudo useradd --no-create-home --shell /bin/false node_exporter

# Create a systemd service
sudo tee /etc/systemd/system/node_exporter.service << 'EOF'
[Unit]
Description=Node Exporter
After=network.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter
Restart=always

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now node_exporter
sudo systemctl is-active node_exporter
```

Verify metrics are available:

```bash
curl http://localhost:9100/metrics | head -20
```

You see hundreds of metrics: `node_cpu_seconds_total`, `node_memory_MemAvailable_bytes`, `node_disk_read_bytes_total`, etc.

Exit the app VM:

```bash
exit
```

---

## Step 2 — Start Prometheus And Grafana

SSH to the monitoring VM:

```bash
yeast ssh monitoring
newgrp docker
mkdir -p /home/ubuntu/monitoring && cd /home/ubuntu/monitoring
```

Create the Prometheus config:

```bash
cat > prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "prometheus"
    static_configs:
      - targets: ["localhost:9090"]

  - job_name: "node"
    static_configs:
      - targets: ["192.168.60.20:9100"]
        labels:
          instance: "app"
          env: "lab"

rule_files:
  - "rules/*.yml"
EOF

mkdir -p rules
cat > rules/alerts.yml << 'EOF'
groups:
  - name: lab_alerts
    rules:
      - alert: HighCPU
        expr: 100 - (avg by(instance)(irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High CPU on {{ $labels.instance }}"
          description: "CPU above 80% for 2 minutes"

      - alert: HighMemory
        expr: (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 > 85
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High memory on {{ $labels.instance }}"
EOF
```

Create the Docker Compose stack:

```bash
cat > compose.yaml << 'EOF'
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./rules:/etc/prometheus/rules:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.enable-lifecycle'
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_USER: admin
      GF_SECURITY_ADMIN_PASSWORD: admin
      GF_USERS_ALLOW_SIGN_UP: "false"
    volumes:
      - grafana-data:/var/lib/grafana
    restart: unless-stopped

volumes:
  prometheus-data:
  grafana-data:
EOF

docker compose up -d
```

Wait 15 seconds and check:

```bash
docker compose ps
curl http://localhost:9090/-/healthy
curl -s http://localhost:3000 | head -5
```

---

## Step 3 — Explore Prometheus UI

From your laptop, create a tunnel to the monitoring VM:

```bash
ssh -N \
  -L 9090:127.0.0.1:9090 \
  -L 3000:127.0.0.1:3000 \
  -p 2226 ubuntu@127.0.0.1
```

Keep that tunnel terminal open. Then open Prometheus from your laptop: `http://localhost:9090`

### Check Targets

Go to Status → Targets. You should see two targets:
- `prometheus` (localhost:9090) — Prometheus scraping itself
- `node` (192.168.60.20:9100) — the app VM's node exporter

Both should show State: UP.

### Run Queries

In the Expression bar, try these PromQL queries:

**CPU idle percentage:**
```
node_cpu_seconds_total{mode="idle"}
```

**CPU usage (rate over 5 minutes):**
```
100 - avg by(instance)(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100
```

**Memory used:**
```
(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / (1024*1024*1024)
```

**Disk usage:**
```
node_filesystem_size_bytes{mountpoint="/"} - node_filesystem_free_bytes{mountpoint="/"}
```

Click "Graph" to see the time series. This is real data from your app VM, scraped every 15 seconds.

---

## Step 4 — Set Up Grafana

With the same SSH tunnel still open, open Grafana at `http://localhost:3000`. Login: admin / admin.

### Add Prometheus Data Source

1. Go to Connections → Data Sources → Add new data source
2. Choose Prometheus
3. Set URL: `http://prometheus:9090` (uses Docker service name)
4. Click "Save & Test" — should say "Data source is working"

### Import The Node Exporter Dashboard

The community maintains pre-built dashboards. Import the most popular Node Exporter dashboard:

1. Dashboards → New → Import
2. Enter dashboard ID: `1860` (Node Exporter Full)
3. Select your Prometheus data source
4. Click Import

You now have a complete dashboard showing CPU, memory, disk, network — all from your app VM — with beautiful graphs, no query writing required.

---

## Step 5 — Generate Load And Watch Metrics

From the app VM, run a CPU-intensive process and watch it appear in Grafana:

```bash
yeast ssh app
# Run a CPU stress test for 60 seconds
python3 -c "
import time
import multiprocessing

def burn(n):
    end = time.time() + n
    while time.time() < end:
        x = sum(i*i for i in range(10000))

procs = [multiprocessing.Process(target=burn, args=(60,)) for _ in range(2)]
for p in procs: p.start()
for p in procs: p.join()
print('done')
"
exit
```

Watch the CPU graph in Grafana update in real time (15-second intervals). You will see the spike during the stress test and the return to baseline after.

---

## Step 6 — Understanding The Alert Rules

Look at the alert rules you defined:

```bash
yeast ssh monitoring
cat /home/ubuntu/monitoring/rules/alerts.yml
exit
```

In Prometheus UI → Alerts, you can see the alert rules and their current state (inactive, pending, firing).

`HighCPU` fires when CPU exceeds 80% for 2 consecutive minutes. `for: 2m` prevents brief spikes from triggering alerts. If you make the stress test run for 3+ minutes at high CPU, this alert will fire.

---

## Step 7 — Reload Prometheus Config Without Restart

When you change `prometheus.yml` or alert rules, you can reload without restarting:

```bash
yeast ssh monitoring
curl -X POST http://localhost:9090/-/reload
exit
```

This is possible because of `--web.enable-lifecycle` in the Prometheus command. Hot reload means no scrape interruption.

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

In Lab 17 — Prometheus And Grafana Monitoring, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What metrics are: numeric, timestamped measurements
- Prometheus pull model: scrapes `/metrics` endpoints at intervals
- Node Exporter: what it exposes and how to install it as a systemd service
- PromQL: basic queries for CPU, memory, disk
- `scrape_configs` and `static_configs` in `prometheus.yml`
- Alert rules: `expr`, `for`, `labels`, `annotations`
- Grafana: adding a data source, importing community dashboards
- Hot reload: `POST /-/reload` instead of restarting
- The goal: define what healthy looks like before something breaks

---

## What Is Next

**Lab 18 — Centralized Logging**

Metrics tell you numbers. Logs tell you what happened. Lab 18 sets up a centralized log aggregation stack — collect logs from multiple VMs, ship them to a central store, and query them. You will trace an incident across services using only log data.
