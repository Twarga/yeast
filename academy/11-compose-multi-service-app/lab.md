# Lab 11 — Compose Multi-Service App

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner to intermediate |
| Estimated time | 60-90 minutes |
| VMs | 1 |
| Minimum VM RAM | 2048 MB |
| SSH ports | 2217 |
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

---

## The Story

In Lab 10 you ran containers one at a time. A real application is not one container — it is several. A Python app, a PostgreSQL database, maybe a Redis cache, an Nginx proxy in front of all of it. Starting each one manually with `docker run`, wiring up the networks and volumes by hand, remembering which environment variables go where — that becomes unmanageable fast.

Docker Compose is the answer. You describe your entire application stack in a single YAML file — `compose.yaml` — and bring it all up with one command. Services discover each other by name. Volumes are defined once. Environment variables are loaded from `.env`. The whole stack starts, stops, and scales as a unit.

This is how virtually every development environment works today. And it is the same mental model as Kubernetes (which you will use in Lab 27) — just simpler and local.

---

## Before You Start — Understanding The Concepts

### What Is Docker Compose?

Docker Compose is a tool for defining and running multi-container applications. It reads a `compose.yaml` file that describes your services, their images, environment, ports, volumes, and dependencies.

Key commands:
- `docker compose up -d` — start all services in the background
- `docker compose down` — stop and remove containers (volumes survive by default)
- `docker compose down -v` — stop, remove containers AND volumes
- `docker compose ps` — show status of services
- `docker compose logs <service>` — read a service's logs
- `docker compose exec <service> <cmd>` — run a command inside a running service
- `docker compose restart <service>` — restart one service

### What Is Service Discovery?

In a Compose stack, each service is reachable by its service name as a hostname. If your database service is named `db`, the app connects to `db:5432`. If your cache is named `redis`, the app connects to `redis:6379`.

Docker handles the DNS automatically. You never need to hardcode IP addresses.

### What Is `depends_on`?

`depends_on` tells Compose which services must start before this one. `app: depends_on: [db]` means Docker starts `db` first, then `app`. But it only waits for the container to start — not for the service inside to be ready. The database might take a few seconds to initialize. This is why apps need retry logic for the initial database connection.

### What Is A Compose Network?

Compose creates a default network for each project. All services in the same `compose.yaml` are automatically on the same network and can reach each other by service name.

You can also define named networks and control which services join which network — useful for isolating the database tier from the proxy tier.

### Environment Variables In Compose

Compose reads a `.env` file in the same directory as `compose.yaml`. Values in `.env` are available as variables in the YAML using `${VAR}` syntax. This is how you inject secrets without putting them in the compose file itself.

---

## What You Are Building

```
Your Laptop
    │
    │  HTTP port 8080 → proxy container port 80
    │  SSH  port 2217 → VM port 22
    ▼
┌──────────────────────────────────────────────────────────┐
│  compose VM                                              │
│                                                          │
│  ┌──────────┐    ┌──────────────┐    ┌───────────────┐  │
│  │  proxy   │───▶│  app         │───▶│  db           │  │
│  │  nginx   │    │  python:8000 │    │  postgres:5432│  │
│  │  :80     │    │              │    │               │  │
│  └──────────┘    └──────────────┘    └───────────────┘  │
│                                                          │
│  All on compose internal network                        │
│  Volume: pgdata (postgres data)                         │
└──────────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 11-compose-multi-service-app
yeast up
yeast ssh compose
newgrp docker
```

---

## Creating The Application

Create the project directory:

```bash
mkdir -p /home/ubuntu/app
cd /home/ubuntu/app
```

### The Python App

```bash
cat > app.py << 'EOF'
#!/usr/bin/env python3
import json
import os
import time
import psycopg2
import psycopg2.extras
from http.server import HTTPServer, BaseHTTPRequestHandler

DB_CONFIG = {
    "host":     os.environ["DB_HOST"],
    "port":     int(os.environ.get("DB_PORT", "5432")),
    "dbname":   os.environ["DB_NAME"],
    "user":     os.environ["DB_USER"],
    "password": os.environ["DB_PASS"],
}

def get_conn():
    return psycopg2.connect(**DB_CONFIG)

def wait_for_db(retries=10, delay=2):
    for i in range(retries):
        try:
            conn = get_conn()
            conn.close()
            print("Database ready")
            return
        except Exception as e:
            print(f"Waiting for database... ({i+1}/{retries}): {e}")
            time.sleep(delay)
    raise RuntimeError("Database never became ready")

def init_db():
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute("""
                CREATE TABLE IF NOT EXISTS items (
                    id SERIAL PRIMARY KEY,
                    name TEXT NOT NULL,
                    created_at TIMESTAMP DEFAULT NOW()
                )
            """)
        conn.commit()

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/healthz":
            self._respond(200, {"status": "ok"})
            return
        try:
            with get_conn() as conn:
                with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                    cur.execute("SELECT * FROM items ORDER BY id")
                    rows = cur.fetchall()
            self._respond(200, {"items": [dict(r) for r in rows]})
        except Exception as e:
            self._respond(500, {"error": str(e)})

    def do_POST(self):
        if self.path != "/items":
            self._respond(404, {"error": "not found"})
            return
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length))
        try:
            with get_conn() as conn:
                with conn.cursor() as cur:
                    cur.execute("INSERT INTO items (name) VALUES (%s) RETURNING id", (body["name"],))
                    row_id = cur.fetchone()[0]
                conn.commit()
            self._respond(201, {"id": row_id, "name": body["name"]})
        except Exception as e:
            self._respond(500, {"error": str(e)})

    def _respond(self, code, data):
        body = json.dumps(data).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        print(f"{self.address_string()} - {fmt % args}")

if __name__ == "__main__":
    wait_for_db()
    init_db()
    print("App listening on :8000")
    HTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
EOF
```

### The Dockerfile

The app needs a container image. A `Dockerfile` describes how to build it:

```bash
cat > Dockerfile << 'EOF'
FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir psycopg2-binary

COPY app.py .

EXPOSE 8000

CMD ["python3", "app.py"]
EOF
```

Let's read every line:

**`FROM python:3.11-slim`** — the base image. `python:3.11-slim` is the official Python 3.11 image, "slim" variant (smaller than the full image, no extra tools). Every image builds on top of a base.

**`WORKDIR /app`** — set the working directory inside the container. All subsequent commands run from here.

**`RUN pip install --no-cache-dir psycopg2-binary`** — run a command during the image build. This installs the PostgreSQL driver. `--no-cache-dir` prevents pip from caching packages, keeping the image smaller.

**`COPY app.py .`** — copy `app.py` from the build context (your current directory) into `/app` in the image.

**`EXPOSE 8000`** — documentation: this container listens on port 8000. Does not actually open a port — that is done with `-p` or in the compose file.

**`CMD ["python3", "app.py"]`** — the default command to run when the container starts. Uses exec form (array) not shell form (string) so signals are delivered directly to the process.

Build the image:

```bash
docker build -t myapp:latest .
```

`-t myapp:latest` tags the image with the name `myapp` and tag `latest`. Watch the build output — you can see each layer being built.

### The Nginx Config

```bash
cat > nginx.conf << 'EOF'
upstream app {
    server app:8000;
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass         http://app;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
    }

    location /healthz {
        proxy_pass http://app;
    }
}
EOF
```

`app:8000` — `app` is the service name from `compose.yaml`. On the Compose network, Nginx resolves `app` to the app container's IP automatically.

### The Environment File

```bash
cat > .env << 'EOF'
DB_HOST=db
DB_PORT=5432
DB_NAME=appdb
DB_USER=appuser
DB_PASS=composelab11
POSTGRES_DB=appdb
POSTGRES_USER=appuser
POSTGRES_PASSWORD=composelab11
EOF

chmod 600 .env
```

Two sets of variables: `DB_*` for the app, `POSTGRES_*` for the official PostgreSQL image's initialization.

### The Compose File

```bash
cat > compose.yaml << 'EOF'
services:

  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB:       ${POSTGRES_DB}
      POSTGRES_USER:     ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 5s
      timeout: 3s
      retries: 10
    restart: unless-stopped

  app:
    image: myapp:latest
    environment:
      DB_HOST: ${DB_HOST}
      DB_PORT: ${DB_PORT}
      DB_NAME: ${DB_NAME}
      DB_USER: ${DB_USER}
      DB_PASS: ${DB_PASS}
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped

  proxy:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
    depends_on:
      - app
    restart: unless-stopped

volumes:
  pgdata:
EOF
```

Let's read the key parts:

**`image: postgres:15-alpine`** — use the official PostgreSQL 15 image, Alpine variant (very small). No Dockerfile needed — the image is pre-built on Docker Hub.

**`volumes: pgdata:/var/lib/postgresql/data`** — mount the named volume `pgdata` at the path where Postgres stores its data files. The data survives `docker compose down` (but not `docker compose down -v`).

**`healthcheck`** — Compose polls this command to determine if the service is healthy. `pg_isready` returns 0 when Postgres is ready to accept connections. The `app` service will not start until `db` is healthy — that is `condition: service_healthy` in `depends_on`.

**`restart: unless-stopped`** — automatically restart the container if it exits, unless you explicitly stopped it.

**`volumes: - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro`** — bind mount: the `nginx.conf` from the current directory on the host is mounted read-only inside the proxy container. Changes to the file on the host appear in the container instantly (though Nginx still needs a reload to pick them up).

**`volumes: pgdata:`** at the bottom — this declares the named volume. Named volumes must be declared in this top-level `volumes:` section.

---

## Starting The Stack

```bash
docker compose up -d
```

Watch what happens:

```bash
docker compose ps
```

Wait for all three services to show `running (healthy)` or `running`. The `db` service takes a few seconds to initialize.

Follow the logs to watch startup:

```bash
docker compose logs -f
```

You should see:
- `db` initializing and becoming ready
- `app` printing "Waiting for database... Database ready" then "App listening on :8000"
- `proxy` starting nginx

Press `Ctrl+C` to stop following logs.

---

## Testing The Stack

From inside the VM:

```bash
curl http://localhost/healthz
curl http://localhost/items
```

From your laptop, first create an SSH tunnel to the Compose VM:

```bash
ssh -N -L 8080:127.0.0.1:8080 -p 2217 ubuntu@127.0.0.1
```

Keep that tunnel terminal open, then test from your laptop:

```bash
curl http://localhost:8080/items
curl -X POST http://localhost:8080/items \
  -H "Content-Type: application/json" \
  -d '{"name": "first compose item"}'
curl http://localhost:8080/items
```

You have a full app → proxy → database stack running from one compose file.

---

## Service Logs

```bash
# All services
docker compose logs

# Just one service, follow
docker compose logs -f app

# Last 20 lines of db
docker compose logs --tail 20 db
```

---

## Exec Into A Service

```bash
# Shell in the app container
docker compose exec app bash

# Run a command in the db container
docker compose exec db psql -U appuser -d appdb -c "SELECT * FROM items;"
```

No SSH required — `docker compose exec` opens a shell or runs a command in a running service container.

---

## Redeploying After A Change

Change the app — add a `/status` endpoint:

```bash
cat >> app.py << 'PYEOF'

# (Already handled in do_GET via /healthz — this simulates a code change)
PYEOF
```

Rebuild the image and restart just the app service:

```bash
docker build -t myapp:latest .
docker compose up -d --no-deps app
```

`--no-deps` means "restart only this service, not its dependencies." This is the redeploy pattern for iterating on one service without disturbing the database.

---

## Data Persistence Across Restarts

Create some data:

```bash
curl -X POST http://localhost/items -H "Content-Type: application/json" -d '{"name":"test"}'
```

Stop and recreate the stack:

```bash
docker compose down
docker compose up -d
```

Check the data:

```bash
curl http://localhost/items
```

The item is still there. The `pgdata` volume persisted across the `down/up` cycle.

Now destroy the volumes too:

```bash
docker compose down -v
docker compose up -d
curl http://localhost/items
```

Empty — `{"items": []}`. The `-v` flag removes volumes. This is a complete reset.

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

In Lab 11 — Compose Multi-Service App, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What Docker Compose is and why single-container `docker run` does not scale
- `compose.yaml` structure: services, volumes, networks
- Service discovery by name: `app`, `db`, `proxy` resolve by hostname inside the stack
- `depends_on` with `condition: service_healthy` — proper startup ordering
- Healthchecks: how Compose knows a service is actually ready, not just started
- `restart: unless-stopped` — automatic recovery from crashes
- `.env` files and `${VAR}` substitution in compose files
- Named volumes vs bind mounts in the compose context
- `docker compose logs`, `exec`, `up -d`, `down`, `down -v`
- `--no-deps` for redeploying one service without touching the others
- The difference between `down` (removes containers) and `down -v` (also removes volumes)

---

## What Is Next

**Lab 12 — Container Build, Scan, And Hardening**

You built a Dockerfile in this lab. Lab 12 goes deeper: what makes a container image safe and small? You will learn Dockerfile best practices, build smaller images, run containers as non-root users, scan for known vulnerabilities with Trivy, and understand what an SBOM is and why it matters.
