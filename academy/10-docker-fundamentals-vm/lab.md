# Lab 10 — Docker Fundamentals On A VM

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner to intermediate |
| Estimated time | 60-90 minutes |
| VMs | 1 |
| Minimum VM RAM | 2048 MB |
| SSH ports | 2216 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic understanding that Docker commands run inside the VM unless stated otherwise

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

---

## The Story

You have been running applications directly on Linux VMs: install a binary, write a systemd unit, manage dependencies manually. Every application shares the OS. If two apps need different versions of Python, you have a conflict. If one app's dependency update breaks another, you have an outage. And recreating that exact environment on a new server means remembering every package, every config, every system tweak.

Containers solve this. A container packages an application and everything it needs to run — libraries, config, runtime — into a single portable unit. It runs in isolation from other containers. The same container image runs identically on your laptop, your staging server, and production. No more "works on my machine."

This lab teaches Docker from first principles. You will pull images, run containers, manage volumes, work with networks, and understand what is actually happening under the hood — because containers are not magic, they are Linux kernel features with good tooling on top.

---

## Before You Start — Understanding The Concepts

### What Is A Container?

A container is a process running in an isolated environment on a Linux host. The isolation comes from two Linux kernel features:

**Namespaces** — hide parts of the system from the process. A container has its own:
- PID namespace: it thinks it is the only process, its PID 1
- Network namespace: it has its own network interfaces, port space
- Mount namespace: it has its own filesystem view
- UTS namespace: it has its own hostname

**cgroups** (control groups) — limit resource usage. You can tell a container "you get 512 MB of RAM and no more."

The container process is still just a Linux process. It is not a virtual machine — there is no separate OS kernel. It shares the host kernel. This is why containers start in milliseconds instead of seconds.

### What Is A Container Image?

A container image is a read-only filesystem snapshot. It contains the files needed to run an application: the binary, libraries, config files, OS user-space tools.

Images are layered. Each layer is a diff from the previous one. The base layer might be Ubuntu. A second layer adds Python. A third layer adds your application code. When you pull an image, Docker downloads only the layers you do not already have.

### What Is Docker?

Docker is the most popular container runtime and toolchain. It provides:
- `docker run` — create and start a container from an image
- `docker pull` — download an image from a registry
- `docker build` — build a new image from a Dockerfile
- `docker ps` — list running containers
- Docker Hub — the default public image registry (millions of images)

### What Is A Registry?

A registry is a server that stores and serves container images. Docker Hub (`hub.docker.com`) is the default public registry. When you run `docker pull nginx`, Docker pulls from Docker Hub.

You can also run your own private registry (Lab 15).

### What Is A Volume?

By default, everything inside a container is ephemeral — when the container is removed, all its data is gone. A volume is a way to persist data outside the container.

Docker volumes are managed by Docker and live at `/var/lib/docker/volumes/` on the host. You attach a volume to a container and the container reads and writes to it. The data survives container restarts and removals.

### What Is Port Mapping?

Inside a container, a process listens on a port. To make that port reachable from outside the container, you map it to a host port:

```bash
docker run -p 8080:80 nginx
```

`8080:80` means: map host port 8080 to container port 80. Traffic to `localhost:8080` on the host reaches Nginx inside the container on port 80.

This is the same concept as Yeast's `ssh_port` mapping — a port on the outside maps to a port on the inside.

### What Is Docker Networking?

Docker creates a virtual network called `bridge` by default. Containers on the same bridge network can reach each other by container name. A container named `db` is reachable at hostname `db` from other containers on the same network.

You can create custom networks to group containers that need to communicate.

---

## What You Are Building

A single VM with Docker installed. You will:
- Run your first container
- Run Nginx in a container with a port mapping
- Persist data with named volumes
- Connect containers on a custom network
- Understand container lifecycle: create, start, stop, remove

---

## Starting The Lab

```bash
cd 10-docker-fundamentals-vm
yeast up
yeast ssh docker
```

The VM boots with Docker already installed via the provision block. Verify:

```bash
docker --version
sudo systemctl is-active docker
```

Add yourself to the docker group (already done in provisioning, but re-login to activate it):

```bash
newgrp docker
docker run hello-world
```

Expected output includes: `Hello from Docker!`

This confirms Docker can pull an image and run a container.

---

## Understanding What Just Happened

When you ran `docker run hello-world`:

1. Docker looked for the `hello-world` image locally — not found
2. Docker pulled it from Docker Hub (a tiny ~13KB image)
3. Docker created a container from that image
4. The container ran its process (printed the message) and exited
5. The container now exists in a "stopped" state

```bash
docker ps -a
```

`-a` shows all containers including stopped ones. You will see the hello-world container with status `Exited (0)`.

`docker ps` without `-a` only shows running containers.

---

## Running An Nginx Container

Pull and run Nginx:

```bash
docker run -d --name webserver -p 8080:80 nginx
```

Flags:
- `-d` — detached mode: run in background
- `--name webserver` — give the container a name instead of a random one
- `-p 8080:80` — map host port 8080 to container port 80

Check it is running:

```bash
docker ps
```

```
CONTAINER ID   IMAGE   COMMAND                  CREATED   STATUS    PORTS                  NAMES
abc123         nginx   "/docker-entrypoint.…"   5s ago    Up 4s     0.0.0.0:8080->80/tcp   webserver
```

Make a request:

```bash
curl http://localhost:8080
```

You get the default Nginx welcome page. The same page you saw in Lab 02 — but this time Nginx is running inside a container, not installed on the host.

Check that Nginx is NOT installed on the host:

```bash
which nginx
```

Nothing. The host has no Nginx. Only the container does.

---

## Container Lifecycle

```bash
# Stop the container (graceful shutdown, SIGTERM then SIGKILL)
docker stop webserver

# Verify it stopped
docker ps        # not here
docker ps -a     # here, status Exited

# Start it again
docker start webserver
curl http://localhost:8080  # still works

# Restart (stop + start)
docker restart webserver

# Remove a stopped container
docker stop webserver
docker rm webserver

# Remove a running container (force)
docker rm -f webserver
```

`docker rm` removes the container. The image stays — you can create a new container from it. Images and containers are different things.

---

## Inspecting Containers

Run a container:

```bash
docker run -d --name webserver -p 8080:80 nginx
```

Look inside the running container without SSHing — containers do not have SSH:

```bash
# Run a command inside a running container
docker exec webserver ls /etc/nginx

# Open an interactive shell inside the container
docker exec -it webserver bash
```

Inside the container you have a minimal Debian environment. Look around:

```bash
ps aux          # only nginx processes — no systemd, no other services
cat /etc/os-release
ls /etc/nginx/conf.d/
exit
```

This is the container's isolated filesystem view. It looks like a minimal Linux install, but it is not a VM — it shares the host kernel.

### Reading Container Logs

```bash
docker logs webserver
docker logs -f webserver   # follow, like tail -f
docker logs --tail 20 webserver  # last 20 lines
```

Nginx writes access and error logs to stdout/stderr. Docker captures that output and makes it available via `docker logs`. This is why containerized applications should log to stdout — not to files — so Docker can capture them.

---

## Volumes: Persisting Data

Run a container without a volume, write something, then remove the container:

```bash
docker run -d --name temp nginx
docker exec temp bash -c "echo 'test data' > /tmp/myfile"
docker exec temp cat /tmp/myfile  # exists
docker rm -f temp
docker run -d --name temp2 nginx
docker exec temp2 cat /tmp/myfile  # gone — new container has fresh filesystem
docker rm -f temp2
```

Data inside a container is ephemeral. Now use a volume:

```bash
# Create a named volume
docker volume create webdata

# Mount it when starting the container
docker run -d --name webserver -p 8080:80 -v webdata:/usr/share/nginx/html nginx

# Write a custom page to the volume via the container
docker exec webserver bash -c "echo '<h1>Persistent</h1>' > /usr/share/nginx/html/index.html"

curl http://localhost:8080  # shows "Persistent"

# Remove and recreate the container
docker rm -f webserver
docker run -d --name webserver -p 8080:80 -v webdata:/usr/share/nginx/html nginx

curl http://localhost:8080  # still "Persistent" — data survived
```

The volume `webdata` exists independently of the container. You can inspect it:

```bash
docker volume ls
docker volume inspect webdata
```

`Mountpoint` in the inspect output shows the path on the host where the data actually lives.

---

## Bind Mounts: Host Directory Into Container

A bind mount connects a directory on the host to a path inside the container:

```bash
mkdir -p /home/ubuntu/mysite
echo '<h1>From the host</h1>' > /home/ubuntu/mysite/index.html

docker run -d --name webserver2 -p 8081:80 \
  -v /home/ubuntu/mysite:/usr/share/nginx/html:ro nginx
```

`:ro` at the end makes the mount read-only inside the container — Nginx can read files but not write them.

```bash
curl http://localhost:8081
```

Now edit the file on the host:

```bash
echo '<h1>Updated from host</h1>' > /home/ubuntu/mysite/index.html
curl http://localhost:8081  # shows the update immediately
```

Bind mounts are useful for development: edit files on the host, the container sees changes immediately. Named volumes are better for production data.

---

## Custom Networks

Create a network and run containers on it:

```bash
docker network create mynet

docker run -d --name app1 --network mynet nginx
docker run -d --name app2 --network mynet nginx

# app1 can reach app2 by name
docker exec app1 curl -s http://app2 | head -5
```

Containers on the same network resolve each other by container name. This is how `docker-compose` services communicate in Lab 11.

Cleanup:

```bash
docker rm -f app1 app2
docker network rm mynet
```

---

## Container Resource Limits

Containers can be constrained to specific CPU and memory:

```bash
docker run -d --name limited \
  --memory 256m \
  --cpus 0.5 \
  nginx

docker stats limited  # shows real-time resource usage, Ctrl+C to exit
```

`--memory 256m` limits the container to 256 MB RAM. `--cpus 0.5` allows at most 50% of one CPU core. If the container tries to use more memory, the OOM (Out of Memory) killer terminates it.

```bash
docker rm -f limited
```

---

## Cleanup Commands

```bash
# Remove all stopped containers
docker container prune

# Remove unused images
docker image prune

# Remove unused volumes
docker volume prune

# Nuclear option: remove everything unused
docker system prune
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

In Lab 10 — Docker Fundamentals On A VM, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What containers are: isolated Linux processes using namespaces and cgroups — not VMs
- Images vs containers: image is a blueprint, container is a running instance
- `docker run`: flags `-d`, `--name`, `-p`, `-v`, `--network`, `--memory`, `--cpus`
- `docker ps`, `docker ps -a`: running vs all containers
- `docker exec -it`: getting a shell inside a running container
- `docker logs`: how containerized apps should log to stdout
- Container lifecycle: run → stop → start → rm
- Named volumes vs bind mounts: when to use each
- Custom networks: containers resolving each other by name
- Resource limits: memory and CPU constraints
- Cleanup commands: prune

---

## What Is Next

**Lab 11 — Compose Multi-Service App**

Running containers one at a time with `docker run` works for learning. For real applications with multiple services — app, database, cache, reverse proxy — you need Docker Compose. It defines your entire stack in one YAML file and brings it all up with one command.
