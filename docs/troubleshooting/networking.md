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
