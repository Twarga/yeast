# Lab 03 — Reverse Proxy To Backend App

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner |
| Estimated time | 45-75 minutes |
| VMs | 2 |
| Minimum VM RAM | 2048 MB |
| SSH ports | 2203, 2204 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
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

The static site from Lab 02 worked fine, but now the team wants to deploy a real application — something that generates responses dynamically, not just serves files from disk.

The problem: you do not want to expose your backend application directly to the internet. Backend apps are often written by developers who are not thinking about network security. They might bind to all interfaces, expose debug endpoints, or lack rate limiting. You want a layer in front of it that handles the "internet-facing" concerns.

The solution is a reverse proxy. You put Nginx on one VM facing the outside world. Behind it, on a private network, you run the backend application on another VM. Nginx forwards requests from the outside to the backend and returns the backend's responses to the client. The backend never needs to be directly reachable from the internet.

This architecture is standard. Almost every production web service you will ever work with has a proxy in front of the actual application.

---

## Before You Start — Understanding The Concepts

### What Is A Reverse Proxy?

A proxy is something that acts on behalf of something else. A forward proxy acts on behalf of clients — your corporate network might run a proxy that all outbound traffic passes through for filtering. A reverse proxy acts on behalf of servers.

When a client makes a request to a reverse proxy:
1. The proxy receives the request
2. The proxy forwards it to a backend server
3. The backend processes it and sends a response to the proxy
4. The proxy forwards the response to the original client

From the client's perspective, they are only talking to the proxy. They never know the backend exists or what its address is.

This gives you several advantages:
- **Security** — the backend is not directly exposed to the internet
- **Flexibility** — you can change backend servers without the client noticing
- **SSL termination** — the proxy handles HTTPS, the backend can use plain HTTP internally
- **Load balancing** — the proxy can distribute requests across multiple backend instances

### What Is An Upstream?

In Nginx proxy configuration, the "upstream" is the backend server (or group of servers) that Nginx forwards requests to. "Upstream" and "downstream" describe the direction of data flow relative to the user: the user is downstream, the backend is upstream.

In Nginx config you define an upstream block with the address and port, then reference it in a `proxy_pass` directive.

### What Is A Private Network?

In this lab, both VMs share a private network (`192.168.10.0/24`). This network exists only between the VMs — it is not reachable from your laptop or the internet. The proxy VM has an IP on this network (`192.168.10.10`) and the backend VM has an IP (`192.168.10.20`).

The proxy can reach the backend via `192.168.10.20:8000`. Your laptop cannot directly reach the backend at all. To reach the proxy from your laptop browser, you create an SSH tunnel to the proxy VM.

This is the same pattern used in real infrastructure: a public subnet for load balancers and proxies, a private subnet for application and database servers.

### What Is An IP Address And A Subnet?

An IP address is a number that identifies a machine on a network. IPv4 addresses are written as four numbers separated by dots: `192.168.10.20`.

A subnet is a range of IP addresses. `192.168.10.0/24` means "the range of addresses from 192.168.10.0 to 192.168.10.255" — the `/24` says the first 24 bits are the network part and the last 8 bits identify individual machines. That gives you 254 usable addresses.

Machines on the same subnet can communicate directly. Machines on different subnets need a router between them.

### What Is Python's Built-In HTTP Server?

Python ships with a simple HTTP server in its standard library. You can start it with one command:

```bash
python3 -m http.server 8000
```

This serves files from the current directory on port 8000. It is not production-grade — it is single-threaded and has no security features — but it is perfect for testing connectivity and proxy configuration.

For a more realistic backend, we will use Python's `http.server` with a custom handler that returns a simple JSON response.

---

## What You Are Building

```
Your Laptop
    │
    │  SSH  port 2203 → proxy VM port 22
    │  SSH  port 2204 → backend VM port 22
    │  SSH tunnel 8080 → proxy VM port 80
    │
    ▼
┌─────────────────────────────────────────────────────┐
│  Private Network: 192.168.10.0/24                   │
│                                                     │
│  ┌──────────────────────┐    ┌──────────────────┐   │
│  │  proxy               │    │  backend         │   │
│  │  192.168.10.10       │───▶│  192.168.10.20   │   │
│  │                      │    │                  │   │
│  │  Nginx :80           │    │  Python app :8000│   │
│  │  proxy_pass →        │    │                  │   │
│  │  192.168.10.20:8000  │    │                  │   │
│  └──────────────────────┘    └──────────────────┘   │
└─────────────────────────────────────────────────────┘
```

Traffic flow:
1. Your browser → `http://localhost:8080`
2. SSH tunnel → proxy VM port 80
3. Nginx on proxy → `http://192.168.10.20:8000`
4. Python app on backend → response
5. Response travels back the same path

---

## The Config File — assets/yeast.yaml

```yaml
version: 1

networks:
  - name: internal
    cidr: 192.168.10.0/24

instances:
  - name: proxy
    hostname: proxy
    image: ubuntu-22.04
    memory: 1024
    cpus: 2
    disk_size: 20G
    ssh_port: 2203
    user: ubuntu
    sudo: nopasswd
    networks:
      - name: internal
        ipv4: 192.168.10.10

    provision:
      packages:
        - nginx
        - curl
      shell:
        - hostnamectl set-hostname proxy
        - timedatectl set-timezone UTC
        - ufw allow OpenSSH
        - ufw allow 'Nginx Full'
        - ufw --force enable
        - systemctl enable --now nginx

  - name: backend
    hostname: backend
    image: ubuntu-22.04
    memory: 1024
    cpus: 2
    disk_size: 20G
    ssh_port: 2204
    user: ubuntu
    sudo: nopasswd
    networks:
      - name: internal
        ipv4: 192.168.10.20

    provision:
      packages:
        - python3
        - curl
      shell:
        - hostnamectl set-hostname backend
        - timedatectl set-timezone UTC
        - ufw allow OpenSSH
        - ufw allow 8000
        - ufw --force enable
```

New concepts in this config:

**`networks` block at the top** — defines a named private network with a subnet CIDR. Yeast creates a virtual network that both VMs can use.

**`networks` inside each instance** — attaches the instance to the named network and assigns it a static IP. The proxy gets `.10`, the backend gets `.20`.

**Two instances in one file** — Yeast can manage multiple VMs from one `yeast.yaml`. `yeast up` boots all of them. `yeast down` stops all of them.

**`ufw allow 8000`** on the backend — the backend app listens on port 8000. We allow it in the firewall so the proxy can reach it over the private network.

---

## Starting The Lab

```bash
cd 03-reverse-proxy-backend-app
yeast up
```

Yeast boots both VMs. This takes a bit longer than a single VM. You can follow along:

```bash
yeast up --events
```

When done, verify both are up:

```bash
yeast status
```

Expected:

```
NAME      STATE    SSH PORT
proxy     running  2203
backend   running  2204
```

---

## Set Up The Backend Application

SSH into the backend VM:

```bash
yeast ssh backend
```

Create a simple Python HTTP server that returns a JSON response:

```bash
mkdir -p /home/ubuntu/app
cat > /home/ubuntu/app/server.py << 'EOF'
#!/usr/bin/env python3
import json
import socket
from http.server import HTTPServer, BaseHTTPRequestHandler

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({
            "message": "Hello from the backend",
            "host": socket.gethostname(),
            "path": self.path
        }).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        print(f"{self.address_string()} - {fmt % args}")

if __name__ == "__main__":
    server = HTTPServer(("0.0.0.0", 8000), Handler)
    print("Backend listening on :8000")
    server.serve_forever()
EOF
chmod +x /home/ubuntu/app/server.py
```

The `<< 'EOF'` syntax is a "here document" — it lets you write multi-line content directly into a file from the terminal without opening an editor. Everything between the two `EOF` markers becomes the file content.

Now create a systemd service file so the app runs as a proper service and restarts automatically:

```bash
sudo tee /etc/systemd/system/backend-app.service << 'EOF'
[Unit]
Description=Backend Application
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
```

`tee` writes to a file and also prints to the terminal. Combined with `sudo`, it lets you write to files you do not own.

The systemd service file has three sections:

**`[Unit]`** — metadata. `After=network.target` means "start this after the network is up."

**`[Service]`** — how to run the program. `Type=simple` means it is a regular foreground process. `User=ubuntu` runs it as the ubuntu user (not root — principle of least privilege). `Restart=always` restarts it if it crashes.

**`[Install]`** — when to start automatically. `WantedBy=multi-user.target` means "start this in normal multi-user mode" — i.e., always.

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now backend-app
sudo systemctl status backend-app
```

`daemon-reload` tells systemd to re-read its service files — required after adding a new `.service` file.

Verify the app is responding:

```bash
curl http://localhost:8000
```

Expected:

```json
{"message": "Hello from the backend", "host": "backend", "path": "/"}
```

Now verify it is listening on the private network interface:

```bash
sudo ss -tlnp | grep 8000
```

Expected:

```
LISTEN  0  5  0.0.0.0:8000  ...  python3
```

`0.0.0.0` means "all interfaces" — the app is reachable on both the loopback (`127.0.0.1`) and the private network interface (`192.168.10.20`).

Exit the backend VM:

```bash
exit
```

---

## Configure Nginx As A Reverse Proxy

SSH into the proxy VM:

```bash
yeast ssh proxy
```

First, verify the proxy can reach the backend:

```bash
curl http://192.168.10.20:8000
```

You should get the same JSON response. If this fails, the private network is not working — check `yeast status` and `ip addr` on both VMs to confirm they have their expected IPs.

Create the Nginx proxy config:

```bash
sudo tee /etc/nginx/sites-available/proxy << 'EOF'
upstream backend {
    server 192.168.10.20:8000;
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://backend;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    access_log /var/log/nginx/proxy.access.log;
    error_log  /var/log/nginx/proxy.error.log;
}
EOF
```

Let's read every line of this config:

**`upstream backend`** — defines a group of backend servers named "backend." Right now there is one server: `192.168.10.20:8000`. In Lab 08 you will add multiple servers here for load balancing.

**`proxy_pass http://backend`** — tells Nginx to forward this request to the `backend` upstream group.

**`proxy_set_header Host $host`** — forwards the original `Host` header to the backend. Without this, the backend sees Nginx's hostname instead of the client's requested hostname.

**`proxy_set_header X-Real-IP $remote_addr`** — adds a header with the real client IP. Since the backend only sees Nginx's IP, it needs this header to know the actual client address. Applications use this for logging and rate limiting.

**`proxy_set_header X-Forwarded-For`** — a standard header for chaining proxies. It records the chain of IP addresses a request passed through.

Enable the site, disable the default, test and reload:

```bash
sudo ln -s /etc/nginx/sites-available/proxy /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

Test the proxy chain from inside the proxy VM:

```bash
curl http://localhost
```

Expected:

```json
{"message": "Hello from the backend", "host": "backend", "path": "/"}
```

The response comes from the backend. Nginx received the request, forwarded it to `192.168.10.20:8000`, and returned the response.

Check the proxy access log:

```bash
sudo tail -5 /var/log/nginx/proxy.access.log
```

You will see your curl request logged. The status code is 200. The request to the backend was transparent.

Exit the proxy VM:

```bash
exit
```

---

## Test The Full Chain From Your Laptop

From a second terminal on your laptop, create a tunnel to the proxy VM:

```bash
ssh -N -L 8080:127.0.0.1:80 -p 2203 ubuntu@127.0.0.1
```

Keep that tunnel terminal open, then run:

```bash
curl http://localhost:8080
```

Expected:

```json
{"message": "Hello from the backend", "host": "backend", "path": "/"}
```

The full path: your laptop → port 8080 → SSH tunnel → proxy VM port 80 → Nginx → backend VM port 8000 → Python app → response back.

Open your browser and go to `http://localhost:8080`. Same result.

---

## Understanding The Headers

Make a request and look at the response headers the backend received. SSH into the backend:

```bash
yeast ssh backend
sudo journalctl -u backend-app --no-pager -n 10
```

You will see requests logged with the client address that the backend sees — which is the proxy's private IP `192.168.10.10`, not your laptop.

This is why the `X-Real-IP` and `X-Forwarded-For` headers matter. If you ever need to know the actual client IP in your application (for logging, abuse detection, geo-targeting), you read from `X-Real-IP`, not from the TCP connection address.

---

## Break Something On Purpose

### Break 1: Backend goes down

SSH into the backend VM and stop the app:

```bash
yeast ssh backend
sudo systemctl stop backend-app
exit
```

Now from your laptop, with the SSH tunnel still open:

```bash
curl http://localhost:8080
```

You get a 502 Bad Gateway response. Nginx received the request, tried to connect to the backend, and the backend refused the connection.

Check the proxy error log:

```bash
yeast ssh proxy
sudo tail -5 /var/log/nginx/proxy.error.log
```

```
[error] connect() failed (111: Connection refused) while connecting to upstream,
client: ..., upstream: "http://192.168.10.20:8000/"
```

Nginx tells you exactly what happened: connection refused to the upstream. In production this is how you diagnose "502 errors" — almost always the backend is down or refusing connections.

Restart the backend:

```bash
exit
yeast ssh backend
sudo systemctl start backend-app
exit

curl http://localhost:8080
```

502 is gone. 200 is back.

### Break 2: Wrong upstream IP

SSH into the proxy VM and change the upstream IP to something that does not exist:

```bash
yeast ssh proxy
sudo sed -i 's/192.168.10.20/192.168.10.99/' /etc/nginx/sites-available/proxy
sudo nginx -t
sudo systemctl reload nginx
exit
```

`sed -i` edits a file in place. `s/old/new/` substitutes the old value for the new one.

From your laptop, with the SSH tunnel still open:

```bash
curl http://localhost:8080
```

This time it hangs for several seconds before returning a 502. Unlike "connection refused" (instant), connecting to a non-existent host times out — Nginx waits for a response that never comes.

Fix it:

```bash
yeast ssh proxy
sudo sed -i 's/192.168.10.99/192.168.10.20/' /etc/nginx/sites-available/proxy
sudo nginx -t
sudo systemctl reload nginx
exit
```

This is an important distinction to recognize in production:
- **Instant 502** — the backend is up but refusing connections (wrong port, firewall, service down)
- **Slow 502** — the backend IP is unreachable (network issue, wrong IP, VM down)

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

In Lab 03 — Reverse Proxy To Backend App, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What a reverse proxy is and why it exists: security, flexibility, separation of concerns
- What an upstream is in Nginx terminology
- Private networks: how to give VMs IP addresses on a network only they share
- The difference between a client-facing IP and an application-facing IP
- `proxy_pass`: how Nginx forwards requests
- `proxy_set_header`: why you forward client IP headers
- Systemd service files: `[Unit]`, `[Service]`, `[Install]` sections
- The difference between an instant 502 (refused) and a slow 502 (timeout)
- How to read Nginx error logs to diagnose upstream failures

---

## What Is Next

**Lab 04 — Database-Backed App**

The backend so far returns a hardcoded response. Real applications read and write data. In the next lab you will add PostgreSQL on a third VM, connect the backend to it, and learn how application-to-database connectivity works — and what happens when it breaks.
