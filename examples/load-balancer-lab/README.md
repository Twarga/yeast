# load-balancer-lab

A 3-VM load balancer lab for Yeast `v1.0`.

## What it does

- `proxy` — Caddy reverse proxy with round-robin load balancing
- `web1` — Python Flask app returning "web1"
- `web2` — Python Flask app returning "web2"

The host accesses `http://127.0.0.1:8080` and requests alternate between `web1` and `web2`.

## Quick start

```bash
mkdir my-lb-lab && cd my-lb-lab
yeast init
cp -r /path/to/yeast/examples/load-balancer-lab/* ./
yeast pull ubuntu-24.04
yeast up
bash scripts/verify.sh
```

## Browse

```
http://127.0.0.1:8080
```

## Full tutorial

See [Tutorial 10: Load Balancer Lab](../../tutorials/10-load-balancer-lab.md) for the complete educational walkthrough.
