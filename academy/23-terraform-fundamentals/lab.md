# Lab 23 — Terraform Fundamentals

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate to advanced |
| Estimated time | 75-120 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2238 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Comfort reading Terraform HCL examples

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

You have configured servers with Ansible. But Ansible manages what is on a server — packages, services, files. What about the server itself? What about the network it is on, the DNS record that points to it, the firewall rules that protect it?

When those things are created manually, they are invisible. You click through a cloud console, create some resources, and now you have infrastructure that exists nowhere except in the cloud provider's database. If someone deletes it accidentally, or if you need to recreate it in a new region, you have to remember what you did.

Terraform manages infrastructure as code. You describe what you want — servers, networks, databases, DNS records — in `.tf` files. Terraform calculates what needs to be created, changed, or deleted to reach that state. It tracks what exists in a state file. Infrastructure becomes reproducible, version-controlled, and reviewable.

---

## Before You Start — Understanding The Concepts

### What Is Infrastructure As Code (IaC)?

Infrastructure as code means treating infrastructure configuration the same way you treat application code: stored in version control, reviewed before applying, testable, and reproducible.

Without IaC: you SSH in and run commands, or click through cloud consoles. The configuration exists only in the running system — if it is destroyed, recreating it requires memory or documentation (which is always out of date).

With IaC: the desired state is in files. You can diff it, review it, roll it back, and apply it to a new environment. The gap between "what you intended" and "what exists" becomes visible.

### What Is Terraform?

Terraform is an IaC tool by HashiCorp. It is declarative: you describe the desired state, not the steps. Terraform figures out the steps to get there.

Terraform supports hundreds of "providers" — plugins that let it manage resources in AWS, GCP, Azure, GitHub, Cloudflare, Kubernetes, Docker, and many more. Each provider exposes "resources" (things you can create) and "data sources" (things you can read).

### What Is A Provider?

A provider is a Terraform plugin that knows how to talk to a specific API. `hashicorp/aws` knows how to create EC2 instances. `hashicorp/docker` knows how to run Docker containers. `hashicorp/local` knows how to manage local files.

In this lab you will use `hashicorp/local` and `hashicorp/random` — providers that work without cloud credentials, so you can learn Terraform concepts without a cloud account.

### What Is A Resource?

A resource is a piece of infrastructure Terraform manages. In code:

```hcl
resource "local_file" "greeting" {
  filename = "/tmp/hello.txt"
  content  = "Hello, Terraform!"
}
```

`local_file` is the resource type. `greeting` is the name you give this specific resource. Together, `local_file.greeting` is the resource address.

### What Is The State File?

Terraform tracks what it has created in a state file (`terraform.tfstate`). This is how it knows what to update or destroy when you change your config.

The state file is the connection between your code and the real world. It maps resource addresses in your code to real resource IDs in the provider.

**Important:** The state file can contain sensitive data (credentials, IP addresses). Never commit it to public git. In production, store state remotely (Terraform Cloud, S3 + DynamoDB).

### What Is `terraform plan`?

`terraform plan` shows you what Terraform will do — before it does it. It compares your code to the current state and tells you: "I will create X, update Y, and destroy Z."

Always run `terraform plan` before `terraform apply`. Review it. Only apply if it looks correct.

### What Is `terraform apply`?

`terraform apply` executes the plan — creates, updates, or destroys resources to match your code. It shows the plan one more time and asks for confirmation before making changes.

### What Is HCL?

HCL (HashiCorp Configuration Language) is the language Terraform uses. It is a declarative language with:
- Blocks: `resource`, `variable`, `output`, `locals`, `data`
- Arguments: `key = value` pairs inside blocks
- Expressions: `var.name`, `resource.type.name.attribute`, string interpolation `"${var.name}"`

---

## What You Are Building

A series of Terraform configurations using local providers — no cloud credentials needed. You will:
1. Write your first resource
2. Use variables and outputs
3. Update a resource and observe the plan
4. Destroy resources
5. Understand state

---

## Starting The Lab

```bash
cd 23-terraform-fundamentals
yeast up
yeast ssh tf
terraform version
```

---

## Step 1 — Your First Resource

```bash
mkdir -p /home/ubuntu/tf-lab && cd /home/ubuntu/tf-lab
```

Create `main.tf`:

```bash
cat > main.tf << 'EOF'
terraform {
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.4"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5"
    }
  }
}

resource "random_id" "server_id" {
  byte_length = 4
}

resource "local_file" "server_config" {
  filename = "/tmp/server-config.txt"
  content  = <<-EOT
    hostname = "lab-${random_id.server_id.hex}"
    environment = "learning"
    created_by = "terraform"
  EOT
}
EOF
```

Initialize Terraform (downloads providers):

```bash
terraform init
```

`terraform init` reads your `required_providers` block, downloads the specified providers, and sets up the working directory. The providers are stored in `.terraform/`.

Plan:

```bash
terraform plan
```

Output shows:
- `random_id.server_id` will be created
- `local_file.server_config` will be created

Apply:

```bash
terraform apply
```

Terraform shows the plan again and asks: "Do you want to perform these actions?" Type `yes`.

Verify:

```bash
cat /tmp/server-config.txt
terraform state list
terraform show
```

`terraform state list` lists all resources Terraform is managing. `terraform show` shows the current state in a readable format.

---

## Step 2 — Variables

Hard-coded values in resources are bad practice. Use variables:

```bash
cat > variables.tf << 'EOF'
variable "environment" {
  description = "Deployment environment (learning, staging, prod)"
  type        = string
  default     = "learning"
}

variable "hostname_prefix" {
  description = "Prefix for generated hostnames"
  type        = string
  default     = "lab"
}

variable "replica_count" {
  description = "Number of config files to generate"
  type        = number
  default     = 3
}
EOF
```

Update `main.tf` to use variables:

```bash
cat > main.tf << 'EOF'
terraform {
  required_providers {
    local  = { source = "hashicorp/local", version = "~> 2.4" }
    random = { source = "hashicorp/random", version = "~> 3.5" }
  }
}

resource "random_id" "server_id" {
  count       = var.replica_count
  byte_length = 4
}

resource "local_file" "server_config" {
  count    = var.replica_count
  filename = "/tmp/server-${count.index + 1}-config.txt"
  content  = <<-EOT
    hostname    = "${var.hostname_prefix}-${random_id.server_id[count.index].hex}"
    environment = "${var.environment}"
    index       = ${count.index + 1}
  EOT
}
EOF
```

New concepts:
- `count` — creates multiple instances of a resource. `count.index` is 0-based.
- `var.name` — reference a variable
- `random_id.server_id[count.index].hex` — reference the nth instance of another resource

Plan with defaults:

```bash
terraform plan
```

Plan overriding variables:

```bash
terraform plan -var="replica_count=5" -var="environment=staging"
```

Apply:

```bash
terraform apply
```

```bash
ls /tmp/server-*-config.txt
cat /tmp/server-1-config.txt
```

---

## Step 3 — Outputs

Outputs expose values from your Terraform config — useful for passing values to other configs or displaying important info after apply:

```bash
cat > outputs.tf << 'EOF'
output "server_hostnames" {
  description = "Generated hostnames for all servers"
  value       = [for id in random_id.server_id : "${var.hostname_prefix}-${id.hex}"]
}

output "config_files" {
  description = "Paths to generated config files"
  value       = local_file.server_config[*].filename
}

output "environment" {
  description = "Current environment"
  value       = var.environment
}
EOF

terraform apply -auto-approve
terraform output
terraform output server_hostnames
```

`-auto-approve` skips the confirmation prompt — only use this in automation, never for production changes you have not reviewed.

---

## Step 4 — Updating A Resource

Change the `environment` variable:

```bash
terraform plan -var="environment=staging"
```

The plan shows: Terraform will update all 3 `local_file` resources (because the content changes). This is a `~` (update) not a `+` (create) or `-` (destroy).

```bash
terraform apply -var="environment=staging"
cat /tmp/server-1-config.txt  # shows "environment = staging"
```

Now reduce `replica_count`:

```bash
terraform plan -var="replica_count=1" -var="environment=staging"
```

The plan shows: 2 resources will be destroyed (`-`). Terraform manages the lifecycle — you do not need to manually delete the extra files.

```bash
terraform apply -var="replica_count=1" -var="environment=staging" -auto-approve
ls /tmp/server-*-config.txt  # only server-1 remains
```

---

## Step 5 — The State File

```bash
cat terraform.tfstate
```

The state file is JSON. It records every resource Terraform manages: the type, the ID within the provider, and all attributes. This is how Terraform knows `local_file.server_config[0]` corresponds to `/tmp/server-1-config.txt`.

**What happens if you modify a resource outside Terraform?**

```bash
echo "manually edited" >> /tmp/server-1-config.txt
terraform plan  # shows a diff — Terraform detected the drift
terraform apply -auto-approve  # restores it to the defined state
cat /tmp/server-1-config.txt  # manual change is gone
```

Terraform always reconciles real state with declared state. Drift is detected and corrected.

---

## Step 6 — Locals

Locals are values computed within Terraform, not exposed as inputs or outputs:

```bash
cat >> main.tf << 'EOF'

locals {
  config_dir  = "/tmp"
  full_prefix = "${var.environment}-${var.hostname_prefix}"
}
EOF

terraform validate  # check for syntax errors
```

`terraform validate` checks HCL syntax and provider-specific validation without contacting any provider. Good to run before `plan`.

---

## Step 7 — Destroy

```bash
terraform destroy
```

Terraform shows you everything it will destroy and asks for confirmation. It removes all resources it manages and updates the state file.

```bash
ls /tmp/server-*-config.txt 2>/dev/null || echo "all cleaned up"
cat terraform.tfstate  # empty resources list
```

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
exit
yeast destroy
```

---

## Quick Recap

In Lab 23 — Terraform Fundamentals, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What IaC is and why it matters: reproducibility, version control, drift detection
- Terraform's role: managing infrastructure resources declaratively
- `terraform init`: downloads providers
- `terraform plan`: shows what will change, review before applying
- `terraform apply`: executes the plan after confirmation
- `terraform destroy`: removes all managed resources
- HCL: resource blocks, argument syntax, expressions
- Variables: `variable` blocks, `var.name`, `-var` flags
- `count`: creating multiple instances of a resource
- Outputs: `output` blocks, `terraform output`
- State file: the link between code and real resources
- Drift detection: Terraform detects and corrects manual changes
- `terraform validate`: syntax checking before planning

---

## What Is Next

**Lab 24 — Terraform Modules And Environments**

One config for one environment is a start. Lab 24 introduces modules — reusable Terraform components — and shows you how to use the same module for dev and prod with different variable values.
