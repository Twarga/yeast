# wireguard-vpn-mesh

A 3-VM WireGuard VPN mesh for Yeast.

## What it does

- `hub` — WireGuard server (10.200.200.1/24)
- `spoke1` — WireGuard client (10.200.200.2/24)
- `spoke2` — WireGuard client (10.200.200.3/24)

Spokes connect to hub. Once connected, all nodes can reach each other over the encrypted tunnel using `10.200.200.x` addresses.

## IMPORTANT: Generate Real Keys

The included `wg0.conf` files contain **dummy keys** for structure demonstration only. WireGuard will not establish a real tunnel with these keys.

Generate proper keys before deploying:

```bash
# For each node
cd files/hub
wg genkey | tee private.key | wg pubkey > public.key
# Repeat for files/spoke1 and files/spoke2
```

Then update the `PublicKey` fields in each `.conf` file to match the corresponding node's public key.

## Quick start

```bash
mkdir my-wg-lab && cd my-wg-lab
yeast init
cp -r /path/to/yeast/examples/wireguard-vpn-mesh/* ./
# Generate real keys first!
yeast pull ubuntu-24.04
yeast up
bash scripts/verify.sh
```

## Full tutorial

See [Tutorial 13: WireGuard VPN Mesh](../../tutorials/13-wireguard-vpn-mesh.md).
