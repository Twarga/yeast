# Lab 04 — Database-Backed App

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner |
| Estimated time | 45-75 minutes |
| VMs | 3 |
| Minimum VM RAM | 3072 MB |
| SSH ports | 2205, 2206, 2207 |
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
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you. If not, the lab will tell you when to use a manual SSH tunnel.

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

---

## The Story

The backend app from Lab 03 always returned the same hardcoded response. That is fine for a health check endpoint, but real applications need to read and write data that persists between requests. Users create accounts. Orders get placed. Blog posts get written. All of that lives in a database.

Today you add a third VM running PostgreSQL. The app VM connects to it over the private network, reads and writes records, and serves that data through the proxy. You will also learn what happens when the database connection breaks — because in production, it always breaks eventually, and knowing how to diagnose it quickly is the skill that matters.

---

## Before You Start — Understanding The Concepts

### What Is A Database?

A database is a program that stores and retrieves structured data. It persists data to disk so it survives restarts. It handles concurrent access — multiple application instances reading and writing at the same time without corrupting each other. It enforces rules about data structure, types, and relationships.

Almost every application that does anything useful has a database behind it.

### What Is PostgreSQL?

PostgreSQL (often called Postgres) is one of the most widely used open-source relational databases. "Relational" means data is organized into tables with rows and columns, and tables can reference each other. You interact with it using SQL (Structured Query Language).

Postgres is known for being reliable, standards-compliant, and feature-rich. It runs as a service on Linux, listens on port 5432 by default, and manages access through users and databases.

### What Is SQL?

SQL (Structured Query Language) is the language you use to work with relational databases. Basic operations:

```sql
-- Create a table
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert a row
INSERT INTO items (name) VALUES ('first item');

-- Read rows
SELECT * FROM items;

-- Delete a row
DELETE FROM items WHERE id = 1;
```

You do not need to be a SQL expert for this lab — we use a very small subset.

### What Is A Database Connection?

When your application wants to talk to a database, it opens a "connection" — a persistent TCP socket to the database server on port 5432. The database authenticates the connection (checks username and password) and then keeps the connection open, ready to execute queries.

Managing connections efficiently is one of the trickier aspects of application development. Too many open connections overwhelm the database. Too few and you create a bottleneck. Connection pooling libraries handle this automatically in production apps — but for this lab we use simple direct connections so you can see what is happening.

### What Is pg_isready?

`pg_isready` is a command that ships with PostgreSQL client tools. It connects to a Postgres instance and checks if it is ready to accept connections. It returns a useful exit code:
- `0` — accepting connections
- `1` — rejecting connections (started but not ready)
- `2` — no response

It is the canonical tool for checking database connectivity from a script.

### What Is psql?

`psql` is the PostgreSQL interactive terminal. You use it to connect to a database and run SQL queries. You can run it from the command line to test queries, inspect schema, check data, and diagnose problems.

---

## What You Are Building

```
Your Laptop
    │
    │  HTTP port 9080 → proxy port 80
    │  SSH  port 2205 → proxy port 22
    │  SSH  port 2206 → app port 22
    │  SSH  port 2207 → db port 22
    ▼
┌──────────────────────────────────────────────────────────────┐
│  Private Network: 192.168.20.0/24                            │
│                                                              │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────────┐ │
│  │  proxy      │    │  app         │    │  db             │ │
│  │  .10        │───▶│  .20         │───▶│  .30            │ │
│  │             │    │              │    │                  │ │
│  │  Nginx :80  │    │  Python :8000│    │  PostgreSQL :5432│ │
│  └─────────────┘    └──────────────┘    └─────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

Three VMs, one private network:
- **proxy** — receives HTTP from outside, forwards to app
- **app** — the Python application, talks to the database
- **db** — PostgreSQL, only reachable from within the private network

Note: The database has no external port mapping. There is no way to reach it directly from your laptop. Only the app VM can connect to it over the private network. This is correct security posture — databases should never be directly internet-facing.

---

## The Config File — assets/yeast.yaml

Three instances on one private network. Each gets a static IP. The db VM only allows SSH and port 5432 in UFW — no HTTP port at all.

Note the `ssh_port` values: 2205, 2206, 2207 — different from previous labs to avoid port conflicts if you ever run labs simultaneously.

---

## Starting The Lab

```bash
cd 04-database-backed-app
yeast up
```

All three VMs boot and get provisioned. Check they are all running:

```bash
yeast status
```

Expected:

```
NAME    STATE    SSH PORT
proxy   running  2205
app     running  2206
db      running  2207
```

---

## Set Up PostgreSQL

SSH into the database VM:

```bash
yeast ssh db
```

Verify PostgreSQL is running:

```bash
sudo systemctl status postgresql
sudo ss -tlnp | grep 5432
```

PostgreSQL listens on `127.0.0.1:5432` by default — only on the loopback. We need to change this so the app VM can connect over the private network.

### Allow network connections

Edit the PostgreSQL config to listen on all interfaces:

```bash
sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" \
  /etc/postgresql/14/main/postgresql.conf
```

This changes `listen_addresses` from `localhost` (loopback only) to `*` (all interfaces including the private network one).

Now tell PostgreSQL which clients are allowed to connect. Edit `pg_hba.conf` (the host-based authentication file):

```bash
sudo tee -a /etc/postgresql/14/main/pg_hba.conf << 'EOF'
# Allow app VM to connect
host    appdb    appuser    192.168.20.20/32    md5
EOF
```

This line reads: for TCP connections (`host`) to database `appdb` by user `appuser` from IP `192.168.20.20` (the app VM), use `md5` password authentication.

### Create the database and user

Switch to the `postgres` system user, which is the database superuser:

```bash
sudo -u postgres psql
```

You are now in the `psql` interactive terminal. Run these SQL commands:

```sql
CREATE USER appuser WITH PASSWORD 'changeme123';
CREATE DATABASE appdb OWNER appuser;
GRANT ALL PRIVILEGES ON DATABASE appdb TO appuser;
\q
```

`\q` exits psql.

### Restart PostgreSQL to apply config changes

```bash
sudo systemctl restart postgresql
sudo ss -tlnp | grep 5432
```

Now you should see Postgres listening on `0.0.0.0:5432` (all interfaces), not just `127.0.0.1`.

### Verify from the DB VM itself

```bash
pg_isready -h localhost -p 5432
```

Expected: `localhost:5432 - accepting connections`

Exit the db VM:

```bash
exit
```

---

## Set Up The Application

SSH into the app VM:

```bash
yeast ssh app
```

Install the Python PostgreSQL driver:

```bash
sudo pip3 install psycopg2-binary
```

`psycopg2` is the standard Python library for connecting to PostgreSQL. The `-binary` variant includes pre-compiled native code so you do not need to install build tools.

Create the application:

```bash
mkdir -p /home/ubuntu/app
cat > /home/ubuntu/app/server.py << 'EOF'
#!/usr/bin/env python3
import json
import os
import psycopg2
import psycopg2.extras
from http.server import HTTPServer, BaseHTTPRequestHandler

DB_CONFIG = {
    "host":     os.getenv("DB_HOST", "192.168.20.30"),
    "port":     int(os.getenv("DB_PORT", "5432")),
    "dbname":   os.getenv("DB_NAME", "appdb"),
    "user":     os.getenv("DB_USER", "appuser"),
    "password": os.getenv("DB_PASS", "changeme123"),
}

def get_conn():
    return psycopg2.connect(**DB_CONFIG)

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
    def send_json(self, status, data):
        body = json.dumps(data, default=str).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/items":
            try:
                with get_conn() as conn:
                    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                        cur.execute("SELECT * FROM items ORDER BY created_at DESC")
                        rows = cur.fetchall()
                self.send_json(200, {"items": [dict(r) for r in rows]})
            except Exception as e:
                self.send_json(500, {"error": str(e)})
        else:
            self.send_json(404, {"error": "not found"})

    def do_POST(self):
        if self.path == "/items":
            length = int(self.headers.get("Content-Length", 0))
            body = json.loads(self.rfile.read(length))
            try:
                with get_conn() as conn:
                    with conn.cursor() as cur:
                        cur.execute("INSERT INTO items (name) VALUES (%s) RETURNING id, name, created_at",
                                    (body["name"],))
                        row = cur.fetchone()
                    conn.commit()
                self.send_json(201, {"id": row[0], "name": row[1], "created_at": row[2]})
            except Exception as e:
                self.send_json(500, {"error": str(e)})
        else:
            self.send_json(404, {"error": "not found"})

    def log_message(self, fmt, *args):
        print(f"{self.address_string()} {fmt % args}")

if __name__ == "__main__":
    init_db()
    print("App listening on :8000")
    HTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
EOF
```

A few things worth noting in this code:

**`os.getenv`** — the database credentials are read from environment variables, with defaults. This is the beginning of good secrets hygiene — credentials are not hardcoded in the logic (though for now they still default to visible values; Lab 09 fixes this properly).

**`init_db()`** — on startup, creates the `items` table if it does not already exist. This is a simple migration strategy. In production you would use a proper migrations tool (Alembic, Flyway, etc.), but the concept is the same.

**`%s` in SQL** — parameterized queries. The value for `name` is passed separately from the SQL string, not interpolated directly. This prevents SQL injection — one of the most common and dangerous security vulnerabilities. Always use parameterized queries, never build SQL strings by concatenation.

Create the systemd service:

```bash
sudo tee /etc/systemd/system/app.service << 'EOF'
[Unit]
Description=App Server
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/app
ExecStart=/usr/bin/python3 /home/ubuntu/app/server.py
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now app
```

Check the service started:

```bash
sudo systemctl status app
sudo journalctl -u app --no-pager -n 10
```

You should see: `App listening on :8000`

Test the app connects to the database:

```bash
curl http://localhost:8000/items
```

Expected:

```json
{"items": []}
```

Empty list — no items yet. But the fact that you got a valid response means the app connected to the database and ran the query successfully.

Create an item:

```bash
curl -s -X POST http://localhost:8000/items \
  -H "Content-Type: application/json" \
  -d '{"name": "my first item"}'
```

Expected:

```json
{"id": 1, "name": "my first item", "created_at": "2026-06-15T12:00:00"}
```

Fetch all items:

```bash
curl http://localhost:8000/items
```

Expected:

```json
{"items": [{"id": 1, "name": "my first item", "created_at": "2026-06-15T12:00:00"}]}
```

The data persisted. Exit the app VM:

```bash
exit
```

---

## Configure The Proxy

SSH into the proxy VM:

```bash
yeast ssh proxy
```

Create a proxy config pointing to the app:

```bash
sudo tee /etc/nginx/sites-available/proxy << 'EOF'
upstream app {
    server 192.168.20.20:8000;
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://app;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/proxy /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
exit
```

Test end-to-end from your laptop:

```bash
curl http://localhost:9080/items
curl -X POST http://localhost:9080/items \
  -H "Content-Type: application/json" \
  -d '{"name": "created through proxy"}'
curl http://localhost:9080/items
```

You now have a full three-tier stack running locally: proxy → app → database.

---

## Connecting Directly To The Database

For debugging, it is essential to be able to connect to the database directly and run queries. SSH into the db VM:

```bash
yeast ssh db
sudo -u postgres psql -d appdb
```

List tables:

```sql
\dt
```

Query the items:

```sql
SELECT * FROM items;
```

You will see the rows your app created. This is a critical debugging skill: when something seems wrong with your data, you bypass the application entirely and look at the database directly. It removes one layer of abstraction and tells you immediately whether the problem is in the app or in the data.

Exit psql:

```sql
\q
```

---

## Break Something On Purpose

### Break 1: Stop the database

From the db VM:

```bash
sudo systemctl stop postgresql
```

From your laptop:

```bash
curl http://localhost:9080/items
```

Response:

```json
{"error": "could not connect to server: Connection refused\n\tIs the server running on host \"192.168.20.30\" and accepting\n\tTCP/IP connections on port 5432?"}
```

The application caught the connection error and returned it as a 500 response. The error message tells you exactly what happened — connection refused on port 5432 of the database host.

Check the app logs:

```bash
yeast ssh app
sudo journalctl -u app --no-pager -n 10
```

You will see the same error logged on the app side.

Restart the database:

```bash
yeast ssh db
sudo systemctl start postgresql
exit
```

Verify recovery:

```bash
curl http://localhost:9080/items
```

200 OK. The app reconnects automatically on the next request.

### Break 2: Wrong credentials

SSH into the app VM and change the database password in the service:

```bash
yeast ssh app
sudo systemctl stop app

# Edit the service to pass a wrong password
sudo sed -i '/ExecStart/a Environment=DB_PASS=wrongpassword' /etc/systemd/system/app.service
sudo systemctl daemon-reload
sudo systemctl start app
sleep 2
```

Try a request:

```bash
curl http://localhost:8000/items
```

```json
{"error": "FATAL:  password authentication failed for user \"appuser\""}
```

PostgreSQL rejected the connection — wrong password. This is one of the most common "it worked before" failures: someone rotated a database credential without updating the application's config.

Fix it:

```bash
sudo systemctl stop app
sudo sed -i '/Environment=DB_PASS/d' /etc/systemd/system/app.service
sudo systemctl daemon-reload
sudo systemctl start app
exit
```

### Break 3: Network unreachable

This simulates what happens when a database VM is rebooted, or a network rule blocks access.

From the app VM, block outbound traffic to the DB port:

```bash
yeast ssh app
sudo iptables -A OUTPUT -d 192.168.20.30 -p tcp --dport 5432 -j DROP
```

Try a request:

```bash
curl http://localhost:8000/items
```

This time the request hangs for several seconds before timing out with a connection error. Unlike "connection refused" (instant), a dropped packet causes the TCP stack to wait for a response that never comes.

In production this pattern — instant refusal vs slow timeout — tells you whether the service is down (refused) or the network is broken (timeout).

Remove the block:

```bash
sudo iptables -D OUTPUT -d 192.168.20.30 -p tcp --dport 5432 -j DROP
exit
```

---

## Validate Your Work

```bash
bash assets/validate.sh
```

Note: the validate script uses port 9080 (proxy) for end-to-end checks. Make sure the proxy is configured and Nginx is running.

---

## Clean Up

```bash
yeast destroy
```

---

## Quick Recap

In Lab 04 — Database-Backed App, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What a database does: structured persistent storage with concurrent access
- PostgreSQL basics: service, port 5432, users, databases, `pg_hba.conf`
- `pg_isready`: the right tool to check database connectivity in scripts
- `psql`: direct database access for debugging and administration
- `listen_addresses` and `pg_hba.conf`: how Postgres controls who can connect from where
- Parameterized SQL queries and why they prevent injection attacks
- Reading database credentials from environment variables
- Three-tier architecture: proxy, app, database — and why each layer is separate
- Three types of database connection failure: service down (refused), wrong credentials (auth error), network blocked (timeout)
- How to bypass the application layer and inspect the database directly when debugging

---

## What Is Next

**Lab 05 — Linux Troubleshooting Drill**

You have now built servers, web servers, proxies, apps, and databases. Lab 05 is different: instead of building something new, you will deliberately break a working server in controlled ways and practice diagnosing each failure from logs and system commands. This is the lab that turns "I followed instructions" into "I can debug."
