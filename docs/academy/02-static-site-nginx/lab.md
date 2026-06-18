# Lab 02 — Static Site With Nginx

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner |
| Estimated time | 45-75 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2202 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Comfort creating SSH tunnels from `ACCESS.md` for browser-based tools

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

Your team needs an internal page online. Nothing fancy — a static site with some HTML. The infrastructure request is simple: "Put up a web server. Point it at some files. Make it reachable on port 80."

You have a clean server from Lab 01. Now you need to install Nginx, configure it to serve files, place your HTML, and verify the whole chain works — from the server process, to the port, to an actual HTTP response in a browser.

And when it breaks — and it will break, because that is part of the lab — you will read logs, trace the failure, and fix it. That skill is more valuable than knowing how to set it up correctly in the first place.

---

## Before You Start — Understanding The Concepts

### What Is A Web Server?

A web server is a program that listens for HTTP requests on a network port and responds with content — usually HTML files, images, or data.

When you type a URL in your browser, your browser:
1. Resolves the domain name to an IP address
2. Opens a TCP connection to that IP on port 80 (HTTP) or 443 (HTTPS)
3. Sends an HTTP request: "GET /index.html"
4. The web server reads that request, finds the file, and sends it back
5. Your browser renders the response

A web server does not know or care what is in the files. It just maps URL paths to files on disk and sends them back. That is the fundamental job.

### What Is Nginx?

Nginx (pronounced "engine-x") is one of the two most popular web servers in the world (the other is Apache). It is fast, memory-efficient, and extremely configurable.

Nginx can do several things:
- Serve static files (HTML, CSS, JS, images) directly from disk — this is what we do in this lab
- Proxy requests to a backend application — Lab 03
- Terminate TLS (HTTPS) — later labs
- Load balance across multiple backend servers — Lab 08

For this lab we are using the simplest use case: serve static files from a directory.

### What Is A Port?

A port is a number (0–65535) that identifies a specific service running on a machine.

When a program wants to receive network connections, it "binds" to a port — it tells the operating system "give me all traffic arriving on port 80." The operating system keeps a table of which process owns which port.

Standard port assignments:
- **22** — SSH
- **80** — HTTP (unencrypted web)
- **443** — HTTPS (encrypted web)
- **5432** — PostgreSQL
- **3306** — MySQL/MariaDB
- **6379** — Redis

You can run any service on any port — these are conventions, not rules. But browsers automatically try port 80 for `http://` and port 443 for `https://`, so web servers almost always use those.

In this lab, Nginx listens on port 80 inside the VM. Yeast v1.1 does not expose that HTTP port automatically, so you test it inside the VM first. If you want to use your laptop browser, you create an SSH tunnel later in the lab.

### What Is HTTP?

HTTP (HyperText Transfer Protocol) is the language browsers and web servers use to talk to each other.

A request looks like:

```
GET /index.html HTTP/1.1
Host: example.com
```

A response looks like:

```
HTTP/1.1 200 OK
Content-Type: text/html

<html>...</html>
```

The number after `HTTP/1.1` in the response is the status code:
- **200** — OK, the request succeeded
- **301/302** — Redirect, the resource moved
- **403** — Forbidden, you do not have permission
- **404** — Not Found, the file does not exist
- **500** — Internal Server Error, something broke on the server side

You will see all of these in logs throughout this course.

### What Is A Virtual Host?

Nginx can serve multiple websites from the same server. It distinguishes between them using "virtual hosts" (also called server blocks in Nginx terms).

Each virtual host is a configuration block that says: "for requests to this domain/IP on this port, serve files from this directory."

For this lab you have one virtual host: serve files from `/var/www/baseline-site` on port 80.

### What Is /var/www?

Linux has a standard directory layout. `/var` is for variable data — things that change during normal operation. `/var/www` is the conventional location for web server content. By default, Nginx serves files from `/var/www/html`. We will create our own directory at `/var/www/baseline-site`.

---

## What You Are Building

```
Your Laptop
    │
    │  HTTP port 8080 → VM port 80
    │  SSH  port 2202 → VM port 22
    ▼
┌──────────────────────────────────────────┐
│  web (Ubuntu 22.04)                      │
│                                          │
│  Nginx listening on :80                  │
│    └── serves /var/www/baseline-site/    │
│                                          │
│  ufw: allows SSH + Nginx (ports 22, 80)  │
└──────────────────────────────────────────┘
```

The site files live on the VM's disk. Nginx reads them and sends them to your browser when you visit `http://localhost:8080`.

---

## The Config File — assets/yeast.yaml

```yaml
version: 1

instances:
  - name: web
    hostname: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 2
    disk_size: 20G
    ssh_port: 2202
    user: ubuntu
    sudo: nopasswd

    provision:
      packages:
        - nginx
        - curl
        - vim
      shell:
        - hostnamectl set-hostname web
        - timedatectl set-timezone UTC
        - ufw allow OpenSSH
        - ufw allow 'Nginx Full'
        - ufw --force enable
        - systemctl enable --now nginx
```

New fields compared to Lab 01:

**`ssh_port: 2202`** — We use 2202 here to avoid conflicts with Lab 01 if you ever run both at the same time. Each lab uses a different SSH port.

**`nginx` in packages** — This installs Nginx from Ubuntu's package repository.

**`ufw allow 'Nginx Full'`** — `Nginx Full` is a UFW application profile that allows both port 80 (HTTP) and port 443 (HTTPS). Nginx installs these profiles automatically.

**`systemctl enable --now nginx`** — Enables Nginx to start on boot and starts it right now.

---

## Starting The Lab

```bash
cd 02-static-site-nginx
yeast up
```

Once Yeast finishes, SSH in:

```bash
yeast ssh web
```

---

## Verify Nginx Is Running

Before touching anything else, verify that Nginx started correctly:

```bash
sudo systemctl status nginx
```

This command shows the full status of the nginx service. Look for:

```
● nginx.service - A high performance web server and a reverse proxy server
     Loaded: loaded (/lib/systemd/system/nginx.service; enabled; vendor preset: enabled)
     Active: active (running) since ...
```

The important parts:
- `enabled` — Nginx will start automatically after a reboot
- `active (running)` — Nginx is running right now

If it says `failed` or `inactive`, check the error:

```bash
sudo journalctl -u nginx --no-pager -n 20
```

`journalctl` is the command for reading systemd's log. `-u nginx` means "show logs for the nginx unit." `--no-pager` prevents it from opening in a pager (which requires key presses to scroll). `-n 20` shows the last 20 lines.

### Check what port Nginx is listening on

```bash
sudo ss -tlnp | grep nginx
```

`ss` is the socket statistics tool — it shows network connections and listening ports. The flags:
- `-t` — TCP only
- `-l` — listening sockets only
- `-n` — show port numbers, not service names
- `-p` — show the process that owns the socket

You should see something like:

```
LISTEN  0  511  0.0.0.0:80  0.0.0.0:*  users:(("nginx",pid=1234,fd=6))
```

This means: Nginx is listening on all interfaces (`0.0.0.0`) on port 80, and the process ID is 1234.

### Make a quick HTTP request from inside the VM

```bash
curl -I http://localhost
```

`curl` makes an HTTP request. `-I` means "HEAD request only" — fetch the headers but not the body. You should see:

```
HTTP/1.1 200 OK
Server: nginx/1.18.0 (Ubuntu)
Date: ...
Content-Type: text/html
...
```

The `200 OK` means Nginx received the request and responded successfully. Right now it is serving Nginx's default page. We are about to replace that with our own content.

---

## Create Your Site

Create the directory where your site files will live:

```bash
sudo mkdir -p /var/www/baseline-site
```

`mkdir -p` creates a directory and any parent directories that do not exist. The `-p` flag means "no error if it already exists."

Create a simple HTML page:

```bash
sudo vim /var/www/baseline-site/index.html
```

`vim` is a terminal text editor. When it opens:
1. Press `i` to enter "insert mode" (you can now type)
2. Type your HTML
3. Press `Esc` to exit insert mode
4. Type `:wq` and press Enter to save and quit

Type this content:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Baseline Site</title>
</head>
<body>
  <h1>Welcome to the Baseline Site</h1>
  <p>This is Lab 02 of the DevOps Bootcamp.</p>
  <p>Server: web</p>
</body>
</html>
```

Now set the correct permissions so Nginx can read the file:

```bash
sudo chown -R www-data:www-data /var/www/baseline-site
sudo chmod -R 755 /var/www/baseline-site
```

**`chown`** changes the owner of a file or directory. `www-data` is the user that Nginx runs as. By setting the owner to `www-data`, you ensure Nginx has permission to read the files.

**`chmod 755`** sets the permissions:
- `7` (owner) = read + write + execute
- `5` (group) = read + execute
- `5` (others) = read + execute

For directories, "execute" means "can enter the directory." For files, it means "can run as a program." HTML files do not need execute permission, but 755 is the standard for web-served content.

---

## Configure Nginx To Serve Your Site

Nginx's configuration lives in `/etc/nginx/`. The main config is `/etc/nginx/nginx.conf`, but individual site configs go in `/etc/nginx/sites-available/` and are enabled by creating a symlink in `/etc/nginx/sites-enabled/`.

This two-directory pattern lets you have multiple site configs ready but only enable the ones you want active.

Create a config for your site:

```bash
sudo vim /etc/nginx/sites-available/baseline-site
```

Enter this content:

```nginx
server {
    listen 80;
    server_name _;

    root /var/www/baseline-site;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    access_log /var/log/nginx/baseline-site.access.log;
    error_log  /var/log/nginx/baseline-site.error.log;
}
```

Let's read this:

**`listen 80`** — this server block handles requests on port 80.

**`server_name _`** — the underscore is a catch-all. It matches any hostname. Since we are not using a real domain name, this is fine.

**`root /var/www/baseline-site`** — the base directory where files are served from.

**`index index.html`** — when a request comes in for a directory (like `/`), serve `index.html` from it.

**`location /`** — this block handles all URL paths. `try_files $uri $uri/ =404` means: try to find a file matching the URL path, then try it as a directory, and if neither exists return a 404.

**`access_log` and `error_log`** — write logs to these specific files rather than the default Nginx log. This keeps your site's logs separate from other sites or the default Nginx page.

### Enable the site

```bash
sudo ln -s /etc/nginx/sites-available/baseline-site /etc/nginx/sites-enabled/
```

`ln -s` creates a symbolic link (symlink) — a pointer from one path to another. The `sites-enabled` directory just contains symlinks into `sites-available`. Disabling a site is as simple as removing the symlink.

### Disable the default site

The default Nginx site is enabled and would compete for port 80:

```bash
sudo rm /etc/nginx/sites-enabled/default
```

### Test the config

Before reloading Nginx, test that the config file has no syntax errors:

```bash
sudo nginx -t
```

Expected output:

```
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
```

If you see errors, Nginx tells you the file and line number. Fix the error and test again. Never reload Nginx with a broken config — it will fail to start.

### Reload Nginx

```bash
sudo systemctl reload nginx
```

`reload` tells Nginx to re-read its config without stopping. It is safer than `restart` because Nginx keeps serving existing connections while loading the new config. In production, always use `reload` when changing config.

---

## Verify The Site

Test from inside the VM:

```bash
curl http://localhost
```

You should see your HTML:

```html
<!DOCTYPE html>
<html lang="en">
...
<h1>Welcome to the Baseline Site</h1>
...
```

To test from your laptop browser, first create an SSH tunnel from a second terminal on your laptop:

```bash
ssh -N -L 8080:127.0.0.1:80 -p 2202 ubuntu@127.0.0.1
```

Keep that tunnel terminal open. Then open a browser and go to:

```
http://localhost:8080
```

The tunnel maps port 8080 on your laptop to port 80 inside the VM. You should see your page rendered in the browser.

You can also test from the command line on your laptop:

```bash
curl http://localhost:8080
```

---

## Reading Logs

Logs are how you understand what a server is doing. Get comfortable reading them now — they are your primary debugging tool for every lab in this course.

### Access log

Every HTTP request Nginx receives is written to the access log:

```bash
sudo tail -f /var/log/nginx/baseline-site.access.log
```

`tail -f` follows a file — it prints new lines as they are added. Leave this running, keep the SSH tunnel open, then make a request from your browser to `http://localhost:8080`. You will see a line appear:

```
127.0.0.1 - - [15/Jun/2026:12:00:00 +0000] "GET / HTTP/1.1" 200 312 "-" "curl/7.81.0"
```

Reading the fields:
- `127.0.0.1` — IP address of the client that made the request
- `[15/Jun/2026:12:00:00 +0000]` — timestamp in UTC
- `"GET / HTTP/1.1"` — the HTTP method, path, and protocol version
- `200` — the response status code
- `312` — the number of bytes in the response body
- `"-"` — the Referer header (which page linked to this one — empty here)
- `"curl/7.81.0"` — the User-Agent (what made the request)

Press `Ctrl+C` to stop following the log.

### Error log

```bash
sudo tail -f /var/log/nginx/baseline-site.error.log
```

This file records errors — permission problems, missing files, config issues. Right now it should be empty or have very few lines. We will see it in action in the next section.

---

## Break Something On Purpose

### Break 1: Wrong file permissions

Change the permissions on your site directory so Nginx cannot read it:

```bash
sudo chmod 700 /var/www/baseline-site
```

Now make a request:

```bash
curl http://localhost
```

You get:

```
<html>
<head><title>403 Forbidden</title></head>
...
```

403 Forbidden. Nginx can reach the directory but does not have permission to read the files. Check the error log:

```bash
sudo cat /var/log/nginx/baseline-site.error.log
```

You will see something like:

```
[error] 1234#1234: *1 open() "/var/www/baseline-site/index.html" failed (13: Permission denied)
```

The log tells you exactly what went wrong: permission denied on the specific file. This is how real debugging works — you read the log, it tells you the problem.

Fix it:

```bash
sudo chmod 755 /var/www/baseline-site
curl http://localhost
```

200 OK again.

### Break 2: Nginx config syntax error

Edit the Nginx config and introduce a deliberate syntax error:

```bash
sudo vim /etc/nginx/sites-available/baseline-site
```

Remove the semicolon from the end of the `listen 80` line so it reads `listen 80` with no semicolon. Save and exit.

Now test the config:

```bash
sudo nginx -t
```

```
nginx: [emerg] invalid parameter "server_name" in /etc/nginx/sites-available/baseline-site:2
nginx: configuration file /etc/nginx/nginx.conf test failed
```

Nginx caught the error before you could reload. This is why you always run `nginx -t` before reloading. If you had reloaded, Nginx would have failed to apply the new config (or in the case of `restart`, failed to start entirely).

Fix it — put the semicolon back — then:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

### Break 3: Missing index file

Rename the index file:

```bash
sudo mv /var/www/baseline-site/index.html /var/www/baseline-site/index.html.bak
```

Make a request:

```bash
curl http://localhost
```

You get a 404. Check the error log:

```bash
sudo cat /var/log/nginx/baseline-site.error.log
```

```
[error] open() "/var/www/baseline-site/index.html" failed (2: No such file or directory)
```

The access log also records the 404:

```bash
sudo tail -3 /var/log/nginx/baseline-site.access.log
```

```
127.0.0.1 - - [...] "GET / HTTP/1.1" 404 153 ...
```

Fix it:

```bash
sudo mv /var/www/baseline-site/index.html.bak /var/www/baseline-site/index.html
curl http://localhost
```

---

## The Debugging Pattern

Every time you debugged a failure in this lab, you followed the same pattern:

1. **Observe the symptom** — wrong status code, connection refused, empty response
2. **Check the service status** — `systemctl status nginx`
3. **Read the error log** — `tail -f /var/log/nginx/error.log`
4. **Check the access log** — what did the server actually receive and respond?
5. **Fix the root cause** — not the symptom, the cause
6. **Verify the fix** — run the same request that failed and confirm it succeeds

This pattern works for every web server, every application, every service. The tools change. The pattern does not.

---

## Take A Snapshot

```bash
exit  # back to your laptop

yeast snapshot web working-site
```

---

## Validate Your Work

```bash
bash assets/validate.sh
```

Note: the validate script connects to the VM over SSH and checks Nginx from inside the VM. You do not need an HTTP tunnel for validation.

---

## Clean Up

```bash
yeast destroy
```

---

## Quick Recap

In Lab 02 — Static Site With Nginx, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What a web server does: listen on a port, map URLs to files, send responses
- What Nginx is and the difference between static serving and proxying
- How ports work and why port 80 is the standard for HTTP
- How to read HTTP status codes: 200 OK, 403 Forbidden, 404 Not Found
- Nginx configuration: `sites-available`, `sites-enabled`, symlinks, server blocks
- File permissions and `chown`: why `www-data` owns web files
- `nginx -t`: always test config before reloading
- `systemctl reload` vs `restart`: reload is safer for config changes
- How to read access and error logs: every field, what it means
- The debugging pattern: symptom → service status → error log → fix → verify

---

## What Is Next

**Lab 03 — Reverse Proxy To Backend App**

You have a web server. Now we add complexity: two VMs, one acting as a reverse proxy and the other running a backend application. You will configure Nginx to forward requests from the internet-facing server to the internal backend, and learn how traffic actually flows through a modern web architecture.
