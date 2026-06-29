# Networking

Yeast has two networking ideas:

- management SSH from host to VM
- optional host-to-guest service port forwarding
- optional private VM-to-VM lab networking

## Management SSH

Each VM gets a host SSH port.

```yaml
instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2222
```

If `ssh_port` is omitted, Yeast chooses a port starting at `2222`.

`management_host` controls which host IP the management port binds to:

```yaml
management_host: 127.0.0.1
```

Default `127.0.0.1` means the SSH port is local to the host.

Use `0.0.0.0` only when you intentionally want the management port reachable from outside the host.

## Service Port Forwarding

Yeast now supports Docker-style service forwards directly in `yeast.yaml`.

Simple syntax:

```yaml
instances:
  - name: web
    image: ubuntu-24.04
    ports:
      - "8080:80"
```

That means:

- your laptop connects to `127.0.0.1:8080`
- Yeast forwards that traffic to guest port `80`

Advanced object syntax:

```yaml
instances:
  - name: monitoring
    image: ubuntu-24.04
    ports:
      - name: grafana
        host: 127.0.0.1
        host_port: 3000
        guest_port: 3000
      - name: prometheus
        host_port: 9090
        guest_port: 9090
```

Rules in this patch:

- default bind host is `127.0.0.1`
- TCP only
- host port collisions are rejected before boot
- `yeast up` and `yeast status` print the forwarded host URLs

## Private Lab Network

Yeast v1.1 supports one private project network.

```yaml
version: 1
networks:
  - name: lab
    cidr: 10.10.10.0/24
instances:
  - name: attacker
    image: ubuntu-24.04
    networks:
      - name: lab
        ipv4: 10.10.10.10
  - name: target
    image: ubuntu-24.04
    networks:
      - name: lab
        ipv4: 10.10.10.20
```

## Current Limits

- one private network per project
- at most one private network attachment per VM
- static IPv4 only
- no DHCP
- no bridge mode
- TCP only for service forwards

## Address Rules

Private network addresses must:

- be IPv4
- be inside the configured CIDR
- not be the network address
- not be the broadcast address
- not be duplicated by another instance on the same network

Good:

```yaml
networks:
  - name: lab
    cidr: 10.10.10.0/24
instances:
  - name: web
    image: ubuntu-24.04
    networks:
      - name: lab
        ipv4: 10.10.10.10
```

Bad:

```yaml
networks:
  - name: lab
    cidr: 10.10.10.0/24
instances:
  - name: web
    image: ubuntu-24.04
    networks:
      - name: lab
        ipv4: 10.10.11.10
```

The bad address is outside `10.10.10.0/24`.

## Mental Model

Management SSH is for you and tools on the host.

Private networking is for VM-to-VM traffic inside the lab.

Do not use private lab IPs as a replacement for `yeast ssh`; use:

```bash
yeast ssh web
yeast exec web -- ip addr
```
