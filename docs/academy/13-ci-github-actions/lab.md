# Lab 13 — CI With GitHub Actions

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2219 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- A GitHub account and `gh` authentication when the lab uses GitHub

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether the lab asked you to open an SSH tunnel first.

### Expected Checkpoints

- After `yeast up`, `yeast status` should show the expected VM or VMs as running.
- After the main setup steps, the service, tool, or workflow introduced by the lab should respond to the verification commands.
- After `bash assets/validate.sh`, the script should report all checks passed.
- After `yeast destroy`, the lab should be cleaned up before you start the next one.

### Common Mistakes To Avoid

- Running a VM command on your laptop, or a laptop command inside the VM.
- Closing an SSH tunnel and then wondering why `localhost:<port>` stopped working.
- Skipping validation because the final page or command "looked fine".
- Forgetting to run `yeast destroy` before moving to the next lab.

---

## The Story

So far everything you have built has been local: scripts run from your terminal, Ansible run from your laptop, Docker images built on a VM. This works for one person. The moment you have a team, you need automation that runs on every code change — automatically, consistently, without anyone having to remember to do it.

That is CI: Continuous Integration. Every time someone pushes code, a pipeline runs: tests, linting, building, scanning. If it passes, the code is good. If it fails, the developer knows immediately.

GitHub Actions is the CI system built into GitHub. You define pipelines in YAML files that live in your repository. GitHub runs them on every push, pull request, or on a schedule. No separate CI server to maintain.

This lab teaches you to write a real GitHub Actions workflow from scratch, understand every line of it, and connect it to the container work you did in Labs 10–12.

---

## Before You Start — Understanding The Concepts

### What Is Continuous Integration?

Continuous Integration is the practice of merging code changes frequently and automatically verifying each merge with a build and test run.

The problem it solves: in teams that do not practice CI, developers work in isolation for days or weeks. When they merge, the integration is painful — everyone's changes conflict, tests fail in unexpected ways, bugs surface that nobody can trace to a specific commit.

With CI, every commit is integrated immediately and automatically verified. Problems surface within minutes, while the context is fresh.

### What Is A Pipeline?

A pipeline is a series of automated steps that code goes through after a change. A typical pipeline:

1. **Checkout** — pull the code from the repo
2. **Lint** — check for style and syntax errors
3. **Test** — run unit and integration tests
4. **Build** — compile the binary or build the container image
5. **Scan** — check for security vulnerabilities
6. **Deploy** — push to staging or production (CD: Continuous Deployment)

Each step runs only if the previous one passed. A failure in "Test" stops the pipeline before "Build."

### What Is GitHub Actions?

GitHub Actions is GitHub's built-in CI/CD system. Workflows are YAML files in `.github/workflows/`. When a trigger fires (a push, a PR, a schedule), GitHub spins up a runner — a cloud VM — and executes the workflow.

Runners come in different flavors:
- `ubuntu-latest` — Ubuntu Linux (most common)
- `windows-latest` — Windows
- `macos-latest` — macOS

You can also run self-hosted runners — your own VMs that execute workflows (Lab 14).

### What Is A Workflow?

A GitHub Actions workflow is a YAML file defining:
- **Triggers** (`on:`) — what events cause the workflow to run
- **Jobs** — groups of steps that run on the same runner
- **Steps** — individual commands or actions within a job

```yaml
on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "Hello from CI"
```

### What Is An Action?

An action is a reusable piece of workflow logic, published to the GitHub Marketplace. `actions/checkout@v4` checks out your repository code. `docker/build-push-action@v5` builds and pushes a Docker image.

You reference actions with `uses:`. You pass inputs with `with:`. Actions hide complexity — `docker/build-push-action` handles layer caching, multi-platform builds, registry login — things that would take 50 lines of shell script.

### What Are Secrets In GitHub Actions?

GitHub repositories have an encrypted secrets store. You add secrets in Settings → Secrets → Actions. Workflows reference them as `${{ secrets.MY_SECRET }}`.

Secrets are:
- Encrypted at rest
- Never printed in logs (GitHub masks them)
- Only accessible to workflows in the same repository

You use secrets for: Docker Hub credentials, cloud API keys, deployment tokens.

### What Is A Job Matrix?

A matrix lets you run the same job with different parameters. Test against Python 3.10, 3.11, and 3.12 simultaneously without writing three separate jobs:

```yaml
strategy:
  matrix:
    python-version: ["3.10", "3.11", "3.12"]
```

GitHub runs all matrix combinations in parallel.

---

## What You Are Building

A GitHub repository with a CI workflow that:
1. Runs on every push and pull request
2. Lints the code
3. Runs tests
4. Builds the Docker image from Lab 12
5. Scans it with Trivy
6. Reports pass/fail status on the commit

This does not require the Yeast VM — it is entirely on GitHub. The VM is just for tooling (gh CLI).

---

## Prerequisites

- A GitHub account
- The `gh` CLI installed (the VM provisions it, or install on your laptop)

---

## Part 1 — Create The Repository

Create a new GitHub repository for this lab:

```bash
gh auth login
# Follow prompts to authenticate with GitHub

gh repo create devops-bootcamp-lab13 --public --clone
cd devops-bootcamp-lab13
```

Or create it on github.com manually and clone it.

---

## Part 2 — The Application Code

Create a small Python app with a test:

```bash
mkdir -p src tests

cat > src/app.py << 'EOF'
#!/usr/bin/env python3
import json
import os
from http.server import HTTPServer, BaseHTTPRequestHandler

VERSION = os.getenv("APP_VERSION", "dev")

def create_handler():
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path == "/healthz":
                body = json.dumps({"status": "ok", "version": VERSION}).encode()
                self.send_response(200)
            elif self.path == "/":
                body = json.dumps({"message": "Hello", "version": VERSION}).encode()
                self.send_response(200)
            else:
                body = json.dumps({"error": "not found"}).encode()
                self.send_response(404)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *a):
            pass

    return Handler

if __name__ == "__main__":
    HTTPServer(("0.0.0.0", 8000), create_handler()).serve_forever()
EOF

cat > tests/test_app.py << 'EOF'
import sys
import os
import json
import unittest
from unittest.mock import patch, MagicMock
from io import BytesIO

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src'))
from app import create_handler

class TestApp(unittest.TestCase):
    def _make_request(self, path):
        handler_class = create_handler()
        mock_request = MagicMock()
        mock_request.makefile.return_value = BytesIO(b"")
        handler = handler_class.__new__(handler_class)
        handler.path = path
        handler.headers = {}
        responses = []

        def send_response(code):
            responses.append(code)
        def send_header(*a): pass
        def end_headers(): pass

        buf = BytesIO()
        handler.wfile = buf
        handler.send_response = send_response
        handler.send_header = send_header
        handler.end_headers = end_headers
        handler.do_GET()
        return responses[0], buf.getvalue()

    def test_healthz_returns_200(self):
        code, _ = self._make_request("/healthz")
        self.assertEqual(code, 200)

    def test_root_returns_200(self):
        code, body = self._make_request("/")
        self.assertEqual(code, 200)
        data = json.loads(body)
        self.assertIn("message", data)

    def test_unknown_path_returns_404(self):
        code, _ = self._make_request("/doesnotexist")
        self.assertEqual(code, 404)

if __name__ == "__main__":
    unittest.main()
EOF
```

The Dockerfile (hardened version from Lab 12):

```bash
cat > Dockerfile << 'EOF'
FROM python:3.11-slim

RUN groupadd --gid 1000 appuser && \
    useradd --uid 1000 --gid 1000 --no-create-home appuser

WORKDIR /app
COPY --chown=appuser:appuser src/ .
USER appuser
EXPOSE 8000
CMD ["python3", "app.py"]
EOF
```

Test it works locally before setting up CI:

```bash
python3 -m pytest tests/ -v
```

---

## Part 3 — The GitHub Actions Workflow

Create the workflow directory and file:

```bash
mkdir -p .github/workflows

cat > .github/workflows/ci.yml << 'EOF'
name: CI

on:
  push:
    branches: ["main", "feature/**"]
  pull_request:
    branches: ["main"]

jobs:

  test:
    name: Test
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: "3.11"

      - name: Run tests
        run: python3 -m pytest tests/ -v

  build-and-scan:
    name: Build and Scan
    runs-on: ubuntu-latest
    needs: test

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: false
          tags: myapp:${{ github.sha }}
          load: true
          cache-from: type=gha
          cache-to: type=gha,mode=max

      - name: Scan with Trivy
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: myapp:${{ github.sha }}
          format: table
          exit-code: "0"
          severity: CRITICAL,HIGH
          ignore-unfixed: true
EOF
```

---

## Reading The Workflow — Every Line Explained

### `on:` — triggers

```yaml
on:
  push:
    branches: ["main", "feature/**"]
  pull_request:
    branches: ["main"]
```

This workflow runs on:
- Any push to `main` or any branch matching `feature/**` (e.g., `feature/auth`, `feature/new-api`)
- Any pull request targeting `main`

`**` is a glob pattern: "any characters including `/`".

### `jobs:` — units of work

Each job runs on its own runner (a fresh VM spun up by GitHub). Jobs run in parallel by default.

### `needs: test`

```yaml
build-and-scan:
  needs: test
```

The `build-and-scan` job only runs if the `test` job passes. If tests fail, there is no point building the image. This creates a dependency graph.

### `uses: actions/checkout@v4`

Checks out your repository code onto the runner. Without this, the runner has an empty workspace. `@v4` pins the action to version 4. Always pin actions to a version — `@master` would change behavior when the action authors push updates.

### `uses: actions/setup-python@v5`

Installs Python on the runner with the specified version. Runners have Python pre-installed, but this action lets you pin the exact version and handles caching.

### `uses: docker/setup-buildx-action@v3`

Sets up Docker Buildx — an extended Docker builder with support for multi-platform builds and build caching. Required for the `docker/build-push-action`.

### `uses: docker/build-push-action@v5`

Builds the Docker image. Key inputs:
- `context: .` — the build context is the current directory
- `push: false` — build but do not push to a registry (no credentials configured yet)
- `tags: myapp:${{ github.sha }}` — tag with the exact git commit hash. `${{ github.sha }}` is a built-in variable containing the full SHA.
- `load: true` — load the built image into the local Docker daemon so Trivy can scan it
- `cache-from/cache-to: type=gha` — use GitHub Actions cache for Docker layer caching. This makes rebuilds fast — unchanged layers are cached between runs.

### `uses: aquasecurity/trivy-action@master`

Runs Trivy to scan the image.
- `exit-code: "0"` — do not fail the build even if vulnerabilities are found (for now — in production you set this to `"1"` to fail on CRITICAL findings)
- `ignore-unfixed: true` — only report vulnerabilities that have a fix available

### `${{ github.sha }}`

GitHub Actions expressions use `${{ }}` syntax. Built-in variables:
- `github.sha` — the full git commit SHA
- `github.ref` — the branch or tag ref
- `github.actor` — who triggered the run
- `github.run_number` — sequential run counter
- `secrets.MY_SECRET` — encrypted secret value

---

## Part 4 — Push And Watch It Run

```bash
git add .
git commit -m "feat: initial app with CI workflow"
git push origin main
```

Open your browser and go to:
```
https://github.com/<your-username>/devops-bootcamp-lab13/actions
```

You will see the workflow running. Click on it to see the live log. Watch each step execute. When it finishes, you get a green checkmark (all passed) or red X (something failed).

Check the status from the terminal:

```bash
gh run list --limit 5
gh run view  # shows the most recent run
```

---

## Part 5 — Making It Fail On Purpose

Break a test:

```bash
# Edit tests/test_app.py — change the assertion to expect the wrong status code
sed -i 's/self.assertEqual(code, 200)/self.assertEqual(code, 999)/' tests/test_app.py

git add tests/test_app.py
git commit -m "test: intentional failure"
git push origin main
```

Watch the Actions tab. The `test` job fails. The `build-and-scan` job does not run — `needs: test` prevents it.

The commit on GitHub shows a red X next to it. This is what CI should do: tell you immediately when something is broken.

Fix it:

```bash
sed -i 's/self.assertEqual(code, 999)/self.assertEqual(code, 200)/' tests/test_app.py
git add tests/test_app.py
git commit -m "fix: restore correct test assertion"
git push origin main
```

Green again.

---

## Part 6 — Adding A Matrix

Test against multiple Python versions simultaneously:

```yaml
jobs:
  test:
    name: Test (Python ${{ matrix.python-version }})
    runs-on: ubuntu-latest
    strategy:
      matrix:
        python-version: ["3.10", "3.11", "3.12"]

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: ${{ matrix.python-version }}
      - run: python3 -m pytest tests/ -v
```

Edit `.github/workflows/ci.yml` to add the matrix. Push. GitHub now runs three test jobs in parallel, one per Python version. If the code breaks on 3.10 but not 3.11, you see it immediately.

---

## Part 7 — Branch Protection

CI only has value if you enforce it. Go to your GitHub repo:

1. Settings → Branches → Add rule
2. Branch name pattern: `main`
3. Check: "Require status checks to pass before merging"
4. Add your workflow's job names as required checks
5. Check: "Require branches to be up to date before merging"

Now no pull request can merge to `main` until CI passes. This is the enforcement layer that makes CI actually protect your codebase.

---

## Validate Your Work

```bash
bash assets/validate.sh
```

Note: this script checks your local `.github/workflows/ci.yml` and `gh` authentication. Most of the work is verified by actually watching your workflow run on GitHub.

---

## Clean Up

```bash
yeast destroy
# The GitHub repo and workflow persist independently of the VM
```

---

## Quick Recap

In Lab 13 — CI With GitHub Actions, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What CI is and the problem it solves: automatic verification on every commit
- GitHub Actions structure: `on:`, `jobs:`, `steps:`, `uses:`, `run:`
- Triggers: push, pull_request, branch patterns
- `needs:` for job dependency chains: stop before building if tests fail
- Actions from the marketplace: `checkout`, `setup-python`, `build-push-action`, `trivy-action`
- `${{ github.sha }}` and other built-in expressions
- Layer caching in CI: `cache-from: type=gha` for fast Docker builds
- Matrix strategy: parallel testing across multiple versions
- Making CI fail on purpose — and seeing how it blocks bad code
- Branch protection: enforcing CI as a merge gate

---

## What Is Next

**Lab 14 — Self-Hosted CI Runner**

GitHub-hosted runners work. But they have limitations: they are ephemeral, they have no access to private infrastructure, they cost credits for large builds. Lab 14 teaches you to register your own Yeast VM as a GitHub Actions runner, so your CI jobs execute on infrastructure you control.
