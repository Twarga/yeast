# Lab 06: Multi-VM Private Networking

Start two VMs on one private Yeast network and verify VM-to-VM traffic.

You will learn:

- how project-level `networks` define a private lab network
- how instance network attachments join that network
- how static IPv4 addresses work
- how management SSH differs from private lab traffic
- why v1.1 supports one private project network

## What You Will Build

```text
Linux host
└── yeast-lab-06/
    └── private lab network 10.10.10.0/24
        ├── attacker VM 10.10.10.10
        └── target VM   10.10.10.20
```

## Before You Start

Run:

```bash
yeast doctor
```

## Step 1: Create The Project

```bash
mkdir yeast-lab-06
cd yeast-lab-06
yeast init --template two-vm-lab
```

Inspect the network config:

```bash
cat yeast.yaml
```

Look for:

- `networks`
- `cidr: 10.10.10.0/24`
- `attacker` address `10.10.10.10`
- `target` address `10.10.10.20`

## Step 2: Start Both VMs

```bash
yeast up
yeast status
```

Expected result:

- `attacker` is running
- `target` is running
- both show management SSH information
- both show lab IP information

## Step 3: Verify Private Connectivity

From `attacker`, ping `target`:

```bash
yeast exec attacker -- ping -c 2 10.10.10.20
```

From `target`, ping `attacker`:

```bash
yeast exec target -- ping -c 2 10.10.10.10
```

## Step 4: Inspect One VM

```bash
yeast inspect attacker
```

Use inspect to see detailed instance state, including management access and lab network information.

## What Happened

The management SSH port is host-to-VM access.

The private lab IP is VM-to-VM access.

Those are separate paths. This matters because tools and people usually connect through management SSH, while lab services communicate over the private network.

## Clean Up

```bash
yeast down
yeast destroy
```

## What You Learned

Yeast can run small multi-VM labs with predictable private addresses.

In v1.1, keep the model simple: one project-level private network, static IPv4 addresses, and explicit instance attachments.

## Next Lab

Continue with [Templates And Reusable Labs](07-templates-and-reusable-labs.md).
