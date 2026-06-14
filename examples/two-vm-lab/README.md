# two-vm-lab

Minimal attacker/target private-network example for Yeast.

What this example does:

- boots two Ubuntu 24.04 VMs
- keeps normal management SSH on host-forwarded ports
- attaches both VMs to one private lab network
- assigns static lab IPs:
  - `attacker` -> `10.10.10.10`
  - `target` -> `10.10.10.20`

What this example does not do:

- bridge mode
- DHCP
- multiple private networks
- guest exec/copy/logs commands
- automatic cross-guest validation from the CLI

## Files

- `yeast.yaml` - two VMs plus one project-level private lab network

## Run

Create a fresh project directory first. `yeast init` creates the project metadata and starter config.

```bash
mkdir my-two-vm-lab
cd my-two-vm-lab
yeast init
cp /path/to/yeast/examples/two-vm-lab/yeast.yaml ./yeast.yaml
```

Then run:

```bash
yeast doctor
yeast pull ubuntu-24.04
yeast up
yeast status
```

Expected status shape:

- `attacker` is reachable over management SSH on `127.0.0.1:2205`
- `target` is reachable over management SSH on `127.0.0.1:2206`
- `yeast status` shows `LAB IP` values:
  - `10.10.10.10`
  - `10.10.10.20`

## Verify The Lab Network

SSH into each VM through the normal management path:

```bash
yeast ssh attacker
ip addr show yeastlab0
ping -c 2 10.10.10.20
exit

yeast ssh target
ip addr show yeastlab0
ping -c 2 10.10.10.10
exit
```

What this proves:

- management SSH remains separate from lab traffic
- both guests boot with the configured static private IP
- the private lab NIC is usable guest-to-guest

## Stop Or Remove

```bash
yeast down
yeast destroy
```

## Notes

- this is the first narrow private-network slice
- Yeast still uses user-mode SSH forwarding for management
- the private lab NIC is separate from that management path
- the first pass supports exactly one project-level private lab network
