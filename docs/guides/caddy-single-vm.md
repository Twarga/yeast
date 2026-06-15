# Caddy Single VM

The `caddy-single-vm` template creates one Ubuntu VM and provisions Caddy.

```bash
mkdir caddy-lab
cd caddy-lab
yeast init --template caddy-single-vm
yeast up
```

Verify inside the VM:

```bash
yeast exec web -- systemctl is-active caddy
yeast exec web -- curl -fsS http://127.0.0.1
```

Clean up:

```bash
yeast down
yeast destroy
```
