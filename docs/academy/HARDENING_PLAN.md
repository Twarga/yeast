# DevOps Bootcamp Hardening Plan

The draft is structurally complete, but it must be hardened before it is treated as ready.

Goal:

> Every lab should run on a fresh Linux/KVM host with Yeast v1.1 using only supported Yeast commands, supported `yeast.yaml` fields, and verified service access patterns.

## Current Status

| Area | Status |
|---|---|
| 30-lab outline | Done |
| `lab.md` files | Draft complete |
| `assets/yeast.yaml` files | Draft complete, needs runtime validation |
| `assets/validate.sh` files | Shell syntax checked only |
| Real KVM execution | Not done |
| Service access/browser checks | Needs correction |
| Yeast v1.1 feature audit | Started |

## Fix First

### 1. Remove Fake Yeast Port Forwarding Assumptions

Problem:

Some labs imply Yeast maps HTTP ports from the host to guests. Yeast v1.1 only documents management SSH forwarding.

Fix:

- Replace “Yeast maps port ...” language.
- Add SSH tunnel steps before browser instructions.
- Keep inside-VM `curl http://localhost:<port>` instructions when the command is run inside the VM.
- Use `ACCESS.md` tunnel examples.

Priority labs:

- Lab 02: static Nginx
- Lab 03: reverse proxy
- Lab 11: Compose app
- Lab 17: Prometheus/Grafana
- Lab 18: Loki/Grafana
- Lab 19: Jaeger
- Lab 20: Prometheus/Grafana/Alertmanager
- Lab 25: Argo CD
- Lab 30: Ollama API

### 2. Remove Unsupported Yeast Commands And Flags

Problem:

Yeast v1.1 does not support raw SSH passthrough such as:

```bash
yeast ssh web -- -L 8080:127.0.0.1:80
```

Fix:

- Use `yeast ssh <instance>` only for interactive SSH.
- Use normal `ssh -N -L ... -p <ssh_port> ubuntu@127.0.0.1` for tunnels.
- Use `yeast exec <instance> -- <command>` for one-off remote commands.
- Use `yeast copy` for file transfers instead of invented helpers.

### 3. Add A Supported Yeast Checklist To Every Lab

Each lab should include a short Yeast control checkpoint near the start:

```bash
yeast up
yeast status
yeast inspect <instance>
```

For automation-oriented labs, also include:

```bash
yeast status --json
```

Use `yeast logs <instance>` only for VM runtime troubleshooting, not application logs.

### 4. Make Validation Mean More Than "Script Exists"

Problem:

`validate.sh` files are syntax-valid, but not proven against a real host.

Fix:

For each lab:

- Run `yeast up`.
- Complete the lab exactly as written.
- Run `bash assets/validate.sh`.
- Run at least one negative/failure check where the lab teaches troubleshooting.
- Run `yeast destroy`.
- Mark `Tested` in `PROGRESS.md` only after the real KVM pass.

### 5. Pin Or Explain Floating Versions

Problem:

Some labs use `latest` Docker images or download “latest” tools. This is acceptable for learning in places, but bad when the lab is about reproducibility.

Fix:

- Pin versions in labs about production, releases, reliability, and security.
- If `latest` is intentionally used, explain that it is for lab convenience and not production.

Priority files:

- Lab 12: Trivy install uses `latest`
- Lab 17/18/20: Prometheus/Grafana/Loki images
- Lab 19: Jaeger image
- Lab 26/29: platform and capstone images

### 6. Add Hardware And Time Expectations

Problem:

Later labs need more memory and CPU. Learners need to know this before starting.

Fix:

Add per-lab metadata:

```markdown
## Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 60-90 minutes |
| VMs | 2 |
| Minimum RAM | 3 GB for VMs |
| Requires internet | Yes |
| Requires GitHub account | No |
```

### 7. Separate Host Commands, VM Commands, And Container Commands

Problem:

Many DevOps failures come from running the right command in the wrong place.

Fix:

Use labels before every command block:

```markdown
On your laptop:
```

```markdown
Inside the `web` VM:
```

```markdown
Inside the container:
```

This is especially important for labs with `localhost`.

## Yeast v1.1 Accuracy Rules

Every lab must obey:

- Do not use unsupported `yeast.yaml` fields such as `ports`, `host_port`, or `guest_port`.
- Do not say Yeast exposes arbitrary service ports to the host.
- Do not invent Yeast commands.
- Do not use `yeast ssh -- <ssh args>` because v1.1 only supports `--verbose`.
- Use `ssh -N -L ...` for browser tunnels.
- Use `yeast status --json` for automation examples.
- Use `yeast exec <instance> -- <command>` for one-shot commands.
- Use `yeast copy <instance> --to-guest ...` and `yeast copy <instance> --from-guest ...` for file transfer.

## Execution Order

### Phase 1: Course Guardrails

- Add `SUPPORTED_YEAST_V1_1.md`.
- Add `ACCESS.md`.
- Update `README.md`.
- Update `CURRICULUM.md`.
- Update `PROGRESS.md`.

### Phase 2: Fix Service Access In Existing Labs

- Patch localhost/browser instructions in labs 02, 03, 05, 07, 11, 15, 17, 18, 19, 20, 25, 26, 29, and 30.
- Make each service access instruction say whether it runs on the laptop or inside a VM.
- Add tunnel snippets where browser access is needed.

### Phase 3: Yeast Command Accuracy Pass

- Search all labs for `yeast`.
- Compare every command against `SUPPORTED_YEAST_V1_1.md`.
- Replace invented behavior with supported commands.
- Add `yeast status`, `yeast inspect`, and `yeast logs` where they teach real Yeast usage.

### Phase 4: Runtime Test Pass

Run labs in order on a real KVM host:

1. Labs 01-05: Linux/services/troubleshooting foundation.
2. Labs 06-09: automation/secrets.
3. Labs 10-16: containers and delivery.
4. Labs 17-22: observability/reliability.
5. Labs 23-26: IaC/platform.
6. Labs 27-30: Kubernetes/AI.

After each lab:

- Fix broken commands immediately.
- Update `PROGRESS.md`.
- Add troubleshooting notes discovered during the run.

## Definition Of Done

A lab is done only when:

- Its Yeast config uses only supported v1.1 fields.
- Every Yeast command exists in v1.1.
- Every `localhost` command says where it runs.
- Browser access uses an SSH tunnel when run from the laptop.
- `bash assets/validate.sh` passes after following the lab.
- `yeast destroy` cleans up successfully.
- The lab is marked tested in `PROGRESS.md`.
