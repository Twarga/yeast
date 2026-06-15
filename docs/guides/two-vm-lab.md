# Two VM Lab

The `two-vm-lab` template creates two Ubuntu VMs on one private Yeast network.

## Create The Project

```bash
mkdir two-vm-lab
cd two-vm-lab
yeast init --template two-vm-lab
```

Inspect the network config:

```bash
sed -n '1,180p' yeast.yaml
```

You should see:

- one network named `lab`
- CIDR `10.10.10.0/24`
- `attacker` at `10.10.10.10`
- `target` at `10.10.10.20`

## Start Both VMs

```bash
yeast up
yeast status
```

## Verify Connectivity

From `attacker` to `target`:

```bash
yeast exec attacker -- ping -c 2 10.10.10.20
```

From `target` to `attacker`:

```bash
yeast exec target -- ping -c 2 10.10.10.10
```

## Inspect State

```bash
yeast inspect attacker
yeast inspect target
```

Use `inspect` when you need detailed per-instance state.

## Clean Up

```bash
yeast down
yeast destroy
```

## What This Template Demonstrates

- one private project network
- static per-instance IPv4 addresses
- VM-to-VM traffic over lab IPs
- host-to-VM management through Yeast commands
