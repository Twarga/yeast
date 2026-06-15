# Lab 06: Multi-VM Private Networking

In this lab, you will start two VMs on one private Yeast network.

You will learn:

- project-level `networks`
- instance network attachments
- static IPv4 addresses
- VM-to-VM connectivity

## What You Will Build

```text
host
└── Yeast project
    ├── attacker VM 10.10.10.10
    └── target VM   10.10.10.20
```

## Create The Project

```bash
mkdir yeast-lab-06
cd yeast-lab-06
yeast init --template two-vm-lab
```

## Start Both VMs

```bash
yeast up
yeast status
```

## Verify Connectivity

From `attacker`, ping `target`:

```bash
yeast exec attacker -- ping -c 2 10.10.10.20
```

From `target`, ping `attacker`:

```bash
yeast exec target -- ping -c 2 10.10.10.10
```

## What Happened

The private network is for VM-to-VM traffic.

The SSH port is for host-to-VM management.

Those are separate ideas.

## Clean Up

```bash
yeast down
yeast destroy
```

Next: [Templates And Reusable Labs](07-templates-and-reusable-labs.md).
