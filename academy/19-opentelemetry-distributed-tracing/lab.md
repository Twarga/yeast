# Lab 19 — OpenTelemetry Distributed Tracing

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 3 |
| Minimum VM RAM | 4096 MB |
| SSH ports | 2230, 2231, 2232 |
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
- Ignoring the forwarded port shown by `yeast up` or `yeast status`, or opening a tunnel when the lab already gave you a forwarded host port.
- Skipping validation because the final page or command "looked fine".
- Forgetting to run `yeast destroy` before moving to the next lab.
- Confusing laptop `localhost`, VM `localhost`, and container `localhost`.
- Opening Grafana, Prometheus, Jaeger, or Argo CD before the forwarded service is ready.

---

## The Story

A user reports that a request took 8 seconds. You check Prometheus — average latency is 120ms. You check the logs — no errors. But something was slow for that user, and you cannot find it.

This is the problem tracing solves. A trace records the entire journey of one request through your system: which services it touched, in what order, how long each step took, where time was spent. When you look at the slow 8-second request's trace, you see immediately: service A took 20ms, service B took 7.8 seconds, and inside service B it was a database query that waited 7.7 seconds for a lock.

Metrics tell you averages. Logs tell you events. Traces tell you the path and duration of individual requests through a distributed system.

---

## Before You Start — Understanding The Concepts

### What Is A Trace?

A trace represents the complete lifecycle of a single request as it travels through your system. It has:

- A **trace ID** — a unique identifier for the entire request journey
- A collection of **spans** — individual units of work within the trace

### What Is A Span?

A span represents a single operation: an HTTP request, a database query, a function call. Each span has:
- A name (e.g., "GET /items", "db.query")
- A start time and duration
- Status (OK or Error)
- Attributes (key-value metadata: HTTP method, DB query, user ID)
- A parent span ID (linking it to the span that called it)

Spans nest. The incoming HTTP request is the root span. It calls service B — that creates a child span. Service B queries the database — another child span. The whole tree is the trace.

### What Is OpenTelemetry?

OpenTelemetry (OTel) is an open standard for collecting telemetry (traces, metrics, logs) from applications. It provides:
- SDKs for every major language (Python, Go, Java, Node.js, etc.)
- A standardized data format (OTLP — OpenTelemetry Protocol)
- The OpenTelemetry Collector — a proxy/router for telemetry data

Before OTel, every tracing system had its own SDK. You picked Jaeger and used the Jaeger SDK; then if you switched to Zipkin you had to rewrite your instrumentation. OTel standardizes the instrumentation — you instrument once, export to any backend.

### What Is Instrumentation?

Instrumentation is the code you add to your application to produce telemetry. There are two kinds:

**Manual instrumentation** — you explicitly create spans in your code:
```python
with tracer.start_as_current_span("db.query") as span:
    span.set_attribute("db.statement", sql)
    result = db.execute(sql)
```

**Auto-instrumentation** — OTel's libraries automatically trace common frameworks (Flask, Django, requests, SQLAlchemy) without code changes. You configure it at startup.

### What Is Jaeger?

Jaeger is an open-source distributed tracing backend. It receives traces via OTLP, stores them, and provides a web UI to search and visualize them. For this lab we use Jaeger as the trace backend — it is simple to run and has a great UI.

### What Is Context Propagation?

For distributed tracing to work, the trace ID must travel with the request across service boundaries. When service A calls service B over HTTP, it adds the trace context as HTTP headers (e.g., `traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01`). Service B reads this header, extracts the trace ID, and creates a child span under the same trace.

Without context propagation, each service's span is isolated — they never connect into a single trace.

---

## What You Are Building

```
Your Laptop
    │  HTTP 16686 → tracing port 16686 (Jaeger UI)
    │  HTTP 8001  → svc-a port 8001
    │  SSH  2230  → tracing port 22
    │  SSH  2231  → svc-a port 22
    │  SSH  2232  → svc-b port 22
    ▼
┌───────────────────────────────────────────────────────────┐
│  Private Network: 192.168.80.0/24                         │
│                                                           │
│  ┌────────────────────────────┐                           │
│  │  tracing (.10)             │                           │
│  │  Jaeger :16686 (UI)        │                           │
│  │  Jaeger OTLP :4317 (gRPC)  │◀── spans from services   │
│  │  Jaeger OTLP :4318 (HTTP)  │                           │
│  └────────────────────────────┘                           │
│                                                           │
│  ┌──────────────┐  calls  ┌──────────────┐               │
│  │  svc-a (.21) │────────▶│  svc-b (.22) │               │
│  │  :8001       │         │  :8002       │               │
│  └──────────────┘         └──────────────┘               │
│  propagates trace context via HTTP headers                │
└───────────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 19-opentelemetry-distributed-tracing
yeast up
```

---

## Step 1 — Start Jaeger

```bash
yeast ssh tracing
newgrp docker

docker run -d \
  --name jaeger \
  --restart unless-stopped \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest

docker ps
curl -s http://localhost:16686/api/services
exit
```

Jaeger `all-in-one` bundles the collector, store, and UI in a single container. In production you would deploy them separately. For this lab, one container is enough.

---

## Step 2 — Instrument Service B

Service B is the backend — it receives calls from A and simulates a database query.

```bash
yeast ssh svc-b

sudo pip3 install -q opentelemetry-sdk opentelemetry-exporter-otlp-proto-http opentelemetry-instrumentation-urllib3

cat > /home/ubuntu/svc_b.py << 'PYEOF'
#!/usr/bin/env python3
import json
import random
import time
from http.server import HTTPServer, BaseHTTPRequestHandler

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator
from opentelemetry import propagate

# Configure tracing
resource = Resource.create({"service.name": "svc-b"})
provider = TracerProvider(resource=resource)
exporter = OTLPSpanExporter(endpoint="http://192.168.80.10:4318/v1/traces")
provider.add_span_processor(BatchSpanProcessor(exporter))
trace.set_tracer_provider(provider)
tracer = trace.get_tracer("svc-b")

propagate.set_global_textmap(TraceContextTextMapPropagator())

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        # Extract trace context from incoming request headers
        ctx = propagate.extract(dict(self.headers))

        with tracer.start_as_current_span("svc-b.handle", context=ctx) as span:
            span.set_attribute("http.method", "GET")
            span.set_attribute("http.path", self.path)

            # Simulate a database query
            with tracer.start_as_current_span("db.query") as db_span:
                db_span.set_attribute("db.statement", "SELECT * FROM items LIMIT 10")
                sleep_ms = random.randint(5, 200)
                time.sleep(sleep_ms / 1000.0)
                db_span.set_attribute("db.rows_returned", random.randint(1, 10))

            body = json.dumps({
                "service": "svc-b",
                "data": "some data",
                "trace_id": format(span.get_span_context().trace_id, '032x')
            }).encode()

            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    def log_message(self, *a):
        pass

print("Service B listening on :8002")
HTTPServer(("0.0.0.0", 8002), Handler).serve_forever()
PYEOF

nohup python3 /home/ubuntu/svc_b.py > /home/ubuntu/svc_b.log 2>&1 &
sleep 2
curl http://localhost:8002
exit
```

---

## Step 3 — Instrument Service A

Service A is the entry point — it calls service B.

```bash
yeast ssh svc-a

sudo pip3 install -q opentelemetry-sdk opentelemetry-exporter-otlp-proto-http opentelemetry-instrumentation-urllib3

cat > /home/ubuntu/svc_a.py << 'PYEOF'
#!/usr/bin/env python3
import json
import urllib.request
from http.server import HTTPServer, BaseHTTPRequestHandler

from opentelemetry import trace, propagate
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

resource = Resource.create({"service.name": "svc-a"})
provider = TracerProvider(resource=resource)
exporter = OTLPSpanExporter(endpoint="http://192.168.80.10:4318/v1/traces")
provider.add_span_processor(BatchSpanProcessor(exporter))
trace.set_tracer_provider(provider)
tracer = trace.get_tracer("svc-a")
propagate.set_global_textmap(TraceContextTextMapPropagator())

SVC_B_URL = "http://192.168.80.22:8002"

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        with tracer.start_as_current_span("svc-a.handle") as span:
            span.set_attribute("http.method", "GET")
            span.set_attribute("http.path", self.path)

            # Inject trace context into outgoing call to svc-b
            with tracer.start_as_current_span("http.call.svc-b") as call_span:
                headers = {}
                propagate.inject(headers)  # adds traceparent header
                req = urllib.request.Request(SVC_B_URL, headers=headers)
                try:
                    with urllib.request.urlopen(req, timeout=5) as resp:
                        upstream_data = json.loads(resp.read())
                    call_span.set_attribute("http.status_code", 200)
                except Exception as e:
                    call_span.record_exception(e)
                    call_span.set_status(trace.StatusCode.ERROR)
                    upstream_data = {"error": str(e)}

            body = json.dumps({
                "service": "svc-a",
                "upstream": upstream_data,
                "trace_id": format(span.get_span_context().trace_id, '032x')
            }).encode()

            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    def log_message(self, *a):
        pass

print("Service A listening on :8001")
HTTPServer(("0.0.0.0", 8001), Handler).serve_forever()
PYEOF

nohup python3 /home/ubuntu/svc_a.py > /home/ubuntu/svc_a.log 2>&1 &
sleep 2
curl http://localhost:8001
exit
```

---

## Step 4 — Generate Traces

Make several requests to service A from your laptop:

```bash
for i in $(seq 10); do
  curl -s http://localhost:8001 | python3 -m json.tool
  sleep 0.5
done
```

Each response includes a `trace_id`. Copy one of them.

---

## Step 5 — Explore Traces In Jaeger UI

Yeast now forwards the Jaeger UI directly to your laptop. Open `http://127.0.0.1:16686` in your browser.

### Search For Traces

1. In the "Service" dropdown, select `svc-a`
2. Click "Find Traces"
3. You will see the 10 traces from your requests

### View A Trace

Click on one trace. You see a waterfall diagram:

```
svc-a.handle           ████████████████████████  (total: ~220ms)
  http.call.svc-b         ████████████████████  (~200ms)
    svc-b.handle           ████████████████████ (~200ms)
      db.query                ████████████████  (~180ms)
```

Each bar shows a span. The indentation shows the parent-child relationship. The width shows duration. You can immediately see: the database query inside svc-b is taking the most time.

### View Span Details

Click on any span. You see its attributes:
- `http.method`, `http.path`
- `db.statement`, `db.rows_returned`
- Start time, duration
- Any errors

This is the information you need to understand exactly what happened and where.

### Compare Traces

Make one slow request:

```bash
yeast ssh svc-b
# Modify the app to simulate a slow query
kill $(pgrep -f svc_b.py)
sed -i 's/random.randint(5, 200)/random.randint(500, 2000)/' /home/ubuntu/svc_b.py
nohup python3 /home/ubuntu/svc_b.py > /home/ubuntu/svc_b.log 2>&1 &
exit

curl -s http://localhost:8001
```

In Jaeger, search for the latest trace. The db.query span is now 500–2000ms — you can see exactly where the slowdown is.

---

## Step 6 — Understanding What You Cannot See Without Tracing

Without tracing you would only know:
- Service A responded in 2 seconds (from its logs)
- The request did not error (status 200)

You would not know:
- Service A itself took 5ms
- The call to service B took 1.995 seconds
- Inside service B, the database query took 1.990 seconds

Tracing gives you the causal chain and duration breakdown — essential for diagnosing latency in distributed systems.

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

In Lab 19 — OpenTelemetry Distributed Tracing, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What a trace is: the complete record of one request's journey through a distributed system
- What a span is: a single operation with start time, duration, and attributes
- Why you need tracing: metrics give averages, logs give events, traces give causality and duration
- OpenTelemetry: the standard SDK for producing vendor-neutral telemetry
- Context propagation: how trace IDs travel between services via HTTP headers (`traceparent`)
- Manual instrumentation: `tracer.start_as_current_span()`, setting attributes
- `propagate.inject()` and `propagate.extract()`: injecting and extracting context across service calls
- Jaeger: searching traces, reading waterfall diagrams, comparing fast and slow traces
- The diagnosis pattern: metrics flag the problem, traces identify the cause

---

## What Is Next

**Lab 20 — SRE: SLOs, Alerts, And Error Budgets**

You have metrics, logs, and traces. Lab 20 teaches you how to use them to define what "good enough" looks like — SLOs (Service Level Objectives) — and how to build alerts that page you only when it matters, not on every blip.
