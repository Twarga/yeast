# Accessing Services In Yeast Labs

Use Yeast service port forwarding first. Use manual SSH tunnels only when a lab explicitly needs them as a fallback.

## The Easy Path

If a lab `yeast.yaml` contains:

```yaml
instances:
  - name: web
    ssh_port: 2202
    ports:
      - "8080:80"
```

then after `yeast up` you can use:

- `yeast ssh web` for the VM shell
- `http://127.0.0.1:8080` from your laptop browser

You do not need a manual `ssh -L` tunnel for that service.

## Three Places `localhost` Can Mean

Be careful with `localhost`:

| Where the command runs | What `localhost` means |
|---|---|
| Your laptop | Your laptop |
| Inside a Yeast VM | That VM |
| Inside a container | That container |

If a lab says:

```bash
curl http://localhost:8080
```

you still need to notice where that command runs.

## Check The Forwarded Ports

From the lab folder, run:

```bash
yeast up
yeast status
```

Look for:

- `SSH`
- `PORTS`
- the host URL Yeast prints for the service

## Common Lab Browser Ports

| Lab | Service | Host URL |
|---|---|---|
| 02 | Nginx | `http://127.0.0.1:8080` |
| 03 | Reverse proxy | `http://127.0.0.1:8080` |
| 11 | Compose app | `http://127.0.0.1:8080` |
| 17 | Prometheus and Grafana | `http://127.0.0.1:9090`, `http://127.0.0.1:3000` |
| 18 | Loki and Grafana | `http://127.0.0.1:3100`, `http://127.0.0.1:3000` |
| 20 | Prometheus, Grafana, Alertmanager | `http://127.0.0.1:9090`, `http://127.0.0.1:3000`, `http://127.0.0.1:9093` |
| 30 | Ollama API and UI | `http://127.0.0.1:11434`, `http://127.0.0.1:3000` |

## When To Still Use A Manual Tunnel

Use a manual SSH tunnel only when:

- the lab intentionally teaches SSH forwarding
- the service is not declared in `ports:`
- you need a one-off path through another VM on the private lab network

Fallback example:

```bash
ssh -N -L 8080:127.0.0.1:80 -p 2202 ubuntu@127.0.0.1
```

## Safer Lab Wording

Prefer this wording in labs:

> From your laptop, open the forwarded Yeast URL shown by `yeast up` or `yeast status`.

Fallback wording:

> If the forwarded port is not configured for this step, create an SSH tunnel from your laptop and keep it open while you test.
