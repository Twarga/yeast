# Accessing Services In Yeast v1.1 Labs

Yeast v1.1 forwards SSH management ports. It does not provide a documented general `ports:` feature for exposing guest HTTP services to the host.

That means this is correct:

```yaml
instances:
  - name: web
    ssh_port: 2202
```

This is not supported in `yeast.yaml` v1.1:

```yaml
instances:
  - name: web
    ports:
      - "8080:80"
```

Docker Compose and Kubernetes still have their own `ports:` fields inside their own files. That is fine. The unsupported part is using `ports:` in `yeast.yaml`.

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

you must know where you are running it. If you are SSHed into the VM, it tests the service inside that VM. If you are on your laptop, it only works after you create a tunnel.

## Check The VM SSH Port

From the lab folder, run:

```bash
yeast status
```

For machine-readable output:

```bash
yeast status --json
```

Look for:

- `management_ip`
- `ssh_port`
- `user`

Most bootcamp labs use:

- `management_ip`: `127.0.0.1`
- `user`: `ubuntu`
- an explicit `ssh_port` in `assets/yeast.yaml`

## Create A Local Browser Tunnel

Use normal SSH local forwarding:

```bash
ssh -N -L <local_port>:127.0.0.1:<service_port> -p <ssh_port> ubuntu@127.0.0.1
```

Example for a VM named `monitoring` with `ssh_port: 2226`, exposing Prometheus on VM port `9090`:

```bash
ssh -N -L 9090:127.0.0.1:9090 -p 2226 ubuntu@127.0.0.1
```

Keep that terminal open. In another terminal, open:

```text
http://localhost:9090
```

Stop the tunnel with `Ctrl+C`.

## Multiple Services From One VM

You can forward more than one port through the same SSH connection:

```bash
ssh -N \
  -L 9090:127.0.0.1:9090 \
  -L 3000:127.0.0.1:3000 \
  -p 2226 ubuntu@127.0.0.1
```

Then open:

```text
http://localhost:9090
http://localhost:3000
```

## Service On Another VM

If the service is on a private lab IP reachable from the SSH target, forward to that private IP:

```bash
ssh -N -L 8080:192.168.10.10:80 -p 2203 ubuntu@127.0.0.1
```

This means:

- browser connects to your laptop on `localhost:8080`
- SSH carries that traffic through VM `proxy`
- VM `proxy` connects to `192.168.10.10:80`

Prefer tunneling to the VM that actually runs the service when possible:

```bash
ssh -N -L 8080:127.0.0.1:80 -p 2203 ubuntu@127.0.0.1
```

## Common Lab Tunnel Examples

| Lab | Service | Tunnel |
|---|---|---|
| 02 | Nginx on `web`, port 80 | `ssh -N -L 8080:127.0.0.1:80 -p 2202 ubuntu@127.0.0.1` |
| 03 | Proxy on `proxy`, port 80 | `ssh -N -L 8080:127.0.0.1:80 -p 2203 ubuntu@127.0.0.1` |
| 11 | Compose app on `compose`, port 8080 | `ssh -N -L 8080:127.0.0.1:8080 -p 2217 ubuntu@127.0.0.1` |
| 17 | Prometheus/Grafana on `monitoring` | `ssh -N -L 9090:127.0.0.1:9090 -L 3000:127.0.0.1:3000 -p 2226 ubuntu@127.0.0.1` |
| 18 | Loki/Grafana on `logs` | `ssh -N -L 3100:127.0.0.1:3100 -L 3000:127.0.0.1:3000 -p 2228 ubuntu@127.0.0.1` |
| 19 | Jaeger on `tracing` | `ssh -N -L 16686:127.0.0.1:16686 -p 2230 ubuntu@127.0.0.1` |
| 20 | Prometheus/Grafana/Alertmanager on `sre` | `ssh -N -L 9090:127.0.0.1:9090 -L 3000:127.0.0.1:3000 -L 9093:127.0.0.1:9093 -p 2233 ubuntu@127.0.0.1` |
| 25 | Argo CD on `gitops-cluster` after `kubectl port-forward` inside VM | `ssh -N -L 8080:127.0.0.1:8080 -p 2240 ubuntu@127.0.0.1` |
| 30 | Ollama API on `llm` | `ssh -N -L 11434:127.0.0.1:11434 -p 2252 ubuntu@127.0.0.1` |

## Safer Wording For Labs

Use this wording:

> Inside the VM, test the service with `curl http://localhost:8080`.

Use this wording when the learner needs a browser:

> From your laptop, first create an SSH tunnel. Keep it open, then browse to `http://localhost:8080`.

Avoid this wording:

> Yeast maps port 8080 on your laptop to port 80 inside the VM.

That is not a Yeast v1.1 feature.
