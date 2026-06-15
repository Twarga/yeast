# Limitations

Current Yeast v1.1 limits:

- Linux host only
- AMD64/x86_64 only
- QEMU/KVM runtime
- one private network per project
- at most one private network attachment per VM
- static IPv4 only for private networks
- no DHCP
- no bridge mode
- no Windows guests
- stopped-VM snapshots only
- no project-wide atomic snapshot helper
- no public `ports:` config support
- no daemon or web API
- no remote template registry
