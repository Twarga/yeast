# Lab 18 — Centralized Logging

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 2 |
| Minimum VM RAM | 3072 MB |
| SSH ports | 2228, 2229 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic understanding that Docker commands run inside the VM unless stated otherwise
- Comfort creating SSH tunnels from `ACCESS.md` for browser-based tools

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
- Opening Grafana, Prometheus, Jaeger, or Argo CD before the tunnel is running.

---

## The Story

It is 2 AM. The app is returning 500 errors. You have three VMs: a proxy, an app server, a database. Each one has its own log file. To understand what happened, you SSH into each one separately, search through potentially gigabytes of logs, and try to stitch together a timeline from three different terminals.

Centralized logging solves this. You ship all logs to one place. One query to find everything related to an incident. One timeline. No SSH required.

This lab sets up the Grafana Loki stack: Loki stores the logs, Promtail ships logs from each server, Grafana lets you query and visualize them. It is lightweight, works well with Prometheus (same Grafana), and is straightforward to run.

---

## Before You Start — Understanding The Concepts

### What Is Centralized Logging?

Centralized logging is the practice of collecting logs from all your servers and services into a single queryable store. Instead of SSHing into each machine to read `/var/log/`, you query a central system.

Benefits:
- **Single pane of glass** — all logs in one place
- **Correlation** — trace a request across multiple services by timestamp
- **Retention** — logs persist even after a server is destroyed
- **Search** — full-text search across millions of log lines instantly
- **Alerting** — trigger alerts on log patterns (e.g., "more than 10 errors per minute")

### What Is Loki?

Loki is a log aggregation system from Grafana Labs. It is designed to be cheap and efficient — unlike Elasticsearch, it does not index the content of log lines. Instead, it indexes only the labels (metadata like `host`, `job`, `level`) and stores the log lines compressed.

This makes Loki much cheaper to run than Elasticsearch, at the cost of slower full-text search. For most use cases — especially infrastructure logs — this trade-off is worth it.

### What Is Promtail?

Promtail is the log shipping agent for Loki. It runs on each server, tails log files, adds labels, and pushes log entries to Loki.

Think of Promtail as "the agent that reads your log files and sends them to Loki." It can read:
- Files: `/var/log/nginx/access.log`, `/var/log/app/*.log`
- Systemd journal: all service logs via `journald`
- Docker container logs

### What Is LogQL?

LogQL is Loki's query language, similar to PromQL. Examples:

```
# All logs from the app host
{host="app"}

# All error lines from nginx
{job="nginx"} |= "ERROR"

# Count of errors per minute
count_over_time({job="nginx"} |= "error" [1m])

# Filter by regex
{host="app"} |~ "status=[45][0-9]{2}"
```

### What Is A Label In Loki?

Labels are key-value metadata attached to log streams. They are how Loki indexes and filters logs.

Common labels:
- `host` — the hostname that generated the log
- `job` — the service or application name
- `level` — log level (info, warn, error)
- `filename` — the source log file path

You define labels in Promtail's config. When querying, you filter by labels first, then by log content.

---

## What You Are Building

```
Your Laptop
    │  HTTP 3100 → logs port 3100 (Loki API)
    │  HTTP 3000 → logs port 3000 (Grafana)
    │  SSH  2228 → logs port 22
    │  SSH  2229 → app port 22
    ▼
┌────────────────────────────────────────────────────────┐
│  Private Network: 192.168.70.0/24                      │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │  logs (.10)                                      │  │
│  │  Loki :3100    (receives logs from promtail)     │  │
│  │  Grafana :3000 (query logs via LogQL)            │  │
│  └──────────────────────────────────────────────────┘  │
│             ↑ push logs                                 │
│  ┌──────────────────────────────────┐                  │
│  │  app (.20)                       │                  │
│  │  Promtail (tails logs → Loki)    │                  │
│  │  Python app :8000                │                  │
│  └──────────────────────────────────┘                  │
└────────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 18-centralized-logging
yeast up
```

---

## Step 1 — Start Loki And Grafana

SSH to the logs VM:

```bash
yeast ssh logs
newgrp docker
mkdir -p /home/ubuntu/logging && cd /home/ubuntu/logging
```

Create a minimal Loki config:

```bash
cat > loki-config.yml << 'EOF'
auth_enabled: false

server:
  http_listen_port: 3100

ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
    final_sleep: 0s
  chunk_idle_period: 5m
  chunk_retain_period: 30s

schema_config:
  configs:
    - from: 2024-01-01
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

storage_config:
  boltdb_shipper:
    active_index_directory: /loki/index
    cache_location: /loki/cache
    shared_store: filesystem
  filesystem:
    directory: /loki/chunks

limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: 168h

chunk_store_config:
  max_look_back_period: 0s

table_manager:
  retention_deletes_enabled: false
  retention_period: 0s
EOF
```

Create the Compose file:

```bash
cat > compose.yaml << 'EOF'
services:
  loki:
    image: grafana/loki:2.9.0
    container_name: loki
    ports:
      - "3100:3100"
    volumes:
      - ./loki-config.yml:/etc/loki/local-config.yaml:ro
      - loki-data:/loki
    command: -config.file=/etc/loki/local-config.yaml
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_USER: admin
      GF_SECURITY_ADMIN_PASSWORD: admin
      GF_AUTH_ANONYMOUS_ENABLED: "false"
    volumes:
      - grafana-data:/var/lib/grafana
    restart: unless-stopped

volumes:
  loki-data:
  grafana-data:
EOF

docker compose up -d
```

Wait 10 seconds and verify Loki is ready:

```bash
curl http://localhost:3100/ready
```

Expected: `ready`

Exit the logs VM:

```bash
exit
```

---

## Step 2 — Install Promtail On The App VM

```bash
yeast ssh app

PROMTAIL_VERSION="2.9.0"
wget -q "https://github.com/grafana/loki/releases/download/v${PROMTAIL_VERSION}/promtail-linux-amd64.zip"
sudo apt-get install -y -qq unzip
unzip -q promtail-linux-amd64.zip
sudo mv promtail-linux-amd64 /usr/local/bin/promtail
rm promtail-linux-amd64.zip
```

Create a systemd user and config:

```bash
sudo useradd --no-create-home --shell /bin/false promtail
sudo usermod -aG systemd-journal promtail  # allow reading journald

sudo mkdir -p /etc/promtail

sudo tee /etc/promtail/config.yml << 'EOF'
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://192.168.70.10:3100/loki/api/v1/push

scrape_configs:
  - job_name: system
    static_configs:
      - targets:
          - localhost
        labels:
          job: syslog
          host: app
          __path__: /var/log/*.log

  - job_name: journal
    journal:
      max_age: 12h
      labels:
        host: app
        job: journal
    relabel_configs:
      - source_labels: ["__journal__systemd_unit"]
        target_label: "unit"
      - source_labels: ["__journal_priority_keyword"]
        target_label: "level"
EOF

sudo tee /etc/systemd/system/promtail.service << 'EOF'
[Unit]
Description=Promtail Log Shipper
After=network.target

[Service]
User=promtail
ExecStart=/usr/local/bin/promtail -config.file=/etc/promtail/config.yml
Restart=always

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now promtail
sudo systemctl is-active promtail
```

Generate some log entries:

```bash
logger -t "lab18" "Test log from app VM"
logger -t "lab18" "ERROR: something failed"
logger -t "lab18" "INFO: application started"
```

Wait 15 seconds for Promtail to ship them, then exit:

```bash
exit
```

---

## Step 3 — Query Logs In Grafana

From your laptop, create a tunnel to the logs VM:

```bash
ssh -N \
  -L 3000:127.0.0.1:3000 \
  -L 3100:127.0.0.1:3100 \
  -p 2228 ubuntu@127.0.0.1
```

Keep that tunnel terminal open. Then open `http://localhost:3000` — login with admin/admin.

### Add Loki Data Source

1. Connections → Data Sources → Add new data source
2. Choose Loki
3. URL: `http://loki:3100`
4. Save & Test

### Explore Logs

1. Click the "Explore" icon (compass)
2. Select Loki data source
3. In the Log Browser, choose label `host` = `app`
4. Run the query

You should see your test log lines appearing.

### Useful LogQL Queries

**All logs from app:**
```
{host="app"}
```

**Only error lines:**
```
{host="app"} |= "ERROR"
```

**Journal logs for nginx (if installed):**
```
{job="journal", unit="nginx.service"}
```

**Count errors per minute:**
```
sum(count_over_time({host="app"} |= "ERROR" [1m]))
```

---

## Step 4 — Run An App And Watch Its Logs

Start a simple Python app on the app VM that generates structured logs:

```bash
yeast ssh app

cat > /home/ubuntu/app.py << 'EOF'
import time, json, logging, random
from http.server import HTTPServer, BaseHTTPRequestHandler

logging.basicConfig(
    format='%(asctime)s %(levelname)s %(message)s',
    handlers=[
        logging.FileHandler("/var/log/app.log"),
        logging.StreamHandler()
    ],
    level=logging.INFO
)
log = logging.getLogger("app")

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if random.random() < 0.1:
            log.error(f"path={self.path} status=500 latency_ms={random.randint(100,500)}")
            b = json.dumps({"error": "internal server error"}).encode()
            code = 500
        else:
            latency = random.randint(5, 100)
            log.info(f"path={self.path} status=200 latency_ms={latency}")
            b = json.dumps({"status": "ok"}).encode()
            code = 200
        self.send_response(code)
        self.send_header("Content-Type","application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)
    def log_message(self, *a): pass

log.info("App starting on :8000")
HTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
EOF

# Make app.log writable
sudo touch /var/log/app.log
sudo chown ubuntu:ubuntu /var/log/app.log

nohup python3 /home/ubuntu/app.py &
sleep 2

# Generate traffic
for i in $(seq 50); do curl -s http://localhost:8000 > /dev/null; done
exit
```

Update Promtail to also watch `/var/log/app.log`:

```bash
yeast ssh app
# Add app log to scrape config
sudo tee -a /etc/promtail/config.yml << 'EOF'

  - job_name: app
    static_configs:
      - targets:
          - localhost
        labels:
          job: myapp
          host: app
          __path__: /var/log/app.log
EOF

sudo systemctl restart promtail
exit
```

Now in Grafana, query:
```
{job="myapp"}
```

You see real application logs: path, status codes, latencies. Filter for errors:
```
{job="myapp"} |= "status=500"
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

In Lab 18 — Centralized Logging, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What centralized logging is and the problem it solves
- Loki: label-based indexing, compressed log storage, why it is cheaper than Elasticsearch
- LogQL: filtering by labels, full-text search with `|=`, regex with `|~`, counting with `count_over_time`
- Promtail: config structure, file tailing, journald scraping, label definition
- The `positions.yaml` file: Promtail's bookmark for where it left off in each file
- Structuring application logs: key=value format for easier parsing
- Connecting Loki to Grafana and querying across services
- The incident investigation workflow: labels first (which service?), then content (what happened?)

---

## What Is Next

**Lab 19 — OpenTelemetry Distributed Tracing**

Metrics tell you numbers. Logs tell you events. Traces tell you the path of a request through your system — which services it touched, how long each took, where it failed. Lab 19 introduces OpenTelemetry and distributed tracing for multi-service request visibility.
