# load-balancer-lab

A 3-VM load balancer lab for Yeast.

## What it does

- `proxy` — Caddy reverse proxy with round-robin load balancing inside the lab network
- `web1` — Python Flask app returning "web1"
- `web2` — Python Flask app returning "web2"

Verify the proxy through `yeast exec proxy -- curl http://localhost`. Public host port mappings are not part of Yeast v1.1.

## Quick start

```bash
mkdir my-lb-lab && cd my-lb-lab
yeast init
cp -r /path/to/yeast/examples/load-balancer-lab/* ./
yeast up
bash scripts/verify.sh
```

`yeast up` downloads the Ubuntu image automatically if it is not cached yet.

## Verify Manually

```bash
yeast exec proxy -- curl -fsS http://localhost
```

## Note

This is an advanced example, not part of the beginner docs path yet.
