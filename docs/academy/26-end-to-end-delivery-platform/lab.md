# Lab 26 — End-To-End Delivery Platform On VMs

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate to advanced |
| Estimated time | 90-150 minutes |
| VMs | 4 |
| Minimum VM RAM | 5120 MB |
| SSH ports | 2244, 2245, 2246, 2247 |
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

---

## The Story

You have built every component separately: proxy, app, database, CI/CD, monitoring, logging. Each one works. But in real operations, these components must work together as a coherent platform. You deploy an app and the monitoring system starts tracking it automatically. You push code and CI builds and deploys it. You have an incident and the logs and metrics are available in Grafana.

This lab is the VM capstone. You will wire up all four VMs into a functioning platform, connect each component to the others, and then simulate a complete deployment cycle: code change → CI build → deploy → verify in monitoring.

---

## Before You Start

This lab assumes you are comfortable with everything from Labs 01–25. It does not re-explain concepts — it integrates them. If you are unclear on any component, go back to its lab.

---

## What You Are Building

```
Your Laptop
    │  HTTP 9090 → plat-proxy port 80  (the app)
    │  HTTP 9190 → plat-monitoring port 9090  (Prometheus)
    │  HTTP 9300 → plat-monitoring port 3000  (Grafana)
    │  SSH ports: 2244 2245 2246 2247
    ▼
┌──────────────────────────────────────────────────────────────────┐
│  Platform Network: 192.168.110.0/24                              │
│                                                                  │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────┐ │
│  │  plat-proxy  │──▶│  plat-app    │──▶│  plat-db             │ │
│  │  .10         │   │  .20         │   │  .30                 │ │
│  │  Nginx :80   │   │  Docker app  │   │  PostgreSQL :5432    │ │
│  └──────────────┘   └──────────────┘   └──────────────────────┘ │
│                          │ metrics                               │
│                          ▼                                       │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │  plat-monitoring (.40)                                   │    │
│  │  Prometheus :9090 + Grafana :3000 + Loki :3100           │    │
│  └──────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 26-end-to-end-delivery-platform
yeast up
```

This boots four VMs. It will take 3–5 minutes. Use this time to review the architecture.

---

## Phase 1 — Database

```bash
yeast ssh plat-db

sudo -u postgres psql << 'SQL'
CREATE USER appuser WITH PASSWORD 'platform26';
CREATE DATABASE appdb OWNER appuser;
GRANT ALL PRIVILEGES ON DATABASE appdb TO appuser;
SQL

sudo -u postgres psql -d appdb << 'SQL'
CREATE TABLE items (id SERIAL PRIMARY KEY, name TEXT, created_at TIMESTAMP DEFAULT NOW());
INSERT INTO items (name) VALUES ('first'),('second'),('third');
SQL

PG_HBA=$(sudo -u postgres psql -t -c "SHOW hba_file;" | tr -d ' ')
echo "host appdb appuser 192.168.110.20/32 md5" | sudo tee -a "$PG_HBA"
sudo -u postgres psql -c "ALTER SYSTEM SET listen_addresses = '*';"
sudo systemctl restart postgresql
sudo ss -tlnp | grep 5432
exit
```

---

## Phase 2 — Application

```bash
yeast ssh plat-app
newgrp docker

mkdir -p /home/ubuntu/app

cat > /home/ubuntu/app/app.py << 'PYEOF'
import json, os, time, psycopg2, psycopg2.extras
from http.server import HTTPServer, BaseHTTPRequestHandler
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST

REQ_COUNT = Counter('http_requests_total', 'Requests', ['method', 'status'])
REQ_LATENCY = Histogram('http_request_duration_seconds', 'Latency')

DB = {"host": os.environ["DB_HOST"], "port": 5432,
      "dbname": os.environ["DB_NAME"], "user": os.environ["DB_USER"],
      "password": os.environ["DB_PASS"]}

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        start = time.time()
        if self.path == "/metrics":
            body = generate_latest()
            self.send_response(200)
            self.send_header("Content-Type", CONTENT_TYPE_LATEST)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        try:
            with psycopg2.connect(**DB) as c:
                with c.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                    cur.execute("SELECT id, name, created_at::text FROM items ORDER BY id")
                    rows = [dict(r) for r in cur.fetchall()]
            body = json.dumps({"items": rows}).encode(); code = 200
            REQ_COUNT.labels(method='GET', status='200').inc()
        except Exception as e:
            body = json.dumps({"error": str(e)}).encode(); code = 500
            REQ_COUNT.labels(method='GET', status='500').inc()
        REQ_LATENCY.observe(time.time() - start)
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass

HTTPServer(("0.0.0.0", 8080), H).serve_forever()
PYEOF

cat > /home/ubuntu/app/Dockerfile << 'EOF'
FROM python:3.11-slim
RUN pip install --no-cache-dir psycopg2-binary prometheus_client
WORKDIR /app
COPY app.py .
EXPOSE 8080
CMD ["python3", "app.py"]
EOF

docker build -t platform-app:1.0.0 /home/ubuntu/app/

cat > /home/ubuntu/app/.env << 'EOF'
DB_HOST=192.168.110.30
DB_NAME=appdb
DB_USER=appuser
DB_PASS=platform26
EOF

docker run -d \
  --name app \
  --restart unless-stopped \
  -p 8080:8080 \
  --env-file /home/ubuntu/app/.env \
  platform-app:1.0.0

sleep 3
curl http://localhost:8080/items
curl http://localhost:8080/metrics | head -10
exit
```

---

## Phase 3 — Proxy

```bash
yeast ssh plat-proxy

sudo tee /etc/nginx/sites-available/platform << 'EOF'
upstream app {
    server 192.168.110.20:8080;
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass         http://app;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    access_log /var/log/nginx/platform.access.log;
    error_log  /var/log/nginx/platform.error.log;
}
EOF

sudo ln -sf /etc/nginx/sites-available/platform /etc/nginx/sites-enabled/platform
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
curl http://localhost/items
exit
```

Test end-to-end from your laptop:

```bash
curl http://localhost:9090/items
```

Expected: the three items from the database.

---

## Phase 4 — Monitoring Stack

```bash
yeast ssh plat-monitoring
newgrp docker
mkdir -p /home/ubuntu/monitoring/rules && cd /home/ubuntu/monitoring

cat > prometheus.yml << 'EOF'
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "prometheus"
    static_configs:
      - targets: ["localhost:9090"]

  - job_name: "app"
    static_configs:
      - targets: ["192.168.110.20:8080"]
        labels:
          env: platform
          service: app

  - job_name: "node-proxy"
    static_configs:
      - targets: ["192.168.110.10:9100"]
        labels:
          env: platform
          role: proxy

  - job_name: "node-app"
    static_configs:
      - targets: ["192.168.110.20:9100"]
        labels:
          env: platform
          role: app

  - job_name: "node-db"
    static_configs:
      - targets: ["192.168.110.30:9100"]
        labels:
          env: platform
          role: db
EOF

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
    volumes:
      - grafana-data:/var/lib/grafana
    restart: unless-stopped

  loki:
    image: grafana/loki:2.9.0
    container_name: loki
    ports:
      - "3100:3100"
    restart: unless-stopped

volumes:
  prometheus-data:
  grafana-data:
EOF

docker compose up -d
sleep 15
curl http://localhost:9090/-/healthy
exit
```

---

## Phase 5 — Install Node Exporter On All VMs

Run this on proxy, app, and db to expose OS metrics:

```bash
for PORT in 2244 2245 2246; do
  ssh -p $PORT -o StrictHostKeyChecking=no ubuntu@127.0.0.1 << 'REMOTE'
NODE_VER="1.7.0"
wget -q "https://github.com/prometheus/node_exporter/releases/download/v${NODE_VER}/node_exporter-${NODE_VER}.linux-amd64.tar.gz"
tar xzf node_exporter-${NODE_VER}.linux-amd64.tar.gz
sudo cp node_exporter-${NODE_VER}.linux-amd64/node_exporter /usr/local/bin/
rm -rf node_exporter-*
sudo useradd --no-create-home --shell /bin/false node_exporter 2>/dev/null || true
echo "[Unit]
Description=Node Exporter
[Service]
User=node_exporter
ExecStart=/usr/local/bin/node_exporter
Restart=always
[Install]
WantedBy=multi-user.target" | sudo tee /etc/systemd/system/node_exporter.service
sudo systemctl daemon-reload
sudo systemctl enable --now node_exporter
echo "node_exporter started on $(hostname)"
REMOTE
done
```

---

## Phase 6 — Verify The Full Platform

From your laptop, run these checks:

```bash
# App responds through proxy
curl http://localhost:9090/items

# Add a new item
curl -X POST http://localhost:9090/items \
  -H "Content-Type: application/json" \
  -d '{"name": "platform-item"}' 2>/dev/null || echo "POST not implemented — GET only in this lab"

# Prometheus is scraping
curl -s "http://localhost:9190/api/v1/targets" | python3 -m json.tool | grep '"health"' | sort | uniq -c

# Grafana is up
curl -s -o /dev/null -w "%{http_code}" http://localhost:9300
```

Open Grafana at `http://localhost:9300` (admin/admin):
1. Add Prometheus data source: `http://prometheus:9090`
2. Import Node Exporter Full dashboard (ID: 1860)
3. You should see metrics from all four VMs

---

## Phase 7 — Simulate A Deployment

This simulates what a CI/CD pipeline would do: build a new image version and deploy it.

```bash
yeast ssh plat-app

# Build v2.0.0 (same app, just a new tag to simulate a change)
docker build -t platform-app:2.0.0 /home/ubuntu/app/

# Deploy with zero-downtime swap
docker stop app && docker rm app
docker run -d \
  --name app \
  --restart unless-stopped \
  -p 8080:8080 \
  --env-file /home/ubuntu/app/.env \
  platform-app:2.0.0

sleep 3
curl http://localhost:8080/items
exit

# Verify through proxy
curl http://localhost:9090/items
```

In a real pipeline, this `docker stop/run` sequence would be replaced by a Compose rolling update or a Kubernetes rollout. The pattern is the same.

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

In Lab 26 — End-To-End Delivery Platform On VMs, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

This lab did not introduce new concepts — it required you to apply all the concepts from Labs 01–25 in a coherent, integrated system. The skills demonstrated:

- Multi-VM platform design and network layout
- Database setup and remote access configuration
- Containerized application deployment with environment variables
- Reverse proxy configuration for routing to the app tier
- Prometheus scrape configuration for multiple targets
- Node Exporter deployment across a fleet
- Grafana dashboard setup against real infrastructure
- Simulated deployment lifecycle: build → stop old → start new → verify

---

## What Is Next

**Lab 27 — Kubernetes Foundations With k3s**

You have mastered VM-based platform engineering. The next three labs move to Kubernetes — where the same concepts (services, networking, storage, configuration) are expressed differently. k3s gives you a real cluster on your laptop.
