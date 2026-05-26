# Yeast v0.7.0 Manual Test Tutorial

This is the real host manual test for the current `v0.7.0` candidate.

It assumes:

- you are testing on Linux
- you are using the built binary from the repo
- you want to validate both:
  - the single-VM provisioning/reset loop
  - the first two-VM private lab network
  - built-in template listing and initialization

Use the built binary directly:

```fish
cd ~/Projects/yeast
set BIN ./dist/yeast-linux-amd64
```

## Fast Path

Run the full smoke suite:

```fish
cd ~/Projects/yeast
TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

That script now proves:

- `yeast init --list-templates`
- `yeast init --template caddy-single-vm`
- lifecycle path
- `disk_size`, `hostname`, and `ssh_port`
- provisioning during `yeast up`
- `yeast exec`
- `yeast copy` in both directions
- `yeast inspect`
- `yeast logs`
- `yeast provision` reruns
- stopped-VM snapshot create/list/restore/delete
- two-VM private lab boot
- visible `LAB IP` values in `yeast status`
- guest-to-guest lab TCP reachability
- negative JSON contracts for:
  - invalid config
  - bad project/state
  - invalid network CIDR/IP/reference
  - duplicate private lab IPs
  - missing templates

### Smoke Script Modes

```fish
TEST_MODE=full ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=positive ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
TEST_MODE=negative ./scripts/manual-smoke.sh ./dist/yeast-linux-amd64
```

- `full`: single-VM provisioning/reset + two-VM lab networking + negative checks
- `positive`: only the real VM and lab workflows
- `negative`: only error-path contract checks

## What v0.7.0 Proves

This candidate can now:

- list built-in templates
- initialize a normal editable project from a built-in template
- boot and manage real QEMU/KVM guests
- provision them after SSH readiness
- snapshot and restore a stopped VM
- attach one project-level private lab network
- assign one static lab IPv4 per attached guest
- keep management SSH separate from lab traffic
- expose `LAB IP` in status output
- run one-shot commands inside the guest
- move files in and out of the guest
- expose VM runtime logs and detailed instance state

This still does not prove:

- remote template downloads
- template registries
- complex template variables
- bridge mode
- DHCP
- multiple private networks
- multi-network topology editing
- LabsBackery / MCP / cloud worker flows

## Manual Flow

If you want to run the same logic by hand instead of the smoke script, use these two reference examples:

- `yeast init --template caddy-single-vm`
- `yeast init --template two-vm-lab`

### Single-VM flow

Create the project:

```fish
mkdir -p /tmp/yeast-caddy-template-test
cd /tmp/yeast-caddy-template-test
yeast init --template caddy-single-vm
```

Run:

```fish
yeast doctor
yeast pull ubuntu-24.04
yeast up
yeast status
yeast ssh web
```

Validate inside the guest:

- `hostname` returns `caddy-lab`
- `sudo systemctl is-active caddy` returns `active`
- `curl -fsS http://127.0.0.1` returns the provisioned page

Then validate:

- `yeast exec web -- whoami`
- `yeast copy web --to-guest ./artifact.txt /home/yeast/artifact.txt`
- `yeast copy web --from-guest /home/yeast/artifact.txt ./artifact-out.txt`
- `yeast inspect web`
- `yeast logs web --tail 20`
- `yeast provision web`
- `yeast down`
- `yeast snapshot web clean`
- `yeast snapshots web`
- `yeast restore web clean`
- `yeast delete-snapshot web clean`
- `yeast destroy`

### Two-VM lab flow

Create the project:

```fish
mkdir -p /tmp/yeast-two-vm-template-test
cd /tmp/yeast-two-vm-template-test
yeast init --template two-vm-lab
```

Then start it:

```fish
yeast up
yeast status
```

Expected:

- `attacker` management SSH on `127.0.0.1:2305`
- `target` management SSH on `127.0.0.1:2306`
- `LAB IP` values:
  - `10.10.10.10`
  - `10.10.10.20`

Then verify inside the guests:

```fish
yeast ssh attacker
ip -4 addr show yeastlab0
bash -lc 'echo > /dev/tcp/10.10.10.20/22'
exit

yeast ssh target
ip -4 addr show yeastlab0
bash -lc 'echo > /dev/tcp/10.10.10.10/22'
exit
```

What this proves:

- the lab NIC exists inside each guest
- each guest got the configured static private IP
- guest-to-guest traffic works over the private lab network
- management SSH remains a separate path

## Pass Criteria

Call `v0.7.0` ready only if all of these hold:

- template list/init smoke passes
- single-VM provisioning/reset smoke passes
- guest-control smoke passes for exec/copy/logs/inspect
- two-VM private lab smoke passes
- `LAB IP` appears in both human and JSON status
- guest-to-guest lab TCP reachability works
- negative JSON checks still pass for config/state/network failures
