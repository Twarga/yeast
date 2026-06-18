# Lab 30 — AI-Assisted DevOps And Local LLM Ops

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate to advanced |
| Estimated time | 75-120 minutes |
| VMs | 1 |
| Minimum VM RAM | 4096 MB |
| SSH ports | 2252 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic shell scripting comfort
- Comfort creating SSH tunnels from `ACCESS.md` for browser-based tools
- Comfort treating AI output as a draft that must be verified

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
- Letting AI output sound convincing without checking logs, metrics, or commands yourself.

---

## The Story

You have built a complete DevOps platform across 29 labs. You know how to operate Linux servers, deploy applications, manage containers, run CI/CD pipelines, monitor systems, and recover from failures.

Now for the final skill: using AI as a tool inside that work — not to replace your judgment, but to accelerate specific tasks where language models genuinely help.

The scenario: an incident happens at 2 AM. You have 500 lines of logs, metrics that are spiking, and a customer report that says "checkout is broken." You need to form a hypothesis fast.

A local LLM can read the logs and suggest probable causes in seconds. It can draft a runbook skeleton. It can explain an unfamiliar error message. It does this on your machine, with your data, without sending anything to a cloud service.

This lab teaches you to run a local LLM with Ollama, wire it into operational scripts, and use it safely — with human review before any action is taken.

---

## Before You Start — Understanding The Concepts

### What Is A Large Language Model (LLM)?

A large language model is a neural network trained on vast amounts of text. It generates responses by predicting the most likely next token given its input. Modern LLMs are remarkably capable at:
- Summarizing and explaining text
- Generating code and scripts
- Answering questions about technical topics
- Pattern matching in unstructured text (like logs)

They are also unreliable in specific ways:
- They can "hallucinate" — confidently state false things
- They do not know your specific infrastructure
- They cannot execute commands or take actions (without tool use)
- Their knowledge has a cutoff date

The right mental model: an LLM is a knowledgeable colleague who has read a lot but who you must verify before trusting.

### What Is Ollama?

Ollama is a tool for running LLMs locally. It downloads model weights, manages GPU/CPU resources, and exposes an HTTP API compatible with OpenAI's API format. Key features:

- Runs on CPU (slower) or GPU (faster)
- Supports many open models: Llama 3, Mistral, Gemma, Phi, Qwen
- API at `http://localhost:11434` from inside the VM
- `ollama run <model>` for interactive chat
- `ollama pull <model>` to download a model

### What Models Are Available?

For operations work on a laptop or small VM, small models are practical:

| Model | Size | Good For |
|---|---|---|
| `llama3.2:1b` | ~0.8 GB | Fast, fits anywhere, basic tasks |
| `llama3.2:3b` | ~2 GB | Better reasoning, still fast |
| `phi3:mini` | ~2 GB | Good at code and analysis |
| `mistral:7b` | ~4 GB | Strong general performance |
| `llama3.1:8b` | ~5 GB | Best quality for 8B class |

For this lab we use `llama3.2:3b` — small enough to run on the Yeast VM's 4 GB RAM, with sufficient capability for log analysis.

### What Is The Ollama API?

Ollama exposes an HTTP API:

```bash
# Generate a completion
curl http://localhost:11434/api/generate \
  -d '{"model": "llama3.2:3b", "prompt": "Explain what SIGKILL means", "stream": false}'

# Chat format (OpenAI-compatible)
curl http://localhost:11434/api/chat \
  -d '{
    "model": "llama3.2:3b",
    "messages": [{"role": "user", "content": "What does OOMKilled mean in Kubernetes?"}]
  }'
```

### What Are The Safety Rules For AI In Operations?

1. **AI suggests, humans approve.** Never let AI apply infrastructure changes without a human reviewing and confirming.
2. **Verify every factual claim.** LLMs hallucinate. If the model says "restart the service with `systemctl restart foo`", verify `foo` is the real service name.
3. **Do not send secrets to cloud AI.** Log lines often contain IP addresses, usernames, and sometimes credentials. Use local models for anything sensitive.
4. **Treat AI output as a starting point.** A suggested runbook is a draft, not a finished document.
5. **Build the skill yourself first.** AI assistance is most useful to someone who already understands the domain. Using it as a crutch before you understand the fundamentals produces confident-sounding wrong answers.

---

## What You Are Building

```
Your Laptop
    │  SSH  2252 → llm VM port 22
    │  HTTP 11434 → llm VM port 11434 (Ollama API)
    ▼
┌─────────────────────────────────────────────────────────┐
│  llm VM (Ubuntu 22.04, 4 CPU, 4 GB RAM, 50 GB disk)     │
│                                                         │
│  Ollama :11434                                          │
│    └── llama3.2:3b (downloaded, loaded on demand)       │
│                                                         │
│  ops-assist.sh  (log analysis + incident hypothesis)    │
│  runbook-draft.sh (AI-drafted runbook from description) │
└─────────────────────────────────────────────────────────┘
```

---

## Starting The Lab

```bash
cd 30-ai-assisted-devops-llm-ops
yeast up
yeast ssh llm
```

Verify Ollama is running:

```bash
sudo systemctl is-active ollama
curl http://localhost:11434/api/tags
```

---

## Step 1 — Pull A Model

```bash
ollama pull llama3.2:3b
```

This downloads the model weights (~2 GB). It takes a few minutes on first run. After download it is cached locally.

List available models:

```bash
ollama list
```

Test it interactively:

```bash
ollama run llama3.2:3b "In one sentence, what is a kernel OOM killer?"
```

---

## Step 2 — Understanding The API

```bash
# Basic generation
curl -s http://localhost:11434/api/generate \
  -H "Content-Type: application/json" \
  -d '{"model": "llama3.2:3b", "prompt": "What does errno 13 mean on Linux?", "stream": false}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['response'])"
```

The key fields in the response:
- `response` — the generated text
- `done` — true when generation is complete
- `eval_count` — tokens generated
- `eval_duration` — time taken in nanoseconds

For operational scripts, `stream: false` waits for the full response before returning — simpler to work with than streaming.

---

## Step 3 — Log Analysis Script

This is the main operational use case: feeding logs to the model and asking for a hypothesis.

```bash
cat > /home/ubuntu/ops-assist.sh << 'SCRIPT'
#!/usr/bin/env bash
# ops-assist.sh — Feed logs to a local LLM for incident hypothesis
# Usage: bash ops-assist.sh <log-file>
# Or:    bash ops-assist.sh --stdin  (read from stdin)

set -euo pipefail

MODEL="llama3.2:3b"
OLLAMA_URL="http://localhost:11434/api/generate"
MAX_LINES=100  # limit log lines to avoid exceeding context

if [ "${1:-}" = "--stdin" ]; then
    LOG_CONTENT=$(head -n "$MAX_LINES")
elif [ -n "${1:-}" ] && [ -f "$1" ]; then
    LOG_CONTENT=$(tail -n "$MAX_LINES" "$1")
else
    echo "Usage: $0 <log-file> OR $0 --stdin < logfile"
    exit 1
fi

LINE_COUNT=$(echo "$LOG_CONTENT" | wc -l)
echo "=== Analyzing $LINE_COUNT log lines with ${MODEL} ==="
echo ""

PROMPT="You are an experienced Linux/DevOps engineer analyzing logs to diagnose an incident.

Here are the log lines:

---
${LOG_CONTENT}
---

Please analyze these logs and provide:
1. PROBABLE CAUSE: What is the most likely root cause of any errors or issues?
2. AFFECTED COMPONENTS: Which services or systems appear affected?
3. IMMEDIATE ACTIONS: What are the first 2-3 things to investigate or try?
4. CONFIDENCE: How confident are you in this analysis (low/medium/high) and why?

Be specific and reference actual log lines where relevant. If the logs look healthy, say so."

# Call the Ollama API
RESPONSE=$(curl -s "$OLLAMA_URL" \
    -H "Content-Type: application/json" \
    -d "$(python3 -c "
import json, sys
data = {'model': '$MODEL', 'prompt': sys.stdin.read(), 'stream': False}
print(json.dumps(data))
" <<< "$PROMPT")")

# Extract and print the response
echo "$RESPONSE" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(data.get('response', 'No response from model'))
    print()
    tokens = data.get('eval_count', 0)
    ms = data.get('eval_duration', 0) // 1_000_000
    print(f'--- ({tokens} tokens, {ms}ms) ---')
except Exception as e:
    print(f'Error parsing response: {e}')
    sys.exit(1)
"

echo ""
echo "=== AI analysis complete. VERIFY all suggestions before acting. ==="
SCRIPT

chmod +x /home/ubuntu/ops-assist.sh
```

Test it with a synthetic log file:

```bash
cat > /tmp/test.log << 'EOF'
2026-06-15 02:14:22 INFO  [nginx] upstream: 192.168.1.20:8000 connected
2026-06-15 02:14:23 INFO  [app] request GET /api/orders status=200 latency=45ms
2026-06-15 02:14:24 INFO  [app] request GET /api/orders status=200 latency=51ms
2026-06-15 02:14:25 ERROR [app] psycopg2.OperationalError: could not connect to server: Connection refused
2026-06-15 02:14:25 ERROR [app] request GET /api/orders status=500 latency=5012ms
2026-06-15 02:14:26 ERROR [app] psycopg2.OperationalError: could not connect to server: Connection refused
2026-06-15 02:14:26 WARN  [nginx] upstream error: 192.168.1.20:8000 returned 500
2026-06-15 02:14:27 ERROR [app] psycopg2.OperationalError: could not connect to server: Connection refused
2026-06-15 02:14:27 ERROR [app] FATAL: max connection retries exceeded
2026-06-15 02:14:28 ERROR [nginx] upstream: 192.168.1.20:8000 connect() failed (111: Connection refused)
2026-06-15 02:14:29 CRIT  [nginx] no live upstreams while connecting to upstream
2026-06-15 02:14:30 ERROR [postgres] FATAL: could not write lock file "postmaster.pid": No space left on device
2026-06-15 02:14:30 ERROR [postgres] database system is shut down
EOF

bash /home/ubuntu/ops-assist.sh /tmp/test.log
```

Read the output carefully. The model should identify:
- PostgreSQL cannot write to disk (no space left on device)
- This caused the database to shut down
- Which caused the app to lose its database connection
- Which caused the API to return 500 errors

**Verify the analysis against the actual log lines.** The model got it right here, but it is your job to confirm — not blindly trust.

---

## Step 4 — Runbook Drafting Script

```bash
cat > /home/ubuntu/runbook-draft.sh << 'SCRIPT'
#!/usr/bin/env bash
# runbook-draft.sh — Draft a runbook skeleton for a described failure scenario
# Usage: bash runbook-draft.sh "describe the failure scenario"

set -euo pipefail

MODEL="llama3.2:3b"
OLLAMA_URL="http://localhost:11434/api/generate"
SCENARIO="${*}"

if [ -z "$SCENARIO" ]; then
    echo "Usage: $0 \"describe the failure scenario\""
    exit 1
fi

echo "=== Drafting runbook for: $SCENARIO ==="
echo ""

PROMPT="You are an experienced SRE writing runbooks for a Linux/DevOps team.

Write a runbook for this failure scenario: ${SCENARIO}

Format the runbook as:
## Symptoms
- What the on-call engineer observes

## Diagnosis Steps
1. First thing to check
2. Second thing to check
(specific commands to run)

## Recovery Steps
1. Step one (include the exact command)
2. Step two
(be specific, include actual Linux commands where applicable)

## Verification
- How to confirm the service is healthy again

## Escalation
- When to escalate and to whom

Keep it practical and specific. Use real Linux/DevOps commands."

RESPONSE=$(curl -s "$OLLAMA_URL" \
    -H "Content-Type: application/json" \
    -d "$(python3 -c "
import json, sys
data = {'model': '$MODEL', 'prompt': sys.stdin.read(), 'stream': False}
print(json.dumps(data))
" <<< "$PROMPT")")

echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data.get('response', 'No response'))
"

echo ""
echo "=== DRAFT ONLY. Review and edit before publishing to team. ==="
SCRIPT

chmod +x /home/ubuntu/runbook-draft.sh

# Test it
bash /home/ubuntu/runbook-draft.sh \
  "PostgreSQL runs out of disk space and the application loses database connectivity"
```

---

## Step 5 — Explain An Error Message

Another practical use: you see an unfamiliar error and want a quick explanation before diving into documentation.

```bash
cat > /home/ubuntu/explain-error.sh << 'SCRIPT'
#!/usr/bin/env bash
# explain-error.sh — Ask the LLM to explain an error message
# Usage: bash explain-error.sh "error message here"

set -euo pipefail
MODEL="llama3.2:3b"
ERROR="${*}"

PROMPT="Explain this Linux/DevOps error message concisely:

\"${ERROR}\"

Include:
1. What caused it
2. How to diagnose it
3. How to fix it

Be specific and practical."

curl -s http://localhost:11434/api/generate \
    -H "Content-Type: application/json" \
    -d "$(python3 -c "import json,sys; print(json.dumps({'model':'$MODEL','prompt':sys.stdin.read(),'stream':False}))" <<< "$PROMPT")" \
    | python3 -c "import sys,json; print(json.load(sys.stdin).get('response',''))"
SCRIPT

chmod +x /home/ubuntu/explain-error.sh

# Test with real errors
bash /home/ubuntu/explain-error.sh \
  "OOMKilled: container exceeded memory limit"

bash /home/ubuntu/explain-error.sh \
  "FATAL: database files are incompatible with server"

bash /home/ubuntu/explain-error.sh \
  "failed to set file attributes on '/tmp/testfile': Operation not permitted"
```

---

## Step 6 — The Limits: What AI Gets Wrong

Understanding where AI fails is as important as knowing where it helps.

Run these prompts and evaluate the responses critically:

```bash
# Ask for a specific system detail it cannot know
bash /home/ubuntu/explain-error.sh \
  "connection refused to 192.168.1.20:5432"
```

The model will give generic advice about PostgreSQL connection issues. It does not know your network layout, which VM that IP belongs to, or what your PostgreSQL config looks like.

```bash
# Ask something with a known-wrong common misconception trap
ollama run llama3.2:3b \
  "What is the difference between SIGTERM and SIGKILL and which one gracefully stops a process?"
```

The model should get this right. But notice if it is overly confident or adds caveats. Healthy skepticism about model output is the right posture.

**The rule:** AI output is a starting hypothesis, not a conclusion. You bring domain knowledge, you verify with actual commands, and you make the final call.

---

## Step 7 — A Complete Incident Workflow

Put it all together. Simulate a real incident workflow:

**1. Incident fires.** Something is wrong. You collect logs:

```bash
# Generate a realistic incident log
cat > /tmp/incident.log << 'EOF'
2026-06-15 03:00:01 INFO  Service started
2026-06-15 03:00:05 INFO  Connected to database
2026-06-15 03:15:22 WARN  Memory usage at 78%
2026-06-15 03:16:01 WARN  Memory usage at 85%
2026-06-15 03:16:45 ERROR java.lang.OutOfMemoryError: GC overhead limit exceeded
2026-06-15 03:16:45 ERROR at java.util.Arrays.copyOf(Arrays.java:3210)
2026-06-15 03:16:46 ERROR Failed to allocate memory for request processing
2026-06-15 03:16:47 CRIT  Service shutting down due to memory pressure
2026-06-15 03:16:48 INFO  Attempting emergency cache flush
2026-06-15 03:16:48 ERROR Cache flush failed: no memory available
2026-06-15 03:16:49 CRIT  Service terminated. Exit code 137
EOF
```

**2. Get AI hypothesis:**

```bash
bash /home/ubuntu/ops-assist.sh /tmp/incident.log
```

**3. Draft a runbook:**

```bash
bash /home/ubuntu/runbook-draft.sh \
  "Java service dies with OutOfMemoryError and exit code 137"
```

**4. Verify the AI's suggestions against what you know:**
- Exit code 137 = killed by signal 9 (SIGKILL) — OOM killer
- `GC overhead limit exceeded` — Java garbage collector cannot keep up
- The AI should suggest checking JVM heap settings, investigating memory leaks

**5. Take action based on verified diagnosis, not raw AI output.**

---

## Step 8 — Privacy And Local-First Thinking

Why run a local model instead of sending logs to a cloud AI?

1. **Logs contain sensitive data.** IP addresses, usernames, session tokens, database query parameters. Sending these to an external API means your internal infrastructure details are processed by a third party.

2. **Credentials sometimes appear in logs.** Application bugs, misconfigured logging, debug mode left on in production. A local model processes this without it ever leaving your network.

3. **Compliance requirements.** Many industries (healthcare, finance, government) have regulations about data leaving the organization's control.

4. **Availability.** A local model works when your internet is down or the cloud provider has an outage — which is exactly when you need it most during an incident.

The tradeoff: local models are less capable than frontier cloud models (GPT-4, Claude). For log summarization and runbook drafting, small local models are usually good enough. For complex reasoning, you may need cloud models — but then you must scrub sensitive data from the prompt first.

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
yeast destroy
```

---

## Quick Recap

In Lab 30 — AI-Assisted DevOps And Local LLM Ops, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What LLMs are: pattern-matching text generators — not oracles, not reliable sources of truth
- Ollama: running local models with a simple HTTP API
- The API: `api/generate`, `stream: false`, parsing the response
- Log analysis script: formatting a structured prompt, feeding logs, parsing output
- Runbook drafting: using AI to generate a first draft for human review
- Error explanation: accelerating research on unfamiliar errors
- The safety rules: AI suggests, humans approve; verify every claim; never send secrets to cloud AI
- The limits: AI does not know your specific infrastructure, can hallucinate, needs domain-expert review
- Privacy rationale: why local models matter for sensitive operational data

---

## What You Have Built

You have completed the DevOps Bootcamp With Yeast. Thirty labs. Cover to cover.

Here is what you can now do:

**Systems:** Operate Linux servers. Configure hostname, timezone, firewall, automatic updates, SSH hardening. Troubleshoot using logs-first methodology.

**Automation:** Write idempotent Bash scripts. Use Ansible for configuration management at scale. Manage multi-server clusters with templates and inventory groups.

**Containers:** Build, scan, and harden Docker images. Run multi-service applications with Compose. Understand the container model from namespaces up.

**CI/CD:** Write GitHub Actions workflows. Build images in CI, scan them, push to a private registry. Run self-hosted runners on your own infrastructure.

**Delivery:** Blue/green and canary deployments. Progressive traffic shifting. Zero-downtime rollouts and instant rollbacks.

**Observability:** Prometheus metrics, PromQL, Grafana dashboards. Centralized logging with Loki and Promtail. Distributed tracing with OpenTelemetry and Jaeger.

**Reliability:** SLOs and error budgets. Alert quality over alert quantity. Backup and restore drills. Chaos engineering and MTTR measurement.

**IaC:** Terraform fundamentals — plan, apply, state, variables, outputs. Modules and environment separation.

**GitOps:** Argo CD. Git as the source of truth. Reconciliation, drift detection, rollback by commit.

**Kubernetes:** k3s clusters, Pods, Deployments, Services, Ingress, ConfigMaps, Secrets, PersistentVolumes. The complete delivery cycle on K8s.

**AI Ops:** Local LLM operations with Ollama. Log analysis, runbook drafting, error explanation — with privacy and human oversight.

These are not theoretical skills. You built real systems on real VMs. You broke things on purpose and fixed them. You know what healthy looks like because you defined it, measured it, and defended it.

That is what a DevOps engineer does.
