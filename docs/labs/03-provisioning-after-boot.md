# Lab 03: Provisioning After Boot

In this lab, you will use Yeast provisioning to install software and run setup commands after the VM is reachable.

You will learn:

- `packages`
- `files`
- `shell`
- `yeast provision`
- `yeast up --reprovision`
- `yeast up --no-provision`

## Create The Project

```bash
mkdir yeast-lab-03
cd yeast-lab-03
yeast init --template caddy-single-vm
```

## Start The VM

```bash
yeast up
```

## Verify Provisioning

```bash
yeast exec web -- systemctl is-active caddy
yeast logs web --tail 80
```

Expected:

```text
active
```

## Rerun Provisioning

```bash
yeast provision web
```

Use this when the VM is already running and you want to reapply provisioning.

## Force Provisioning During Up

```bash
yeast up --reprovision
```

## Skip Provisioning During Up

```bash
yeast up --no-provision
```

## Clean Up

```bash
yeast down
yeast destroy
```

Next: [Status, Logs, Inspect, JSON](04-status-logs-inspect-json.md).
