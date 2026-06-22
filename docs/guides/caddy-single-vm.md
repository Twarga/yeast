# Caddy Single VM

The `caddy-single-vm` template creates one Ubuntu VM and provisions Caddy with a small static site.

## Create The Project

```bash
mkdir caddy-lab
cd caddy-lab
yeast init --template caddy-single-vm
```

Inspect the template files:

```bash
ls -la
ls -la site
cat yeast.yaml
```

## Start The VM

```bash
yeast up
```

Yeast installs Caddy, copies the site files, installs the Caddyfile, enables Caddy, and restarts the service.

## Verify Inside The VM

```bash
yeast exec web -- systemctl is-active caddy
yeast exec web -- curl -fsS http://127.0.0.1
```

Expected service output:

```text
active
```

## Rerun Provisioning

If you edit `site/index.html` or `site/Caddyfile`, rerun provisioning:

```bash
yeast provision web
```

Then verify again:

```bash
yeast exec web -- curl -fsS http://127.0.0.1
```

## Clean Up

```bash
yeast down
yeast destroy
```

## What This Template Demonstrates

- top-level provisioning
- package installation
- file copy into the guest
- shell commands with `sudo`
- service verification through `yeast exec`
