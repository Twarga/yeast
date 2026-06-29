# Lab 24 — Terraform Modules And Environments

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate to advanced |
| Estimated time | 75-120 minutes |
| VMs | 1 |
| Minimum VM RAM | 1024 MB |
| SSH ports | 2239 |
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

In Lab 23 you wrote one Terraform config. It works. But now imagine you have two environments — dev and prod. They are almost identical, but dev has 1 server and prod has 3. Dev uses smaller sizes, prod uses larger. They deploy to different directories.

If you copy-paste the whole config, you have two things to maintain. When you add a new resource, you add it twice. When you fix a bug, you fix it twice. And you inevitably forget one.

Terraform modules solve this. A module is a reusable Terraform component — a directory of `.tf` files that takes variables as input and produces resources as output. You write the logic once, instantiate it multiple times with different variables.

---

## Before You Start — Understanding The Concepts

### What Is A Terraform Module?

Every directory of `.tf` files is a module. When you call `terraform apply`, you are in the root module. A module you call from within another module is a child module.

A module has:
- **Inputs** (`variable` blocks) — what callers must provide
- **Resources** — what it creates
- **Outputs** (`output` blocks) — what it exposes to callers

You call a module with a `module` block:

```hcl
module "dev_server" {
  source      = "./modules/server"
  environment = "dev"
  count       = 1
}
```

### What Is The Module Source?

The `source` argument tells Terraform where to find the module:
- `"./modules/server"` — local directory
- `"hashicorp/consul/aws"` — Terraform Registry
- `"git::https://github.com/org/repo.git//modules/server"` — Git repo

For this lab we use local modules. In production teams, modules live in a shared Git repo or the Terraform Registry.

### What Is Environment Separation?

Each environment (dev, staging, prod) should have its own:
- Terraform state file (so a plan in dev cannot accidentally destroy prod)
- Variable values (different sizes, replica counts, feature flags)
- Apply history

The standard pattern is one directory per environment:

```
environments/
  dev/
    main.tf          ← calls the module with dev values
    terraform.tfstate
  prod/
    main.tf          ← calls the same module with prod values
    terraform.tfstate
```

`terraform apply` in `environments/dev/` only affects dev. Completely separate from prod.

### What Is `terraform.tfvars`?

A `.tfvars` file provides variable values automatically:

```hcl
# environments/prod/terraform.tfvars
environment    = "prod"
replica_count  = 5
```

When you run `terraform apply` in that directory, Terraform loads `terraform.tfvars` automatically. No `-var` flags needed.

### What Is Drift Between Environments?

Drift happens when dev and prod configurations diverge in unexpected ways — a change was made in prod but never in dev, or vice versa. With modules, this is prevented: both environments call the same module. If dev tests a change and it works, prod gets the exact same module logic — just with different variable values.

---

## What You Are Building

```
/home/ubuntu/tf-modules/
  modules/
    server-config/        ← the reusable module
      main.tf
      variables.tf
      outputs.tf
  environments/
    dev/
      main.tf             ← calls module with dev values
      terraform.tfvars
    prod/
      main.tf             ← calls module with prod values
      terraform.tfvars
```

---

## Starting The Lab

```bash
cd 24-terraform-modules-environments
yeast up
yeast ssh tf-modules
mkdir -p /home/ubuntu/tf-modules/modules/server-config
mkdir -p /home/ubuntu/tf-modules/environments/dev
mkdir -p /home/ubuntu/tf-modules/environments/prod
cd /home/ubuntu/tf-modules
```

---

## Step 1 — Write The Module

```bash
cat > modules/server-config/variables.tf << 'EOF'
variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
}

variable "hostname_prefix" {
  description = "Prefix for server hostnames"
  type        = string
  default     = "server"
}

variable "replica_count" {
  description = "Number of servers to configure"
  type        = number
  default     = 1
}

variable "output_dir" {
  description = "Directory to write config files to"
  type        = string
  default     = "/tmp"
}
EOF

cat > modules/server-config/main.tf << 'EOF'
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

resource "local_file" "config" {
  count    = var.replica_count
  filename = "${var.output_dir}/${var.environment}-server-${count.index + 1}-config.txt"
  content  = <<-EOT
    hostname    = "${var.hostname_prefix}-${var.environment}-${random_id.server_id[count.index].hex}"
    environment = "${var.environment}"
    replica     = ${count.index + 1} of ${var.replica_count}
    managed_by  = "terraform"
  EOT
}
EOF

cat > modules/server-config/outputs.tf << 'EOF'
output "server_ids" {
  description = "Generated server IDs"
  value       = random_id.server_id[*].hex
}

output "config_files" {
  description = "Paths to generated config files"
  value       = local_file.config[*].filename
}

output "server_count" {
  description = "Number of servers configured"
  value       = var.replica_count
}
EOF
```

---

## Step 2 — Dev Environment

```bash
cat > environments/dev/main.tf << 'EOF'
terraform {
  required_providers {
    local  = { source = "hashicorp/local", version = "~> 2.4" }
    random = { source = "hashicorp/random", version = "~> 3.5" }
  }
}

module "servers" {
  source = "../../modules/server-config"

  environment     = var.environment
  hostname_prefix = var.hostname_prefix
  replica_count   = var.replica_count
  output_dir      = var.output_dir
}

variable "environment"     { type = string }
variable "hostname_prefix"  { type = string }
variable "replica_count"    { type = number }
variable "output_dir"       { type = string }

output "servers" {
  value = module.servers.server_ids
}
EOF

cat > environments/dev/terraform.tfvars << 'EOF'
environment     = "dev"
hostname_prefix = "lab"
replica_count   = 1
output_dir      = "/tmp"
EOF
```

---

## Step 3 — Prod Environment

```bash
cat > environments/prod/main.tf << 'EOF'
terraform {
  required_providers {
    local  = { source = "hashicorp/local", version = "~> 2.4" }
    random = { source = "hashicorp/random", version = "~> 3.5" }
  }
}

module "servers" {
  source = "../../modules/server-config"

  environment     = var.environment
  hostname_prefix = var.hostname_prefix
  replica_count   = var.replica_count
  output_dir      = var.output_dir
}

variable "environment"     { type = string }
variable "hostname_prefix"  { type = string }
variable "replica_count"    { type = number }
variable "output_dir"       { type = string }

output "servers" {
  value = module.servers.server_ids
}
EOF

cat > environments/prod/terraform.tfvars << 'EOF'
environment     = "prod"
hostname_prefix = "lab"
replica_count   = 3
output_dir      = "/tmp"
EOF
```

---

## Step 4 — Apply Dev

```bash
cd /home/ubuntu/tf-modules/environments/dev
terraform init
terraform plan
terraform apply -auto-approve

terraform output
cat /tmp/dev-server-1-config.txt
```

---

## Step 5 — Apply Prod Independently

```bash
cd /home/ubuntu/tf-modules/environments/prod
terraform init
terraform plan
terraform apply -auto-approve

terraform output
ls /tmp/prod-server-*-config.txt
```

Each environment has its own state file. Applying prod has zero effect on dev.

---

## Step 6 — Update The Module And Propagate

Add a new field to the module config:

```bash
# Edit the module
sed -i 's/managed_by  = "terraform"/managed_by  = "terraform"\n    last_updated = "${timestamp()}"/' \
  /home/ubuntu/tf-modules/modules/server-config/main.tf
```

Now plan both environments:

```bash
cd /home/ubuntu/tf-modules/environments/dev
terraform plan  # shows the update to server-1

cd /home/ubuntu/tf-modules/environments/prod
terraform plan  # shows the update to all 3 prod servers
```

The same module change propagates to both environments. Apply dev first, validate, then apply prod:

```bash
cd /home/ubuntu/tf-modules/environments/dev
terraform apply -auto-approve
cat /tmp/dev-server-1-config.txt  # has last_updated field

cd /home/ubuntu/tf-modules/environments/prod
terraform apply -auto-approve
cat /tmp/prod-server-1-config.txt  # same change
```

---

## Step 7 — State Separation

Verify the two state files are independent:

```bash
ls -la /home/ubuntu/tf-modules/environments/*/terraform.tfstate
```

```bash
# Dev state only knows about dev resources
cd /home/ubuntu/tf-modules/environments/dev
terraform state list

# Prod state only knows about prod resources
cd /home/ubuntu/tf-modules/environments/prod
terraform state list
```

If you destroy dev, prod is unaffected:

```bash
cd /home/ubuntu/tf-modules/environments/dev
terraform destroy -auto-approve
ls /tmp/dev-server-* 2>/dev/null || echo "dev configs gone"
ls /tmp/prod-server-*  # prod still exists
```

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
cd /home/ubuntu/tf-modules/environments/prod && terraform destroy -auto-approve
exit
yeast destroy
```

---

## Quick Recap

In Lab 24 — Terraform Modules And Environments, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- What a Terraform module is: a reusable component with inputs, resources, and outputs
- Module directory structure: `variables.tf`, `main.tf`, `outputs.tf`
- `module` block: calling a module, passing variable values
- `source`: local path modules vs registry modules
- `terraform.tfvars`: automatic variable loading per environment directory
- Environment separation: independent state files, independent apply/destroy
- Module propagation: update the module once, plan/apply to each environment
- The standard pattern: `modules/` directory + `environments/` directory

---

## What Is Next

**Lab 25 — GitOps With Argo CD Or Flux**

Terraform manages infrastructure. GitOps manages application deployment. Lab 25 installs Argo CD on your Kubernetes cluster and configures it to watch a Git repository — when you push a change to the repo, Argo CD automatically applies it to the cluster. Git becomes the source of truth for your deployment state.
