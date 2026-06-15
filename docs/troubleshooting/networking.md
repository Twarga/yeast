# Networking Troubleshooting

Check status:

```bash
yeast status
```

For private network labs:

```bash
yeast exec attacker -- ip addr
yeast exec target -- ip addr
yeast exec attacker -- ping -c 2 10.10.10.20
```

Common causes:

- IP outside the network CIDR
- duplicate static IP
- more than one private network in the project
- confusing management SSH ports with private lab IPs

## Check The Config

```bash
sed -n '1,220p' yeast.yaml
```

Confirm:

- there is at most one top-level network
- each attached instance references that network by name
- each attached instance has `ipv4`
- each `ipv4` is inside the CIDR

## Management Port Conflicts

If `yeast up` says a port is busy, either stop the process using that port or choose a different `ssh_port`.

Example:

```yaml
instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2223
```
