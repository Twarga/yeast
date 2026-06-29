# Lab 05 — Linux Troubleshooting Drill

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner |
| Estimated time | 45-75 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2208 |
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

Your team's server stopped working. You get a Slack message: "The site is down, nothing loads." No details. No error message. Just "it's broken."

This is real work. Unlike the previous labs where you built things step by step, here someone hands you a broken system and your job is to figure out what is wrong and fix it. You will break this server yourself — in seven different ways — and practice diagnosing and recovering each one.

By the end of this lab, you will have a repeatable mental model for approaching any broken Linux server. Not a checklist to blindly follow — a way of thinking.

---

## Before You Start — Understanding The Concepts

### The Logs-First Mindset

The single most important habit in Linux troubleshooting is this: **look at logs before you do anything else.**

When a service fails, your first instinct might be to restart it, or edit a config file, or reinstall the package. Resist that. If you act before you understand, you might:
- Restart a service and lose the error that was in its output
- Edit a config and make the problem worse
- Waste time fixing the wrong thing

The correct order is always:
1. Observe the symptom
2. Read the relevant logs
3. Form a hypothesis about the cause
4. Test the hypothesis (often a single command)
5. Fix the root cause
6. Verify the fix

### The Most Important Commands For Troubleshooting

**`systemctl status <service>`** — shows if a service is running, enabled, and the last few log lines. Start here when a service seems broken.

**`journalctl -u <service> --no-pager -n 50`** — reads the systemd journal for a specific service. Shows you exactly what the service printed when it started or crashed. The `-n 50` shows the last 50 lines.

**`sudo ss -tlnp`** — shows all listening TCP sockets and which process owns them. Answers "is anything listening on port 80?" and "which process is using port 5432?"

**`curl -v http://localhost`** — makes an HTTP request and shows the full request/response including headers. The `-v` flag is "verbose."

**`sudo journalctl -xe`** — shows the most recent system log entries with context. Useful when something failed at boot or service startup.

**`df -h`** — shows disk usage. A full disk breaks almost everything and is a surprisingly common cause of mysterious failures.

**`du -sh /var/log/*`** — shows how much space each log directory uses. Log files that are never rotated will eventually fill the disk.

**`ps aux`** — lists all running processes. Useful to check if a process is actually running and what user it runs as.

**`sudo lsof -i :80`** — shows which process has port 80 open. `lsof` means "list open files" — in Linux, sockets are files.

**`sudo cat /var/log/nginx/error.log`** — reads Nginx's error log directly without systemd.

**`sudo chmod`, `sudo chown`** — fix file permissions. Many service failures are caused by a file being owned by the wrong user or having wrong permissions.

---

## What You Are Building

One VM running Nginx and PostgreSQL. You will break it in seven ways and fix each one.

```
Your Laptop
    │
    │  SSH  port 2208 → drill VM port 22
    │  HTTP port 8080 → drill VM port 80
    ▼
┌──────────────────────────────────┐
│  drill (Ubuntu 22.04)            │
│                                  │
│  Nginx :80                       │
│  PostgreSQL :5432                │
│  ufw: SSH + HTTP allowed         │
└──────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 05-linux-troubleshooting-drill
yeast up
yeast ssh drill
```

Before breaking anything, verify the starting state. A good troubleshooter always establishes a baseline — what does "working" look like?

```bash
sudo systemctl is-active nginx
sudo systemctl is-active postgresql
sudo ss -tlnp | grep -E ':80|:5432'
curl -s -o /dev/null -w "%{http_code}" http://localhost
df -h /
```

Expected: nginx active, postgres active, ports 80 and 5432 listening, HTTP returns 200, disk not full.

Take a snapshot from your laptop now:

```bash
exit
yeast snapshot drill clean-state
yeast ssh drill
```

---

## Break 1 — Service Stopped

Break it:

```bash
sudo systemctl stop nginx
```

From your laptop, try to load the site:

```bash
curl http://localhost:8080
```

Result: `curl: (7) Failed to connect to localhost port 8080: Connection refused`

Now diagnose it from inside the VM. Follow the process:

**Step 1 — Check the service:**

```bash
sudo systemctl status nginx
```

```
● nginx.service - A high performance web server and a reverse proxy server
     Loaded: loaded (/lib/systemd/system/nginx.service; enabled; vendor preset: enabled)
     Active: inactive (dead) since ...
```

`inactive (dead)` — the service is not running. Note it says `enabled` — it is set to start on boot, but it is not running right now.

**Step 2 — Check the journal:**

```bash
sudo journalctl -u nginx --no-pager -n 20
```

This shows you what happened when nginx stopped. In this case it was a clean stop — someone called `systemctl stop`.

**Step 3 — Fix and verify:**

```bash
sudo systemctl start nginx
sudo systemctl is-active nginx
curl -s -o /dev/null -w "%{http_code}" http://localhost
```

Expected: `active`, `200`.

**The lesson:** A service being `enabled` does not mean it is `active`. Enable = starts on boot. Active = running right now. Both can be true or false independently. When diagnosing a "service not running" problem, always check both.

---

## Break 2 — Nginx Config Syntax Error

Break it:

```bash
sudo sed -i 's/listen 80;/listen 80/' /etc/nginx/sites-enabled/default
sudo systemctl restart nginx
```

Check the status:

```bash
sudo systemctl status nginx
```

```
● nginx.service
   Active: failed (Result: exit-code)
...
nginx: [emerg] invalid parameter "server_name" in /etc/nginx/sites-enabled/default:3
```

Nginx failed to start because the config is broken. The error message tells you the file and line number.

**Diagnose:**

```bash
sudo nginx -t
```

```
nginx: [emerg] invalid parameter "server_name" in /etc/nginx/sites-enabled/default:3
nginx: configuration file /etc/nginx/nginx.conf test failed
```

`nginx -t` validates the config without starting Nginx. It is safe to run even when Nginx is stopped or broken.

**Fix:**

```bash
sudo sed -i 's/listen 80$/listen 80;/' /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl start nginx
curl -s -o /dev/null -w "%{http_code}" http://localhost
```

**The lesson:** When Nginx fails to start, check `nginx -t` before looking at logs. Config syntax errors are the most common cause of Nginx startup failures and `nginx -t` gives you the exact file and line.

---

## Break 3 — Wrong File Permissions

Break it:

```bash
sudo chmod 000 /var/www/html
```

Make a request:

```bash
curl http://localhost
```

Result: `403 Forbidden`

**Diagnose:**

Start with the Nginx error log:

```bash
sudo tail -5 /var/log/nginx/error.log
```

```
[error] open() "/var/www/html/index.nginx-debian.html" failed (13: Permission denied)
```

Permission denied. Check what permissions the directory has:

```bash
ls -la /var/www/
```

```
d---------  2 root root 4096 Jun 15 12:00 html
```

`000` means no permissions for anyone — not owner, not group, not others. Nginx runs as `www-data` and cannot read anything.

**Fix:**

```bash
sudo chmod 755 /var/www/html
curl -s -o /dev/null -w "%{http_code}" http://localhost
```

**The lesson:** 403 Forbidden from Nginx almost always means a file permission problem. Go straight to the error log — it tells you which file and why.

---

## Break 4 — Port Conflict

Break it:

```bash
# Start a dummy process that takes port 80
sudo python3 -c "
import socket
s = socket.socket()
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(('0.0.0.0', 80))
s.listen(1)
print('listening')
import time; time.sleep(3600)
" &

sudo systemctl restart nginx
```

Check the status:

```bash
sudo systemctl status nginx
```

```
● nginx.service
   Active: failed
...
nginx: [emerg] bind() to 0.0.0.0:80 failed (98: Address already in use)
```

Nginx cannot start because something else is already using port 80.

**Diagnose:**

```bash
sudo ss -tlnp | grep ':80'
```

```
LISTEN  0  1  0.0.0.0:80  ...  users:(("python3",pid=1234,fd=3))
```

A Python process is holding port 80. Get its PID from this output.

```bash
sudo lsof -i :80
```

This also shows you which process owns the port, with more detail.

**Fix:**

```bash
sudo kill 1234  # use the actual PID from ss output
sudo systemctl start nginx
sudo ss -tlnp | grep ':80'
```

**The lesson:** When a service fails with "address already in use," use `ss -tlnp` or `lsof -i :<port>` to find what is holding the port. Kill the conflicting process, then start your service.

---

## Break 5 — Full Disk

Break it by creating a large file:

```bash
# Fill the disk to ~95% (this creates a ~1GB file — adjust size if needed)
df -h /
sudo dd if=/dev/zero of=/var/tmp/bigfile bs=1M count=15000 2>/dev/null || true
df -h /
```

Now try to write something — like Nginx logging a request:

```bash
curl http://localhost
sudo tail -5 /var/log/nginx/access.log
```

And try to write a file yourself:

```bash
touch /tmp/testfile
echo "test" >> /var/log/test.log
```

The `echo` may fail with: `bash: /var/log/test.log: No space left on device`

A full disk breaks many things silently:
- Log files stop being written (so you get no logs of the actual problem)
- Application databases cannot write
- Package installs fail
- Temp files cannot be created
- Services crash because they cannot write their runtime files

**Diagnose:**

```bash
df -h
```

The `/` filesystem is at 95%+ usage.

```bash
du -sh /var/tmp/*
```

Shows the `bigfile` using most of that space.

**Fix:**

```bash
sudo rm /var/tmp/bigfile
df -h /
```

**The lesson:** A full disk is one of the most common causes of mysterious, hard-to-diagnose failures. When something breaks unexpectedly and the logs are not telling you anything useful, check `df -h` immediately.

---

## Break 6 — Service Enabled But Wrong Unit File

Break it:

```bash
sudo systemctl stop postgresql
sudo mv /etc/postgresql/14/main/postgresql.conf \
        /etc/postgresql/14/main/postgresql.conf.bak
sudo systemctl start postgresql
```

Check the status:

```bash
sudo systemctl status postgresql
```

```
● postgresql.service
   Active: failed
...
Error in configuration file "/etc/postgresql/14/main/postgresql.conf"
```

**Diagnose:**

```bash
sudo journalctl -u postgresql --no-pager -n 20
```

The journal shows the PostgreSQL error in detail — it cannot find or read the config file.

```bash
ls -la /etc/postgresql/14/main/
```

You can see `postgresql.conf` is missing and `postgresql.conf.bak` exists.

**Fix:**

```bash
sudo mv /etc/postgresql/14/main/postgresql.conf.bak \
        /etc/postgresql/14/main/postgresql.conf
sudo systemctl start postgresql
sudo systemctl is-active postgresql
```

**The lesson:** When a service fails with a config error, `journalctl -u <service>` almost always tells you exactly which config file and what is wrong. Read it before guessing.

---

## Break 7 — Wrong Ownership On A Critical File

Break it:

```bash
sudo chown nobody:nogroup /etc/nginx/nginx.conf
sudo systemctl restart nginx
```

Check:

```bash
sudo systemctl status nginx
```

```
● nginx.service
   Active: failed
...
nginx: [alert] could not open error log file: open() "/var/log/nginx/error.log" failed
nginx: [emerg] open() "/etc/nginx/nginx.conf" failed (13: Permission denied)
```

Wait — Nginx runs as root during startup (to bind port 80), then drops to `www-data`. But the config file is now owned by `nobody:nogroup` with mode `644`, and root should be able to read files owned by others... 

Actually this break reveals something interesting. Nginx's startup process that reads the config runs as root. Root can read any file regardless of permissions. So this break works differently depending on your exact Nginx version and setup. Let's check what actually happens:

```bash
ls -la /etc/nginx/nginx.conf
sudo nginx -t
```

If `nginx -t` passes (root can read the file), the real lesson is: ownership matters for the service user, not the startup user. 

Let's make it clearer — break it in a way Nginx definitely cannot read:

```bash
sudo chmod 000 /etc/nginx/nginx.conf
sudo nginx -t
```

```
nginx: [emerg] open() "/etc/nginx/nginx.conf" failed (13: Permission denied)
```

Now even root is blocked (mode 000 denies everyone including root for regular files).

**Fix:**

```bash
sudo chmod 644 /etc/nginx/nginx.conf
sudo chown root:root /etc/nginx/nginx.conf
sudo nginx -t
sudo systemctl start nginx
```

**The lesson:** `chmod 000` blocks everyone including root. `chmod 644` (read for owner and group, read-only for others) is the standard for config files. The owner should be `root:root` for system config files.

---

## The Recovery Verification

After working through all seven breaks, verify the server is fully recovered:

```bash
sudo systemctl is-active nginx
sudo systemctl is-active postgresql
sudo ss -tlnp | grep -E ':80|:5432'
curl -s -o /dev/null -w "%{http_code}" http://localhost
df -h /
```

All green. Now run the validation script from your laptop:

```bash
bash assets/validate.sh
```

---

## The Troubleshooting Mental Model

Every time you debug a broken Linux service, you follow this path:

```
1. What is the symptom?
   (connection refused / 5xx response / service shows failed / disk error)
   
2. Is the service running?
   sudo systemctl status <service>
   
3. What did the service say when it failed?
   sudo journalctl -u <service> --no-pager -n 50
   
4. Is anything listening on the expected port?
   sudo ss -tlnp | grep :<port>
   
5. Is the disk full?
   df -h
   
6. Are the file permissions correct?
   ls -la <config file or content directory>
   
7. Is the config valid?
   sudo <service> -t  (for nginx, apache)
   
8. Fix the root cause, not the symptom.
   Then verify.
```

Write this down. Use it every time. The tools change. This pattern does not.

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

In Lab 05 — Linux Troubleshooting Drill, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- `systemctl status` — the first command to run when a service breaks
- `journalctl -u <service>` — reads the service's actual output and errors
- `ss -tlnp` — who is listening on which port
- `lsof -i :<port>` — which process owns a specific port
- `nginx -t` — validate config without restarting
- `df -h` — disk usage, always check this when something breaks mysteriously
- `du -sh` — where disk space is going
- `chmod` and `chown` — fix permission and ownership problems
- The difference between `enabled` (starts on boot) and `active` (running now)
- Seven concrete failure modes and how to diagnose each one
- The troubleshooting mental model: symptom → service status → logs → port check → disk check → permissions → config → fix → verify

---

## What Is Next

**Lab 06 — Bash Automation For Server Setup**

You have now set up servers manually five times. You know the steps. Lab 06 is about making those steps repeatable, safe, and scriptable. You will write a Bash script that automates everything you have done manually so far — and learn why automating something correctly is harder than just doing it manually once.
