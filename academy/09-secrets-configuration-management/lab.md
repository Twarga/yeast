# Lab 09 — Secrets And Configuration Management

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Beginner to intermediate |
| Estimated time | 60-90 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2215 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic shell scripting comfort

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you. If not, the lab will tell you when to use a manual SSH tunnel.

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

---

## The Story

Look back at every lab you have built so far. Lab 04 had a database password hardcoded in a Python file: `"password": "changeme123"`. Lab 07 had `ansible_ssh_private_key_file` pointing to a real key path. Lab 06 had credentials as default function arguments.

Now imagine you push that code to GitHub. The password is in the repository forever — in the commit history, in every clone, readable by anyone with access. This happens constantly in real teams. It is one of the most common causes of security breaches.

Today you fix that habit. You learn how secrets are separated from code, how they are injected into applications at runtime, and how tools like Ansible Vault encrypt secrets so they can be safely stored alongside the rest of your infrastructure code.

---

## Before You Start — Understanding The Concepts

### What Is A Secret?

A secret is any value that grants access to a resource or system:
- Database passwords
- API keys (AWS, Stripe, Twilio, etc.)
- SSH private keys
- TLS certificates and private keys
- OAuth tokens
- Encryption keys

The defining property of a secret: if it leaks, someone else can impersonate you or access your data.

### Why Hardcoded Secrets Are Dangerous

When a secret is written directly in code:

```python
DB_PASS = "my-super-secret-password"
```

It is:
- Committed to version control — now in the git history permanently
- Readable by everyone with repo access
- The same in every environment (dev, staging, production)
- Impossible to rotate without a code change and redeploy

GitHub scans public repositories for common secret patterns and notifies you. But private repos, internal git servers, and CI logs are not scanned by default. The secret is in the clear everywhere the code exists.

### The Principle: Separate Config From Code

The right pattern is called "The Twelve-Factor App" principle III: store config in the environment.

Config (things that vary between deployments) goes in:
- Environment variables
- `.env` files (loaded at runtime, not committed to git)
- A secrets manager (Vault, AWS Secrets Manager, etc.)

Code stays in git. Config stays out of git.

### What Is An Environment Variable?

An environment variable is a key-value pair set in the shell or process environment. Every running process inherits the environment variables of its parent.

```bash
export DB_PASS="my-secret"
echo $DB_PASS
```

In Python:

```python
import os
password = os.getenv("DB_PASS")
```

The code reads the password from the environment — it never contains the password itself. You set the environment variable on the server through a mechanism that is not version control.

### What Is A `.env` File?

A `.env` file is a text file containing environment variable assignments:

```
DB_HOST=192.168.20.30
DB_PORT=5432
DB_NAME=appdb
DB_USER=appuser
DB_PASS=changeme123
```

Your application loads this file at startup using a library (`python-dotenv`, `godotenv`, etc.) or the systemd `EnvironmentFile` directive.

The key rule: **`.env` files are never committed to git.** Add `.env` to `.gitignore` before you write the first line.

### What Is `.gitignore`?

`.gitignore` is a file in a git repository that lists patterns for files git should not track. Files matching the patterns are invisible to `git add` and `git status`.

```
.env
.env.*
*.secret
secrets/
```

Put this in `.gitignore` from the start of every project. Never add a secret file to git and then try to remove it — it lives in the history forever unless you rewrite history, which is painful and risky.

### What Is Ansible Vault?

Ansible Vault is a feature built into Ansible that encrypts files using AES-256 symmetric encryption. You can encrypt a file containing secrets:

```bash
ansible-vault encrypt secrets.yml
```

Now `secrets.yml` contains encrypted ciphertext, not plaintext. You can safely commit it to git. When Ansible runs a playbook that uses this file, you provide the vault password and Ansible decrypts it in memory.

This solves the problem of "I need secrets in my Ansible playbooks but I cannot put plaintext passwords in git."

### What Is The Difference Between Config And Secrets?

Not all configuration is secret:

| Type | Example | Secret? |
|---|---|---|
| DB host | `192.168.20.30` | No |
| DB port | `5432` | No |
| DB name | `appdb` | No |
| DB user | `appuser` | No |
| DB password | `changeme123` | **Yes** |
| API endpoint | `https://api.stripe.com` | No |
| API key | `sk_live_abc123...` | **Yes** |

Non-secret config can live in version control. Secrets cannot. The distinction matters because it tells you what needs vault/env treatment and what does not.

---

## What You Are Building

One VM running a Python application backed by PostgreSQL. You will:
1. Start from the Lab 04 pattern (hardcoded credentials)
2. Refactor to use `.env` files
3. Use systemd's `EnvironmentFile` directive to inject secrets at runtime
4. Encrypt the `.env` file with Ansible Vault
5. Understand what a real secrets rotation looks like

---

## Starting The Lab

```bash
cd 09-secrets-configuration-management
yeast up
yeast ssh secrets
```

---

## Part 1 — The Problem: Hardcoded Credentials

Set up a small app the wrong way first, so you feel what you are fixing.

```bash
sudo pip3 install psycopg2-binary

mkdir -p /home/ubuntu/app
cat > /home/ubuntu/app/server.py << 'PYEOF'
#!/usr/bin/env python3
import json
import psycopg2
from http.server import HTTPServer, BaseHTTPRequestHandler

# BAD: credentials hardcoded in source code
DB_CONFIG = {
    "host":     "localhost",
    "port":     5432,
    "dbname":   "appdb",
    "user":     "appuser",
    "password": "hardcoded_password_never_do_this",
}

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({"status": "running"}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        pass

if __name__ == "__main__":
    HTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
PYEOF
```

Now look at the file you just created:

```bash
cat /home/ubuntu/app/server.py
```

There is the password, plaintext, inside the source file. If this went to GitHub, anyone with access to the repository would have the database password. Now you understand why this is wrong.

Delete this version:

```bash
rm /home/ubuntu/app/server.py
```

---

## Part 2 — The Fix: Environment Variables

Create the correct version that reads credentials from environment variables:

```bash
cat > /home/ubuntu/app/server.py << 'PYEOF'
#!/usr/bin/env python3
import json
import os
import psycopg2
import psycopg2.extras
from http.server import HTTPServer, BaseHTTPRequestHandler

# GOOD: credentials come from the environment, never from this file
DB_CONFIG = {
    "host":     os.environ["DB_HOST"],
    "port":     int(os.environ.get("DB_PORT", "5432")),
    "dbname":   os.environ["DB_NAME"],
    "user":     os.environ["DB_USER"],
    "password": os.environ["DB_PASS"],
}

def get_conn():
    return psycopg2.connect(**DB_CONFIG)

def init_db():
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute("""
                CREATE TABLE IF NOT EXISTS items (
                    id SERIAL PRIMARY KEY,
                    name TEXT NOT NULL
                )
            """)
        conn.commit()

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        try:
            with get_conn() as conn:
                with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                    cur.execute("SELECT * FROM items ORDER BY id")
                    rows = cur.fetchall()
            body = json.dumps({"items": [dict(r) for r in rows]}).encode()
            self.send_response(200)
        except Exception as e:
            body = json.dumps({"error": str(e)}).encode()
            self.send_response(500)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        pass

if __name__ == "__main__":
    init_db()
    print("App listening on :8000")
    HTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
PYEOF
```

Notice: `os.environ["DB_HOST"]` will raise a `KeyError` if the variable is not set. That is intentional. An application that fails loudly because a required secret is missing is much safer than one that falls back to a hardcoded default or silently connects to the wrong database.

---

## Part 3 — The `.env` File

Create the `.env` file with the actual credentials:

```bash
cat > /home/ubuntu/app/.env << 'EOF'
DB_HOST=localhost
DB_PORT=5432
DB_NAME=appdb
DB_USER=appuser
DB_PASS=lab09secret
EOF

# Set permissions: only the owner can read it
chmod 600 /home/ubuntu/app/.env
ls -la /home/ubuntu/app/.env
```

`chmod 600` means: owner can read and write, group and others have no permissions. This is the correct permission for a file containing secrets.

Verify:

```bash
stat -c "%a %U %G %n" /home/ubuntu/app/.env
```

Expected: `600 ubuntu ubuntu /home/ubuntu/app/.env`

---

## Part 4 — Setting Up PostgreSQL To Accept The Credentials

```bash
sudo -u postgres psql << 'SQL'
CREATE USER appuser WITH PASSWORD 'lab09secret';
CREATE DATABASE appdb OWNER appuser;
GRANT ALL PRIVILEGES ON DATABASE appdb TO appuser;
\q
SQL
```

---

## Part 5 — Systemd EnvironmentFile

The correct way to inject `.env` into a systemd service is the `EnvironmentFile` directive. Systemd reads the file and sets each line as an environment variable before starting the process.

```bash
sudo tee /etc/systemd/system/app.service << 'EOF'
[Unit]
Description=App Server
After=network.target postgresql.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/app
EnvironmentFile=/home/ubuntu/app/.env
ExecStart=/usr/bin/python3 /home/ubuntu/app/server.py
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now app
sudo systemctl status app
```

Check that systemd loaded the environment variables:

```bash
sudo systemctl show app -p Environment
```

You should see `DB_HOST=localhost DB_PORT=5432` etc. listed. The service knows the credentials. The source code does not contain them.

Test the app:

```bash
curl http://localhost:8000
```

Expected: `{"items": []}`

Now check: can you find the password in the source code?

```bash
grep -r "lab09secret" /home/ubuntu/app/server.py
```

Nothing. The password is not in the code.

---

## Part 6 — What Happens When A Secret Is Missing

Test what happens when a required environment variable is missing:

```bash
# Temporarily remove the .env file
sudo mv /home/ubuntu/app/.env /home/ubuntu/app/.env.bak
sudo systemctl restart app
sudo systemctl status app
```

The service fails to start — or starts and immediately crashes — because `os.environ["DB_PASS"]` raises a `KeyError`. Check the journal:

```bash
sudo journalctl -u app --no-pager -n 10
```

You will see a `KeyError: 'DB_PASS'` traceback. The app is loud about the missing secret. This is correct behavior — it is better than silently connecting somewhere wrong.

Restore:

```bash
sudo mv /home/ubuntu/app/.env.bak /home/ubuntu/app/.env
sudo systemctl restart app
```

---

## Part 7 — Ansible Vault

Exit the VM and work from your laptop for this section.

```bash
exit
```

Create a secrets file for an Ansible playbook. First, without encryption:

```bash
cat > group_vars/all/vars.yml << 'EOF'
db_host: localhost
db_port: 5432
db_name: appdb
db_user: appuser
EOF

cat > group_vars/all/vault.yml << 'EOF'
vault_db_pass: lab09secret
vault_api_key: sk_fake_abc123xyz
EOF
```

The convention: prefix vault variables with `vault_`. Non-secret vars go in `vars.yml`. Secrets go in `vault.yml`.

Now encrypt the vault file:

```bash
mkdir -p group_vars/all
ansible-vault encrypt group_vars/all/vault.yml
```

Ansible asks for a password. Enter something you will remember for this lab: `vaultpass`

After encryption, look at the file:

```bash
cat group_vars/all/vault.yml
```

```
$ANSIBLE_VAULT;1.1;AES256
38316139323836656434363065356461303163353165616539633338396563...
```

It is ciphertext. Commit this to git — there is no plaintext secret visible. Anyone who gets the file without the vault password cannot read it.

Edit the file (Ansible decrypts it temporarily for editing):

```bash
ansible-vault edit group_vars/all/vault.yml
```

View the file (decrypts in memory and prints):

```bash
ansible-vault view group_vars/all/vault.yml
```

Decrypt back to plaintext (use sparingly — usually you keep files encrypted):

```bash
ansible-vault decrypt group_vars/all/vault.yml
```

Re-encrypt:

```bash
ansible-vault encrypt group_vars/all/vault.yml
```

When running an Ansible playbook that uses vault-encrypted files:

```bash
ansible-playbook -i inventory.ini site.yml --ask-vault-pass
```

Or store the vault password in a file (never commit this file to git):

```bash
echo "vaultpass" > .vault_pass
chmod 600 .vault_pass
ansible-playbook -i inventory.ini site.yml --vault-password-file .vault_pass
```

---

## Part 8 — Secret Rotation

Secret rotation means changing a credential. In production this happens regularly — either on a schedule or after a suspected compromise. Understanding the process is as important as understanding initial setup.

The correct rotation process for the database password:

**Step 1:** Create the new password in the database (without removing the old one yet):

```bash
yeast ssh secrets
sudo -u postgres psql -c "ALTER USER appuser PASSWORD 'newlab09secret';"
```

**Step 2:** Update the `.env` file on the server:

```bash
sed -i 's/DB_PASS=lab09secret/DB_PASS=newlab09secret/' /home/ubuntu/app/.env
```

**Step 3:** Restart the service:

```bash
sudo systemctl restart app
curl http://localhost:8000
```

Verify it still works.

**Step 4:** If you use Ansible Vault, re-encrypt the new secret:

```bash
# Back on your laptop
ansible-vault edit group_vars/all/vault.yml
# Change vault_db_pass to newlab09secret
# Save and exit
```

**Step 5:** Commit the re-encrypted vault file to git. The old password is no longer in the file. The git history contains encrypted ciphertext for both versions — but without the vault password, neither version is readable.

This is the correct flow. The old plaintext password was never in git at any point.

---

## The Rules To Remember

1. **Never commit a plaintext secret to git.** Ever. Not even to a private repo.
2. **Add `.env` to `.gitignore` before writing the first line of code.**
3. **Use `chmod 600` on every file containing secrets.**
4. **Use `os.environ["KEY"]` not `os.getenv("KEY", "default")` for required secrets** — fail loudly if missing.
5. **Rotate secrets regularly.** Use a process that works without downtime.
6. **Use Ansible Vault (or equivalent) for secrets in infrastructure code.**
7. **The vault password itself is a secret.** Store it in a password manager, not in a file in the repo.

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

In Lab 09 — Secrets And Configuration Management, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- Why hardcoded credentials are dangerous: version control, repo access, no rotation
- Environment variables: how to read secrets at runtime without storing them in code
- `.env` files: the pattern, `chmod 600`, why they must not be committed to git
- `.gitignore`: how to prevent secret files from ever touching git
- Systemd `EnvironmentFile`: injecting secrets into a service at start time
- The correct failure mode: crash loudly on missing secrets rather than silently fail or use defaults
- Ansible Vault: AES-256 encryption of secrets files for safe storage in version control
- Secret rotation: the step-by-step process for changing a credential without downtime

---

## What Is Next

**Lab 10 — Docker Fundamentals On A VM**

You have been managing processes directly on Linux VMs: install the binary, write a systemd service, manage the lifecycle manually. Containers change this model entirely. In Lab 10 you install Docker and learn the container model: images, containers, volumes, networks, and the `docker` command line. Everything you know about Linux applies inside containers — and you will prove it.
