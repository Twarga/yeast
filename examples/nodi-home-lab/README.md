# nodi-home-lab

Complex multi-VM home lab proof for Yeast.

> **New to Yeast?**
> Start with the official docs path first. This is an advanced example and is not part of the beginner docs path yet.

What this example does:

- boots **4 Ubuntu 24.04 VMs** orchestrated by a single `yeast.yaml`
- assigns each VM a role on one private lab network (`192.168.2.0/24`):
  - `gateway` (`192.168.2.1`) — Caddy landing page + service directory
  - `storage` (`192.168.2.50`) — Nodi file manager + Samba + NFS
  - `alpha` (`192.168.2.11`) — Dev workstation with NFS mounts
  - `beta` (`192.168.2.12`) — Backup workstation with SMB mounts
- sets up **real inter-VM services**: file shares, web UI, client mounts, sync
- exposes services inside the lab network for verification with `yeast exec`
- uses provisioning retries so VMs with dependencies tolerate startup order

What this proves about Yeast:

- multi-VM orchestration from one project config
- private lab networking with static IPs
- host-to-guest port forwarding for services
- heavy post-boot provisioning (Docker install, Nodi build, Samba, NFS)
- cross-VM dependency handling via retry loops
- guest control verification (`yeast exec` across multiple VMs)
- snapshot/reset of a complex multi-service baseline

## Files

| File | Purpose |
|---|---|
| `yeast.yaml` | 4-instance config with network and provisioning |
| `files/gateway/Caddyfile` | Static web server config for the landing page |
| `files/gateway/index.html` | Service directory landing page |
| `files/storage/smb.conf` | Samba share definitions |
| `scripts/verify.sh` | Host-side automated verification script |

## Prerequisites

- Linux host with KVM/QEMU
- `yeast` installed
- Internet access from VMs (Nodi install pulls from GitHub)
- At least **~5 GB free RAM** and **~40 GB disk** for all 4 VMs

## Run

Create a fresh project and copy the example:

```bash
mkdir my-nodi-lab
cd my-nodi-lab
yeast init
cp -r /path/to/yeast/examples/nodi-home-lab/* ./
```

Check the host and start the lab:

```bash
yeast doctor
yeast up
```

`yeast up` downloads the Ubuntu image automatically if it is not cached yet.

**First boot takes time.** The `storage` VM downloads and builds Nodi inside Docker. Total first-up time is typically **3–8 minutes** depending on host internet speed.

Check status:

```bash
yeast status
```

Expected shape:

- `gateway` running, SSH on `127.0.0.1:2222`, lab IP `192.168.2.1`
- `storage` running, SSH on `127.0.0.1:2223`, lab IP `192.168.2.50`
- `alpha` running, SSH on `127.0.0.1:2224`, lab IP `192.168.2.11`
- `beta` running, SSH on `127.0.0.1:2225`, lab IP `192.168.2.12`

## Inspect The Lab

Public host port mappings are not part of Yeast v1.1. Inspect the services from inside the VMs:

```bash
yeast exec gateway -- curl -fsS http://localhost
yeast exec storage -- curl -fsS http://localhost:7319
```

For Nodi login details, check `~/.yeast/projects/<project-id>/instances/storage/provision.log` for the generated password, or log in with the default Nodi credentials if they apply.

## Verify Automatically

Run the provided verification script from the project root:

```bash
bash scripts/verify.sh
```

It checks:

- gateway landing page returns HTTP 200
- Nodi UI is reachable
- alpha has NFS mounts and shared files
- beta has SMB mounts and synced backup data
- cross-VM ping works (alpha → storage, beta → storage, alpha → beta)

## Manual Verification

Explore each VM:

```bash
# Gateway — check Caddy is serving
yeast ssh gateway
systemctl is-active caddy
exit

# Storage — check Nodi, Samba, NFS
yeast ssh storage
systemctl is-active nodi || systemctl status nodi
cat /etc/exports
showmount -e localhost
exit

# Alpha — check NFS mounts and shared file
yeast ssh alpha
mount | grep nfs
cat /mnt/nfs/shared/readme.txt
exit

# Beta — check SMB mounts and backup sync
yeast ssh beta
mount | grep cifs
ls /mnt/smb/backup/beta-workspace/
exit
```

## Snapshot The Baseline

Once the lab is verified, snapshot the clean state:

```bash
yeast down
yeast snapshot storage baseline --description "Nodi + Samba + NFS provisioned"
yeast snapshot alpha baseline
yeast snapshot beta baseline
yeast snapshot gateway baseline
yeast up
```

Later, after breaking or testing:

```bash
yeast down
yeast restore storage baseline
yeast restore alpha baseline
yeast restore beta baseline
yeast restore gateway baseline
yeast up
```

## Stop Or Remove

```bash
yeast down
yeast destroy
```

## Architecture Notes

- **One flat private network:** All VMs share `192.168.2.0/24`. There is no router VM because Yeast supports only one lab network and max 2 NICs per VM.
- **No DHCP:** All lab IPs are static, written by cloud-init.
- **Client resilience:** `alpha` and `beta` use retry loops to mount shares because VMs provision in parallel. They will wait up to 5 minutes for `storage` to finish.
- **Service access:** v1.1 does not expose public service port mappings. Use `yeast exec` from the host, or guest-to-guest lab IPs inside the private network.
- **Windows replacement:** `alpha` and `beta` are Ubuntu VMs that exercise the same SMB/CIFS and NFS client protocols Windows would use.

## Limits

- `yeast up` first boot is slow because Nodi builds a Docker image inside the VM.
- If `storage` provisioning fails (e.g., no internet), `alpha`/`beta` will also fail after retries.
- Single flat network only; there is no inter-subnet routing demo.
- No Windows GUI or Active Directory.

## What This Is

This is not a GNS3 clone. It is a **Yeast-native proof** that Yeast can orchestrate a realistic, multi-service, multi-VM home lab with shared storage, clients, and web services — entirely from one `yeast.yaml` file.
