# Networking

Yeast has two networking ideas:

- management SSH from host to VM
- optional private VM-to-VM lab networking

## Management SSH

Each VM gets a host SSH port.

```yaml
instances:
  - name: web
    image: ubuntu-24.04
    ssh_port: 2222
```

If `ssh_port` is omitted, Yeast chooses a port starting at `2222`.

## Private Lab Network

Yeast v1.1 supports one private project network.

```yaml
version: 1
networks:
  - name: lab
    cidr: 10.10.10.0/24
instances:
  - name: attacker
    image: ubuntu-24.04
    networks:
      - name: lab
        ipv4: 10.10.10.10
  - name: target
    image: ubuntu-24.04
    networks:
      - name: lab
        ipv4: 10.10.10.20
```

## Current Limits

- one private network per project
- at most one private network attachment per VM
- static IPv4 only
- no DHCP
- no bridge mode
- no documented `ports:` support in v1.1
