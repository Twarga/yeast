# Lab 22 — Chaos And Failure Recovery Drill

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 2 |
| Minimum VM RAM | 2048 MB |
| SSH ports | 2236, 2237 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic shell scripting comfort

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

An engineer on your team pushes a config change at 4 PM on a Friday. The database goes down. The app starts returning 500 errors. The proxy serves a blank page. Three things are broken, and nobody has a runbook.

Chaos engineering is the practice of deliberately breaking your system in controlled conditions so you understand how it fails and know exactly how to recover. The goal is not to destroy things — it is to practice recovery so that when real failures happen, you execute from muscle memory, not from panic.

This lab gives you a running three-tier platform and then breaks it in five ways. Before each break, you write the recovery procedure. During the break, you follow the procedure. After recovery, you verify the system is healthy.

---

## Before You Start — Understanding The Concepts

### What Is Chaos Engineering?

Chaos engineering is a discipline that involves experimenting on a system to build confidence in its ability to withstand turbulent conditions in production. It was pioneered at Netflix with Chaos Monkey, which randomly terminated production instances to prove their system could survive it.

You do not need to run chaos in production to benefit. Running it in a controlled lab environment — as you will today — builds the skills and procedures you need when uncontrolled failures happen in production.

### What Is A Runbook?

A runbook is a documented procedure for responding to a specific operational scenario. It answers: "what do I do when X happens?"

A good runbook:
- States the symptoms that trigger it
- Lists the diagnosis steps (what to check first)
- Lists the recovery steps in order
- Includes verification steps (how do you know it worked?)
- Includes escalation criteria (when to call for help)

A bad runbook is one that does not exist, or one that has never been tested.

### Why Practice Under Pressure?

When a real incident happens, you are under pressure. Your hands shake. You mistype commands. You forget steps. You waste time because you have never done this before on this system.

Practicing in a lab, where nothing is on fire, builds procedural memory. The first time you restore a database under pressure should not be during a production incident.

### What Is MTTR?

MTTR (Mean Time To Recovery) is the average time it takes to restore a service after a failure. It is the primary SLO for reliability. Chaos drills directly reduce MTTR by ensuring the recovery procedures are practiced and refined.

---

## What You Are Building

A three-tier platform on two VMs:
- `chaos-proxy` — Nginx reverse proxy
- `chaos-app` — Python app + PostgreSQL

You will break it five ways and recover each one.

```
Your Laptop
    │  HTTP 8880 → chaos-proxy port 80
    │  SSH  2236 → chaos-proxy port 22
    │  SSH  2237 → chaos-app port 22
    ▼
┌────────────────────────────────────────────────────┐
│  Private Network: 192.168.91.0/24                  │
│                                                    │
│  ┌────────────────────┐   ┌─────────────────────┐  │
│  │  chaos-proxy (.10) │──▶│  chaos-app (.20)    │  │
│  │  Nginx :80         │   │  Python app :8000   │  │
│  │                    │   │  PostgreSQL :5432   │  │
│  └────────────────────┘   └─────────────────────┘  │
└────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 22-chaos-failure-recovery-drill
yeast up
```

---

## Step 1 — Build The Platform

### Set Up The Database

```bash
yeast ssh chaos-app

sudo -u postgres psql << 'SQL'
CREATE USER appuser WITH PASSWORD 'chaoslab22';
CREATE DATABASE appdb OWNER appuser;
GRANT ALL PRIVILEGES ON DATABASE appdb TO appuser;
SQL

sudo -u postgres psql -d appdb << 'SQL'
CREATE TABLE items (id SERIAL PRIMARY KEY, name TEXT);
INSERT INTO items (name) VALUES ('alpha'),('beta'),('gamma');
SQL

PG_HBA=$(sudo -u postgres psql -t -c "SHOW hba_file;" | tr -d ' ')
echo "host appdb appuser 127.0.0.1/32 md5" | sudo tee -a "$PG_HBA"
sudo systemctl reload postgresql
```

### Set Up The App

```bash
sudo pip3 install -q psycopg2-binary

cat > /home/ubuntu/app.py << 'PYEOF'
import json, os, psycopg2
from http.server import HTTPServer, BaseHTTPRequestHandler

DB = {"host":"localhost","port":5432,"dbname":"appdb","user":"appuser","password":"chaoslab22"}

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/healthz":
            b = json.dumps({"status":"ok"}).encode(); code = 200
        else:
            try:
                with psycopg2.connect(**DB) as c:
                    with c.cursor() as cur:
                        cur.execute("SELECT id,name FROM items ORDER BY id")
                        rows = [{"id":r[0],"name":r[1]} for r in cur.fetchall()]
                b = json.dumps({"items":rows}).encode(); code = 200
            except Exception as e:
                b = json.dumps({"error":str(e)}).encode(); code = 500
        self.send_response(code)
        self.send_header("Content-Type","application/json")
        self.send_header("Content-Length",str(len(b)))
        self.end_headers()
        self.wfile.write(b)
    def log_message(self,*a): pass

HTTPServer(("0.0.0.0",8000),H).serve_forever()
PYEOF

sudo tee /etc/systemd/system/app.service << 'EOF'
[Unit]
Description=Chaos App
After=postgresql.service

[Service]
User=ubuntu
ExecStart=/usr/bin/python3 /home/ubuntu/app.py
Restart=always

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now app
curl http://localhost:8000/healthz
exit
```

### Set Up The Proxy

```bash
yeast ssh chaos-proxy

sudo tee /etc/nginx/sites-available/app << 'EOF'
server {
    listen 80;
    server_name _;
    location / {
        proxy_pass http://192.168.91.20:8000;
        proxy_set_header Host $host;
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

Verify end-to-end from your laptop:

```bash
curl http://localhost:8880
```

Expected: `{"items":[{"id":1,"name":"alpha"},...]}`

---

## Step 2 — Write Your Runbook First

Before breaking anything, write the recovery procedures. This is critical — doing it under pressure produces garbage.

```bash
yeast ssh chaos-app

cat > /home/ubuntu/runbook.md << 'EOF'
# Chaos Lab 22 — Recovery Runbook

## Break 1: App service stopped
Symptoms: 502 Bad Gateway from proxy, curl to app:8000 fails
Diagnosis: sudo systemctl status app
Recovery: sudo systemctl start app
Verify: curl http://localhost:8000/healthz

## Break 2: Database stopped
Symptoms: app returns 500 errors, {"error":"...connection refused..."}
Diagnosis: sudo systemctl status postgresql
Recovery: sudo systemctl start postgresql
Verify: sudo systemctl is-active postgresql && curl http://localhost:8000/items

## Break 3: Disk full
Symptoms: app writes fail, logs stop, database crashes
Diagnosis: df -h /
Recovery: find and remove large files, then restart affected services
Verify: df -h / && sudo systemctl restart postgresql app

## Break 4: Config file corrupted
Symptoms: service fails to start after restart
Diagnosis: sudo systemctl status app / nginx -t
Recovery: restore config from backup / fix syntax error
Verify: service starts, endpoint responds

## Break 5: Firewall blocks app port
Symptoms: proxy 502, but app locally is fine
Diagnosis: sudo ufw status — look for missing rule
Recovery: sudo ufw allow 8000
Verify: curl from proxy to app:8000
EOF

exit
```

---

## Break 1 — App Service Stopped

```bash
yeast ssh chaos-app
sudo systemctl stop app
exit

curl http://localhost:8880
# Expected: 502 Bad Gateway
```

**Diagnose and recover using your runbook.** Do not read the answer below until you have tried.

```bash
yeast ssh chaos-app
sudo systemctl status app   # shows it is stopped
sudo systemctl start app
curl http://localhost:8000/healthz  # verify local
exit

curl http://localhost:8880  # verify through proxy
```

Time how long this took. Write it down. This is your MTTR for this failure.

---

## Break 2 — Database Stopped

```bash
yeast ssh chaos-app
sudo systemctl stop postgresql
exit

curl http://localhost:8880
# Expected: 500 error — app is up but cannot reach DB
```

Follow your runbook:

```bash
yeast ssh chaos-app
# Symptom check:
curl http://localhost:8000/items  # 500 — connection refused
sudo systemctl status postgresql  # stopped
# Recovery:
sudo systemctl start postgresql
sleep 2
curl http://localhost:8000/items  # 200 — items are back
exit
```

---

## Break 3 — Fill The Disk

```bash
yeast ssh chaos-app
# Fill disk to ~90%
sudo dd if=/dev/zero of=/var/tmp/fill bs=1M count=8000 2>/dev/null || true
df -h /  # should be ~90%+
exit
```

Now generate some requests and watch what happens when the disk is too full for log files:

```bash
for i in $(seq 10); do curl -s http://localhost:8880; done
```

The app may start returning errors. New data cannot be written to the database. Log files stop growing.

Diagnose and recover:

```bash
yeast ssh chaos-app
df -h /           # disk near full
du -sh /var/tmp/* # find the culprit
sudo rm /var/tmp/fill
df -h /           # back to normal
sudo systemctl restart postgresql app
exit
curl http://localhost:8880
```

---

## Break 4 — Corrupt A Config File

```bash
yeast ssh chaos-app
sudo systemctl stop app
# Introduce a syntax error into the service file
sudo sed -i 's/ExecStart=/ExecStart=BROKEN /' /etc/systemd/system/app.service
sudo systemctl daemon-reload
sudo systemctl start app  # will fail
```

Diagnose:

```bash
sudo systemctl status app  # shows failed
sudo journalctl -u app --no-pager -n 10  # shows "BROKEN" in ExecStart
```

Recover:

```bash
sudo sed -i 's/ExecStart=BROKEN /ExecStart=/' /etc/systemd/system/app.service
sudo systemctl daemon-reload
sudo systemctl start app
sudo systemctl is-active app
curl http://localhost:8000/healthz
exit
```

---

## Break 5 — Firewall Blocks The App Port

```bash
yeast ssh chaos-app
sudo ufw deny 8000
sudo ufw reload
exit

curl http://localhost:8880
# Expected: 502 — proxy cannot reach app even though app is running
```

This is subtle: the app is running fine, but the firewall is blocking it.

Diagnose:

```bash
yeast ssh chaos-proxy
curl http://192.168.91.20:8000  # hangs — network blocked, not refused
exit

yeast ssh chaos-app
sudo systemctl is-active app    # active — app is fine
sudo ufw status                 # shows port 8000 DENY
```

Recover:

```bash
sudo ufw delete deny 8000
sudo ufw allow 8000
sudo ufw reload
exit

curl http://localhost:8880  # works again
```

---

## Post-Drill Review

After all five breaks, review your runbook. Update it with:
- The actual time each recovery took
- Any step that was harder than expected
- Commands you had to look up
- The order that would have been fastest

This review is as important as the drill itself. The runbook gets better every time you practice.

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

In Lab 22 — Chaos And Failure Recovery Drill, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- Why you write the runbook before the failure, not during it
- Five concrete failure modes and how to diagnose and recover each
- MTTR: how to measure it by timing your own recovery
- The diagnosis pattern for each failure type: stopped service → systemctl; database down → logs; disk full → df -h; config error → journalctl; firewall → ufw status
- The difference between "service is running" (systemctl) and "service is reachable" (network/firewall)
- Why chaos drills in the lab make production incidents faster and calmer to handle

---

## What Is Next

**Lab 23 — Terraform Fundamentals**

You have been configuring servers manually and with Ansible. Terraform takes infrastructure-as-code further: it describes not just server configuration, but the infrastructure itself — VMs, networks, DNS records, cloud resources — as declarative code. Lab 23 teaches the fundamentals: plan, apply, state, variables, and outputs.
