# Two VM Lab

The `two-vm-lab` template creates two Ubuntu VMs on one private network.

```bash
mkdir two-vm-lab
cd two-vm-lab
yeast init --template two-vm-lab
yeast up
```

Verify connectivity:

```bash
yeast exec attacker -- ping -c 2 10.10.10.20
yeast exec target -- ping -c 2 10.10.10.10
```

Clean up:

```bash
yeast down
yeast destroy
```
