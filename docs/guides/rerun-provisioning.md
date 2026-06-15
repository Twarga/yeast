# Rerun Provisioning

Provisioning runs after a VM is reachable over SSH.

Use this guide when you changed `provision` in `yeast.yaml` and want to apply it again.

## Rerun On A Running VM

```bash
yeast provision web
```

Use this when:

- the VM is already running
- you changed package/file/shell provisioning
- you want to debug provisioning without recreating the VM

## Force Provisioning During Up

```bash
yeast up --reprovision
```

Use this when you want `yeast up` to run provisioning even if Yeast thinks the VM was already provisioned.

## Skip Provisioning During Up

```bash
yeast up --no-provision
```

Use this when you want to debug boot or SSH readiness without running provision steps.

## Watch Events

```bash
yeast provision web --json --events
```

This is useful for tools that need progress output.

## Verify

After provisioning, run a command that proves the expected change happened:

```bash
yeast exec web -- systemctl is-active caddy
yeast exec web -- test -f /var/www/html/index.html
```

## Make Provisioning Re-Runnable

Write provisioning so it can run more than once:

- use `install -D` instead of assuming directories exist
- use `systemctl restart` when a service may already be running
- include `sudo` for root-owned paths
- avoid commands that fail if a file already exists

For troubleshooting, see [Provisioning Troubleshooting](../troubleshooting/provisioning.md).
