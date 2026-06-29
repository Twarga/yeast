# Lab 20 — SRE: SLOs, Alerts, And Error Budgets

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 1 |
| Minimum VM RAM | 2048 MB |
| SSH ports | 2233 |
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
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you.
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
- Opening Grafana, Prometheus, Jaeger, or Argo CD before the tunnel is running.

---

## The Story

You have Prometheus. You have dashboards. You have hundreds of metrics. And yet your on-call engineer gets paged at 3 AM for a CPU spike that resolved itself in 30 seconds. And nobody got paged when the checkout flow was returning 5% errors for two hours.

This is the alerting problem. Too many noisy alerts means people ignore them. Too few means real problems go unnoticed. The solution is not more dashboards — it is a principled way to define what "broken" means for your users, and only alert when that definition is violated.

That is what SLOs are. Service Level Objectives define exactly what reliability you are committing to, measured in terms users care about. Error budgets give you a way to quantify how much unreliability you have left before you need to stop shipping features and fix the platform.

---

## Before You Start — Understanding The Concepts

### What Is An SLI?

SLI (Service Level Indicator) is a metric that measures the quality of your service from the user's perspective.

Good SLIs are things users actually feel:
- **Availability** — what percentage of requests succeed?
- **Latency** — what percentage of requests complete in under 300ms?
- **Error rate** — what percentage of requests return an error?

Bad SLIs are internal metrics users do not directly feel:
- CPU usage (users care about latency, not CPU)
- Number of database connections (users care about error rate, not connection pool size)

### What Is An SLO?

SLO (Service Level Objective) is a target for an SLI over a time window. Examples:
- "99.9% of requests succeed" — measured over 30 days
- "95% of requests complete in under 500ms" — measured over 7 days
- "Error rate stays below 0.1%" — measured over 30 days

An SLO is a commitment. Meeting it means your users are having the experience you promised.

### What Is An SLA?

SLA (Service Level Agreement) is a contract with an external party — usually a customer — that includes financial penalties if you miss your SLOs. Your SLOs should be stricter than your SLAs to give you buffer.

You do not need an SLA to benefit from SLOs. SLOs are useful even as internal targets.

### What Is An Error Budget?

If your SLO is 99.9% availability over 30 days, you are allowed 0.1% failures. In a 30-day month with 43,200 minutes, 0.1% is 43.2 minutes of downtime.

That 43.2 minutes is your error budget. You can spend it on:
- Incidents (unplanned)
- Deployments that cause brief errors (planned)
- Experiments and risky feature rollouts

When the error budget is exhausted, you stop spending it — no new risky deployments, focus on reliability. When it is healthy, you can move fast.

This gives engineering and product a shared language: "we have 35 minutes of error budget left this month — is this deploy worth risking 5 of them?"

### What Is Alert Fatigue?

Alert fatigue is what happens when you get too many alerts that do not require action. Engineers learn to ignore them. The critical alert gets buried in noise. Real incidents get missed.

The solution: only alert on things that violate your SLO. If CPU spikes but your error rate and latency SLIs are fine, do not alert. If your error budget burn rate is too high — alert.

### What Is Burn Rate?

Burn rate measures how fast you are consuming your error budget relative to the expected consumption rate. A burn rate of 1x means you are on pace to exactly use up the budget. A burn rate of 10x means you are burning budget 10 times faster than expected and will exhaust it in 1/10 of the window.

Alert at high burn rates (e.g., > 5x) to catch problems early.

---

## What You Are Building

A Prometheus + Alertmanager + Grafana stack configured with:
- SLI metrics for a sample application
- SLO-based alerting rules
- Error budget tracking
- Alertmanager routing to a webhook

---

## Starting The Lab

```bash
cd 20-sre-slos-alerts-error-budgets
yeast up
yeast ssh sre
newgrp docker
mkdir -p /home/ubuntu/monitoring && cd /home/ubuntu/monitoring
```

---

## Step 1 — Define A Fake Application With Metrics

Create a simple Python app that exposes Prometheus metrics and randomly generates errors and latency:

```bash
sudo pip3 install -q prometheus_client

mkdir -p /home/ubuntu/app && cat > /home/ubuntu/app/app.py << 'PYEOF'
#!/usr/bin/env python3
import time, random, threading
from prometheus_client import Counter, Histogram, start_http_server

REQUEST_COUNT = Counter(
    'http_requests_total',
    'Total HTTP requests',
    ['method', 'status']
)
REQUEST_LATENCY = Histogram(
    'http_request_duration_seconds',
    'HTTP request latency',
    buckets=[0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]
)

def simulate_traffic():
    while True:
        for _ in range(random.randint(10, 50)):
            latency = random.expovariate(1/0.15)  # mean 150ms
            REQUEST_LATENCY.observe(latency)

            # ~1% error rate normally, occasionally spike to 5%
            error_rate = 0.05 if random.random() < 0.05 else 0.01
            if random.random() < error_rate:
                REQUEST_COUNT.labels(method='GET', status='500').inc()
            else:
                REQUEST_COUNT.labels(method='GET', status='200').inc()

        time.sleep(1)

start_http_server(8000)
print("App metrics on :8000/metrics")
threading.Thread(target=simulate_traffic, daemon=True).start()

# Keep running
while True:
    time.sleep(60)
PYEOF

nohup python3 /home/ubuntu/app/app.py > /home/ubuntu/app/app.log 2>&1 &
sleep 2
curl http://localhost:8000/metrics | grep http_requests
```

---

## Step 2 — SLO Alert Rules

Create the SLO rules file:

```bash
mkdir -p rules

cat > rules/slo.yml << 'EOF'
groups:
  - name: slo_rules
    interval: 30s
    rules:

      # SLI: availability = successful requests / total requests
      - record: slo:availability:ratio_rate5m
        expr: |
          sum(rate(http_requests_total{status=~"2.."}[5m]))
          /
          sum(rate(http_requests_total[5m]))

      # SLI: error rate
      - record: slo:error_rate:ratio_rate5m
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m]))
          /
          sum(rate(http_requests_total[5m]))

      # Error budget consumed (over 30 days, SLO = 99.9%)
      # Budget = 0.1% * 30 days * 24h * 60m = 43.2 minutes
      - record: slo:error_budget_remaining:ratio
        expr: |
          1 - (
            sum(increase(http_requests_total{status=~"5.."}[30d]))
            /
            sum(increase(http_requests_total[30d]))
          ) / 0.001

  - name: slo_alerts
    rules:

      # Page: high burn rate — burning budget 5x faster than expected
      - alert: SLOHighErrorBurnRate
        expr: slo:error_rate:ratio_rate5m > (5 * 0.001)
        for: 2m
        labels:
          severity: critical
          slo: availability
        annotations:
          summary: "High error burn rate ({{ $value | humanizePercentage }})"
          description: "Error rate {{ $value | humanizePercentage }} exceeds 5x burn rate threshold. Error budget will exhaust in ~6 days at this rate."

      # Warn: sustained elevated error rate
      - alert: SLOElevatedErrorRate
        expr: slo:error_rate:ratio_rate5m > 0.001
        for: 5m
        labels:
          severity: warning
          slo: availability
        annotations:
          summary: "Error rate above SLO threshold"
          description: "Error rate {{ $value | humanizePercentage }} has been above 0.1% for 5 minutes."

      # Page: latency SLO breach (p99 above 500ms)
      - alert: SLOHighLatency
        expr: |
          histogram_quantile(0.99,
            sum(rate(http_request_duration_seconds_bucket[5m]))
            by (le)
          ) > 0.5
        for: 3m
        labels:
          severity: warning
          slo: latency
        annotations:
          summary: "p99 latency above 500ms"
          description: "p99 latency is {{ $value | humanizeDuration }}."
EOF
```

---

## Step 3 — Prometheus Config With Alertmanager

```bash
cat > prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 30s

alerting:
  alertmanagers:
    - static_configs:
        - targets: ["alertmanager:9093"]

rule_files:
  - "rules/*.yml"

scrape_configs:
  - job_name: "app"
    static_configs:
      - targets: ["172.17.0.1:8000"]  # host network from container
EOF
```

---

## Step 4 — Alertmanager Config

```bash
cat > alertmanager.yml << 'EOF'
global:
  resolve_timeout: 5m

route:
  group_by: ["alertname", "slo"]
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h
  receiver: "webhook"

  routes:
    - match:
        severity: critical
      receiver: webhook
      continue: true

receivers:
  - name: webhook
    webhook_configs:
      - url: "http://172.17.0.1:9999/alert"
        send_resolved: true
EOF
```

---

## Step 5 — Start The Stack

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

  alertmanager:
    image: prom/alertmanager:latest
    container_name: alertmanager
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro
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

volumes:
  prometheus-data:
  grafana-data:
EOF

docker compose up -d
```

---

## Step 6 — Explore The SLO Dashboard

From your laptop, create a tunnel to the SRE VM:

```bash
ssh -N \
  -L 9090:127.0.0.1:9090 \
  -L 3000:127.0.0.1:3000 \
  -L 9093:127.0.0.1:9093 \
  -p 2233 ubuntu@127.0.0.1
```

Keep that tunnel terminal open. Then open Prometheus at `http://localhost:9090`.

Query the SLI recordings:

```
slo:availability:ratio_rate5m
slo:error_rate:ratio_rate5m
slo:error_budget_remaining:ratio
```

Query the raw p99 latency:
```
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
```

In Alerts tab, you will see `SLOElevatedErrorRate` firing occasionally when the app's random error spikes trigger it.

---

## Step 7 — Trigger An Alert On Purpose

Modify the app to generate a high error rate:

```bash
cat > /home/ubuntu/app/app.py << 'PYEOF'
#!/usr/bin/env python3
import time, random, threading
from prometheus_client import Counter, Histogram, start_http_server

REQUEST_COUNT = Counter('http_requests_total', 'Total HTTP requests', ['method', 'status'])
REQUEST_LATENCY = Histogram('http_request_duration_seconds', 'Request latency',
    buckets=[0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0])

def simulate_traffic():
    while True:
        for _ in range(random.randint(10, 50)):
            REQUEST_LATENCY.observe(random.expovariate(1/0.2))
            # 10% error rate — clearly above SLO
            if random.random() < 0.10:
                REQUEST_COUNT.labels(method='GET', status='500').inc()
            else:
                REQUEST_COUNT.labels(method='GET', status='200').inc()
        time.sleep(1)

start_http_server(8000)
threading.Thread(target=simulate_traffic, daemon=True).start()
while True:
    time.sleep(60)
PYEOF

pkill -f "app/app.py" 2>/dev/null || true
nohup python3 /home/ubuntu/app/app.py > /home/ubuntu/app/app.log 2>&1 &
```

Wait 2–3 minutes. In Prometheus Alerts, `SLOHighErrorBurnRate` should fire — the error rate is 10x the threshold.

---

## Step 8 — Reading Error Budget

Query the error budget remaining:

```
slo:error_budget_remaining:ratio
```

When the error rate spikes, this value drops. When errors stop, it recovers slowly over the 30-day window. This is your budget — when it hits zero, you freeze risky changes.

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
docker compose down -v
exit
yeast destroy
```

---

## Quick Recap

In Lab 20 — SRE: SLOs, Alerts, And Error Budgets, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- SLIs: what to measure — user-facing quality signals, not internal metrics
- SLOs: target values over a time window, and why they beat ad-hoc alerting
- Error budgets: the quantified amount of unreliability you can afford
- Burn rate: how fast you are consuming the budget — the key alerting signal
- Alertmanager: routing alerts to destinations, grouping, silencing
- Recording rules: precomputing expensive queries for faster dashboards and alerting
- `histogram_quantile`: computing percentile latency from Prometheus histograms
- Alert quality: the difference between paging on a symptom vs paging on a cause

---

## What Is Next

**Lab 21 — Backup And Restore Drill**

Your database has data. Do you know how to restore it? Have you actually tried? Lab 21 teaches backup strategies, validates them by actually restoring, and gives you the muscle memory for data recovery under pressure.
