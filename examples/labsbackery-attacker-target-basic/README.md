# attacker-target-basic

LabsBakery-ready attacker/target example for Yeast.

This is a local Yeast template package with extra LabsBakery-owned metadata. Yeast consumes `template.yaml`, `yeast.yaml`, and provision files. LabsBakery consumes `lab.yaml`, `scenario/instructions.md`, and `scenario/checks.yaml`.

## What this example proves

- two Ubuntu 24.04 VMs
- one private lab network
- static lab IPs:
  - `attacker` -> `10.20.30.10`
  - `target` -> `10.20.30.20`
- target-side file provisioning
- terminal connection metadata through `status --json`
- check commands that LabsBakery can run through `yeast exec`
- clean baseline snapshot/reset workflow

## Files

- `template.yaml` - local Yeast template metadata
- `yeast.yaml` - Yeast VM/network/provisioning config
- `lab.yaml` - LabsBakery-owned lab metadata
- `files/target/flag.txt` - provisioned target marker file
- `scenario/instructions.md` - learner instructions
- `scenario/checks.yaml` - LabsBakery-owned check definitions

## Initialize

From a fresh directory:

```bash
mkdir my-labsbackery-lab
cd my-labsbackery-lab
yeast init --template /path/to/yeast/examples/labsbackery-attacker-target-basic
```

## Run

```bash
yeast doctor
yeast pull ubuntu-24.04
yeast up --json --events
yeast status --json
```

Expected `status --json` data:

- `attacker` is `running`
- `target` is `running`
- both instances include `management_ip`, `ssh_port`, and `user`
- `attacker.lab_ip` is `10.20.30.10`
- `target.lab_ip` is `10.20.30.20`

## Validate

Run the checks manually the same way LabsBakery can run them:

```bash
yeast exec attacker --json -- bash -lc 'echo > /dev/tcp/10.20.30.20/22'
yeast exec target --json -- bash -lc 'test -f /home/yeast/labsbackery-target.txt && grep -q labsbackery-ready /home/yeast/labsbackery-target.txt'
```

## Baseline And Reset

Create a clean baseline while the lab is stopped:

```bash
yeast down --json --events
yeast snapshot attacker clean --description "Clean attacker baseline" --json
yeast snapshot target clean --description "Clean target baseline" --json
```

Reset the lab:

```bash
yeast down --json --events
yeast restore attacker clean --json --events
yeast restore target clean --json --events
yeast up --json --events
```

## Cleanup

```bash
yeast destroy --json --events
```

## Boundary

Yeast does not parse `lab.yaml`, instructions, or check definitions. LabsBakery owns those files and calls Yeast through the JSON/event contract.
