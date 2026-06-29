# Lab 14 — Self-Hosted CI Runner

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 1 |
| Minimum VM RAM | 2048 MB |
| SSH ports | 2220 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic understanding that Docker commands run inside the VM unless stated otherwise
- A GitHub account and `gh` authentication when the lab uses GitHub

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you. If not, the lab will tell you when to use a manual SSH tunnel.
- Run Docker commands inside the VM unless the lab explicitly says otherwise.

### Expected Checkpoints

- After `yeast up`, `yeast status` should show the expected VM or VMs as running.
- After the main setup steps, the service, tool, or workflow introduced by the lab should respond to the verification commands.
- After `bash assets/validate.sh`, the script should report all checks passed.
- After `yeast destroy`, the lab should be cleaned up before you start the next one.

### Common Mistakes To Avoid

- Running a VM command on your laptop, or a laptop command inside the VM.
- Ignoring the forwarded port shown by `yeast up` or `yeast status`, or opening a tunnel when the lab already gave you a forwarded host port.
- Skipping validation because the final page or command "looked fine".
- Forgetting to run `yeast destroy` before moving to the next lab.
- Confusing laptop `localhost`, VM `localhost`, and container `localhost`.

---

## The Story

GitHub-hosted runners are convenient — GitHub provisions a clean VM, runs your workflow, and destroys it. But they have real limitations. They cannot reach your private network. They have a fixed set of tools and a fixed amount of CPU and RAM. Large builds are slow. And if your organization has compliance requirements, you may not be allowed to run code on GitHub's infrastructure.

Self-hosted runners are the solution. You register your own VM with GitHub. When a workflow runs with `runs-on: self-hosted`, GitHub sends the job to your runner instead of its own cloud infrastructure. Your runner builds on your hardware, on your network, with tools you installed.

In this lab you will turn a Yeast VM into a registered GitHub Actions runner, run a workflow on it, and understand how the runner architecture works.

---

## Before You Start — Understanding The Concepts

### What Is A Runner?

A runner is a machine that listens for workflow jobs from GitHub and executes them. It is a long-running process that polls GitHub's API, picks up queued jobs, executes them in a workspace directory, streams logs back to GitHub, and reports the result.

Runners come in two types:

**GitHub-hosted** — provisioned on demand in GitHub's cloud. Fresh environment for every job. Limited to GitHub's available hardware. You pay per minute on private repos.

**Self-hosted** — your own machine, registered with GitHub. Persistent — the runner stays running between jobs. Has access to whatever tools and network you installed. You pay nothing to GitHub (only your own infrastructure cost).

### How Does The Runner Work?

The GitHub Actions runner is an open-source application (written in .NET) that you download from GitHub and run on your machine. It:
1. Registers itself with GitHub using a token
2. Polls GitHub's API for queued jobs
3. When a job arrives, downloads the workflow steps
4. Executes each step in a workspace directory
5. Streams logs back to GitHub in real-time
6. Reports success or failure

The runner needs outbound HTTPS access to GitHub (port 443). It does not need any inbound ports — it polls outbound.

### What Are Runner Labels?

Runners have labels that workflows use to target them. The default labels are `self-hosted`, the OS (`linux`, `windows`, `macos`), and the architecture (`x64`, `arm64`).

You can add custom labels: `docker`, `gpu`, `large`, `staging`. In the workflow, `runs-on: [self-hosted, docker]` targets runners with both labels.

### Runner Security Considerations

Self-hosted runners execute arbitrary code from your repository. For **public repositories**, any fork can trigger a workflow that runs on your runner — giving that code access to your private network. For this reason:

- Only use self-hosted runners with private repositories, or public repos where you carefully control who can trigger workflows
- Do not run runners as root
- Consider ephemeral runners (one job, then destroy the runner VM) for better isolation

For this lab we use a private repository — the Lab 13 repository you created.

### What Is A Workspace?

The runner creates a fresh workspace directory for each job. It checks out the repository code there, runs all the steps, and then (optionally) cleans up. The workspace is at `~/actions-runner/_work/<repo>/<repo>` by default.

---

## What You Are Building

```
Your Laptop
    │
    │  SSH port 2220
    ▼
┌──────────────────────────────────────────────────┐
│  runner VM (Ubuntu 22.04)                        │
│                                                  │
│  GitHub Actions runner process                   │
│    ↕ polls https://api.github.com                │
│                                                  │
│  Docker installed (for container builds)         │
└──────────────────────────────────────────────────┘
        ↕
  GitHub.com — sends workflow jobs to this runner
```

---

## Starting The Lab

```bash
cd 14-self-hosted-ci-runner
yeast up
yeast ssh runner
newgrp docker
```

---

## Step 1 — Download The Runner

GitHub provides a download link specific to your architecture. Get the latest version:

```bash
# Create the runner directory
mkdir -p /home/ubuntu/actions-runner && cd /home/ubuntu/actions-runner

# Download the latest runner (amd64 Linux)
RUNNER_VERSION=$(curl -s https://api.github.com/repos/actions/runner/releases/latest | \
  grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')

echo "Downloading runner version: $RUNNER_VERSION"

curl -o actions-runner-linux-x64.tar.gz -L \
  "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz"

tar xzf actions-runner-linux-x64.tar.gz
ls -la
```

You should see `run.sh`, `config.sh`, and a `bin/` directory.

---

## Step 2 — Get A Registration Token

You need a token from GitHub to register the runner. Do this from your laptop (or inside the VM if `gh` is authenticated):

```bash
# From your laptop
gh api repos/<your-username>/devops-bootcamp-lab13/actions/runners/registration-token \
  --method POST \
  -q .token
```

This returns a short-lived token (valid for 1 hour) that you use in the next step.

Alternatively, get the token from GitHub's web UI:
1. Go to your repo on GitHub
2. Settings → Actions → Runners → New self-hosted runner
3. GitHub shows the token in the configuration commands

---

## Step 3 — Configure The Runner

Back inside the VM, run the configuration script with the token:

```bash
cd /home/ubuntu/actions-runner

./config.sh \
  --url https://github.com/<your-username>/devops-bootcamp-lab13 \
  --token <TOKEN-FROM-STEP-2> \
  --name "yeast-runner-01" \
  --labels "self-hosted,linux,docker,yeast" \
  --work "_work" \
  --unattended
```

Flags:
- `--name` — the runner's display name in GitHub's UI
- `--labels` — custom labels for targeting this runner in workflows
- `--unattended` — skip interactive prompts
- `--work` — the directory where jobs run

After this runs, your runner is registered but not yet started.

Verify it registered:

```bash
# From your laptop
gh api repos/<your-username>/devops-bootcamp-lab13/actions/runners
```

You should see your runner listed with `status: offline` (because it is not running yet).

---

## Step 4 — Install As A Systemd Service

You want the runner to start automatically and run in the background:

```bash
cd /home/ubuntu/actions-runner

# Install as a systemd service (requires sudo)
sudo ./svc.sh install ubuntu
sudo ./svc.sh start

# Check it is running
sudo ./svc.sh status
```

The `svc.sh` script creates a systemd service unit and starts it. The service name will be something like `actions.runner.<org>.<repo>.<name>.service`.

Check the runner is online:

```bash
sudo systemctl status "actions.runner.*"
```

Check from GitHub:

```bash
gh api repos/<your-username>/devops-bootcamp-lab13/actions/runners
```

The status should now be `online`. Your runner is listening for jobs.

---

## Step 5 — Update The Workflow To Use Your Runner

From your laptop, edit the workflow in your Lab 13 repository:

```bash
cd devops-bootcamp-lab13

# Edit .github/workflows/ci.yml
# Change runs-on: ubuntu-latest to runs-on: [self-hosted, yeast]
sed -i 's/runs-on: ubuntu-latest/runs-on: [self-hosted, yeast]/g' .github/workflows/ci.yml

git add .github/workflows/ci.yml
git commit -m "ci: switch to self-hosted runner"
git push origin main
```

---

## Step 6 — Watch The Job Run On Your Runner

Check the Actions tab on GitHub. The new workflow run should be picked up by your runner.

Watch the runner logs from inside the VM:

```bash
yeast ssh runner
sudo journalctl -u "actions.runner.*" -f
```

You will see the runner:
1. Receive the job
2. Check out the repository
3. Execute each step
4. Stream output back to GitHub
5. Report success

Check the workflow result on GitHub — it should show it ran on your self-hosted runner, identifiable by the label.

---

## Step 7 — Runner With Docker Access

Your runner has Docker installed and the `ubuntu` user is in the `docker` group. This means workflow steps can use Docker:

Add a step to your workflow to confirm Docker is available on the self-hosted runner:

```yaml
- name: Verify Docker on self-hosted runner
  run: |
    docker --version
    docker run --rm hello-world
```

Push the change and watch it run. The `hello-world` container runs on your Yeast VM — on your hardware, not GitHub's.

This is the key capability: Docker builds, large test suites, access to private network services — all on infrastructure you control.

---

## Step 8 — Runner Monitoring

Check what jobs have run on your runner:

```bash
ls /home/ubuntu/actions-runner/_work/
ls /home/ubuntu/actions-runner/_diag/
```

`_work/` contains the checked-out repository for the last job. `_diag/` contains diagnostic logs for the runner itself — useful when a runner misbehaves.

Runner logs:

```bash
sudo journalctl -u "actions.runner.*" --no-pager -n 50
```

---

## Step 9 — Unregistering The Runner

When you are done with this lab:

```bash
yeast ssh runner
cd /home/ubuntu/actions-runner

sudo ./svc.sh stop
sudo ./svc.sh uninstall

# Get a remove token (different from registration token)
REMOVE_TOKEN=$(curl -s -X POST \
  -H "Authorization: token $(cat ~/.github_token)" \
  https://api.github.com/repos/<your-username>/devops-bootcamp-lab13/actions/runners/remove-token \
  | jq -r .token)

./config.sh remove --token "$REMOVE_TOKEN"
```

Or from your laptop:

```bash
gh api repos/<your-username>/devops-bootcamp-lab13/actions/runners \
  --jq '.[].id'
# Get the runner ID, then:
gh api repos/<your-username>/devops-bootcamp-lab13/actions/runners/<ID> \
  --method DELETE
```

After removal, the runner no longer appears in GitHub's runners list.

---

## Ephemeral Runners: The Better Pattern

The runner in this lab is persistent — it stays running between jobs. A persistent runner accumulates state: leftover files from previous jobs, cached credentials, installed packages. This can cause jobs to behave differently depending on what ran before — violating the "clean environment" assumption.

The better pattern for production is **ephemeral runners**:
1. A new VM (or container) is created for each job
2. The runner registers, runs the job, deregisters
3. The VM is destroyed

This is implemented with tools like `actions-runner-controller` (for Kubernetes) or custom scripts that provision Yeast VMs on demand. We revisit this concept in the Kubernetes labs.

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
# Unregister the runner first (see Step 9)
# Then:
yeast destroy
```

---

## Quick Recap

In Lab 14 — Self-Hosted CI Runner, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What a self-hosted runner is: a process that polls GitHub and executes jobs on your infrastructure
- How runner registration works: token-based, short-lived
- Runner labels: how workflows target specific runners with `runs-on:`
- Installing the runner as a systemd service: persistent, auto-restart
- `svc.sh`: the runner's own service management script
- Runner workspace and diagnostic directories
- Why Docker access on a self-hosted runner is more powerful than GitHub-hosted
- The security consideration: only use self-hosted runners with private repos
- The ephemeral runner pattern: why persistent runners accumulate state problems

---

## What Is Next

**Lab 15 — Private Container Registry**

Your CI pipeline builds container images. Right now they stay on the build machine. In Lab 15 you run a private Docker registry on a Yeast VM, push images to it from CI, and deploy them to another VM by pulling from the registry. This is the standard image promotion flow used in production.
