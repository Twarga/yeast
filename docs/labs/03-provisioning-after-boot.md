# Lab 03: Provisioning After Boot

Use Yeast provisioning to install packages, copy files, and run shell commands after the VM is reachable.

You will learn:

- what `provision.packages` does
- what `provision.files` does
- what `provision.shell` does
- how top-level provisioning applies to instances
- when to use `yeast provision`
- when to use `yeast up --reprovision` or `yeast up --no-provision`

## What You Will Build

```text
yeast-lab-03/
└── web VM
    ├── Ubuntu 24.04
    ├── Caddy installed
    ├── static index.html copied
    └── Caddy restarted by shell commands
```

## Before You Start

Run:

```bash
yeast doctor
```

## Step 1: Create The Project

```bash
mkdir yeast-lab-03
cd yeast-lab-03
yeast init --template caddy-single-vm
```

Inspect the provisioning config:

```bash
sed -n '1,120p' yeast.yaml
find site -maxdepth 2 -type f | sort
```

The template uses top-level `provision`, which applies to the `web` instance.

## Step 2: Start The VM

```bash
yeast up
```

After SSH becomes ready, Yeast installs packages, copies files, and runs shell commands from `yeast.yaml`.

## Step 3: Verify Provisioning

```bash
yeast exec web -- systemctl is-active caddy
```

Expected output:

```text
active
```

Check the copied file:

```bash
yeast exec web -- test -f /var/www/html/index.html
```

## Step 4: Read Logs

```bash
yeast logs web --tail 80
```

Use logs when the VM starts but something inside the boot or runtime path looks wrong.

## Step 5: Rerun Provisioning On A Running VM

```bash
yeast provision web
```

Use this when the VM is already running and you want to reapply provisioning without recreating the VM.

## Step 6: Compare Provisioning Flags

Force provisioning during `up`:

```bash
yeast up --reprovision
```

Skip provisioning during `up`:

```bash
yeast up --no-provision
```

These flags are useful when debugging provisioning separately from VM startup.

## Clean Up

```bash
yeast down
yeast destroy
```

## What You Learned

Cloud-init prepares access. Provisioning turns a reachable VM into the machine you want.

Provisioning is Yeast's built-in way to make small, repeatable setup steps part of the project.

## Next Lab

Continue with [Status, Logs, Inspect, JSON](04-status-logs-inspect-json.md).
