---
hide:
  - toc
---

<div class="academy-shell">
  <div class="academy-shell__inner">

    <a class="academy-back" href="../index.md">Academy home</a>

    <div class="academy-page-hero">
      <div class="academy-section-label">Twarga Academy · Curriculum</div>
      <h1>30 labs. Five phases. One clear path.</h1>
      <p>
        The course runs from a clean Linux server baseline to Kubernetes delivery and AI-assisted ops.
        Read the phases in order — each lab builds directly on the last one.
      </p>
      <div class="academy-meta">
        <span>30 labs</span>
        <span>5 phases</span>
        <span>Yeast v1.1 surface only</span>
        <span>Linux / KVM first</span>
      </div>
    </div>

    <div class="academy-section">
      <div class="academy-section-label">Phase overview</div>
      <table class="academy-table">
        <thead>
          <tr>
            <th>Phase</th>
            <th>Labs</th>
            <th>Focus</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><strong>1 · Systems</strong></td>
            <td>01–05</td>
            <td>Linux server baseline, web servers, reverse proxy, databases, troubleshooting</td>
          </tr>
          <tr>
            <td><strong>2 · Automation</strong></td>
            <td>06–09</td>
            <td>Bash scripting, Ansible single and multi-VM, secrets and configuration management</td>
          </tr>
          <tr>
            <td><strong>3 · Containers &amp; delivery</strong></td>
            <td>10–16</td>
            <td>Docker, Compose, container hardening, GitHub Actions CI, self-hosted runners, registry, progressive delivery</td>
          </tr>
          <tr>
            <td><strong>4 · Operations</strong></td>
            <td>17–22</td>
            <td>Prometheus and Grafana, centralized logging, distributed tracing, SLOs and error budgets, backup drills, chaos drills</td>
          </tr>
          <tr>
            <td><strong>5 · Platform</strong></td>
            <td>23–30</td>
            <td>Terraform, GitOps, end-to-end delivery platform, Kubernetes with k3s, AI-assisted DevOps</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="academy-section">
      <div class="academy-section-label">All 30 labs</div>

      <div class="academy-phase-header">
        <span class="academy-phase-number">1</span>
        <h2>Systems</h2>
        <span>Labs 01–05</span>
      </div>
      <table class="academy-table">
        <thead>
          <tr><th>#</th><th>Lab</th><th>What you build</th></tr>
        </thead>
        <tbody>
          <tr><td>01</td><td><a href="../01-linux-server-baseline/lab.md">Linux Server Baseline</a></td><td>Clean operational Linux server — users, SSH hardening, firewall, updates</td></tr>
          <tr><td>02</td><td><a href="../02-static-site-nginx/lab.md">Static Site With Nginx</a></td><td>Web server setup, virtual hosts, logs, and SSH port forwarding to browser</td></tr>
          <tr><td>03</td><td><a href="../03-reverse-proxy-backend-app/lab.md">Reverse Proxy To Backend App</a></td><td>Nginx reverse proxy to a Python/Node backend — traffic flow and headers</td></tr>
          <tr><td>04</td><td><a href="../04-database-backed-app/lab.md">Database-Backed App</a></td><td>App-to-database connectivity, persistence, and basic query validation</td></tr>
          <tr><td>05</td><td><a href="../05-linux-troubleshooting-drill/lab.md">Linux Troubleshooting Drill</a></td><td>Logs-first debugging — journalctl, systemd, strace, broken services</td></tr>
        </tbody>
      </table>

      <div class="academy-phase-header">
        <span class="academy-phase-number">2</span>
        <h2>Automation</h2>
        <span>Labs 06–09</span>
      </div>
      <table class="academy-table">
        <thead>
          <tr><th>#</th><th>Lab</th><th>What you build</th></tr>
        </thead>
        <tbody>
          <tr><td>06</td><td><a href="../06-bash-automation-server-setup/lab.md">Bash Automation For Server Setup</a></td><td>Repeatable shell scripts for server provisioning — idempotence and error handling</td></tr>
          <tr><td>07</td><td><a href="../07-ansible-one-server/lab.md">Ansible For One Server</a></td><td>Ansible playbooks, inventory, and idempotent config management on one VM</td></tr>
          <tr><td>08</td><td><a href="../08-ansible-multi-vm-web-cluster/lab.md">Ansible For Multi-VM Web Cluster</a></td><td>Multi-host Ansible — roles, templates, group_vars, and multi-VM orchestration</td></tr>
          <tr><td>09</td><td><a href="../09-secrets-configuration-management/lab.md">Secrets And Configuration Management</a></td><td>Vault-style secret separation, Ansible Vault, environment-aware config</td></tr>
        </tbody>
      </table>

      <div class="academy-phase-header">
        <span class="academy-phase-number">3</span>
        <h2>Containers &amp; Delivery</h2>
        <span>Labs 10–16</span>
      </div>
      <table class="academy-table">
        <thead>
          <tr><th>#</th><th>Lab</th><th>What you build</th></tr>
        </thead>
        <tbody>
          <tr><td>10</td><td><a href="../10-docker-fundamentals-vm/lab.md">Docker Fundamentals On A VM</a></td><td>Containers, volumes, logs, and networking inside a Yeast VM</td></tr>
          <tr><td>11</td><td><a href="../11-compose-multi-service-app/lab.md">Compose Multi-Service App</a></td><td>Docker Compose — service discovery, data volumes, dependency order</td></tr>
          <tr><td>12</td><td><a href="../12-container-build-scan-hardening/lab.md">Container Build, Scan, Hardening</a></td><td>Safer image builds, vulnerability scanning, tagging strategy</td></tr>
          <tr><td>13</td><td><a href="../13-ci-github-actions/lab.md">CI With GitHub Actions</a></td><td>Pipeline basics — build, test, and lint on push with GitHub Actions</td></tr>
          <tr><td>14</td><td><a href="../14-self-hosted-ci-runner/lab.md">Self-Hosted CI Runner</a></td><td>Self-hosted runner on a Yeast VM — registration, job routing, teardown</td></tr>
          <tr><td>15</td><td><a href="../15-private-container-registry/lab.md">Private Container Registry</a></td><td>Private registry on a VM — push, pull, image promotion workflow</td></tr>
          <tr><td>16</td><td><a href="../16-progressive-delivery-rollback/lab.md">Progressive Delivery And Rollback</a></td><td>Canary patterns, health checks, and safe rollback with snapshots</td></tr>
        </tbody>
      </table>

      <div class="academy-phase-header">
        <span class="academy-phase-number">4</span>
        <h2>Operations</h2>
        <span>Labs 17–22</span>
      </div>
      <table class="academy-table">
        <thead>
          <tr><th>#</th><th>Lab</th><th>What you build</th></tr>
        </thead>
        <tbody>
          <tr><td>17</td><td><a href="../17-prometheus-grafana-monitoring/lab.md">Prometheus And Grafana Monitoring</a></td><td>Metrics collection, alerting rules, and Grafana dashboards</td></tr>
          <tr><td>18</td><td><a href="../18-centralized-logging/lab.md">Centralized Logging</a></td><td>Multi-service log aggregation — Loki or ELK — with search and filtering</td></tr>
          <tr><td>19</td><td><a href="../19-opentelemetry-distributed-tracing/lab.md">OpenTelemetry Distributed Tracing</a></td><td>Trace instrumentation, spans, and Jaeger or Tempo integration</td></tr>
          <tr><td>20</td><td><a href="../20-sre-slos-alerts-error-budgets/lab.md">SRE: SLOs, Alerts, Error Budgets</a></td><td>SLO definition, alerting on burn rate, and error budget tracking</td></tr>
          <tr><td>21</td><td><a href="../21-backup-restore-drill/lab.md">Backup And Restore Drill</a></td><td>Backup strategies, restore validation, and Yeast snapshot integration</td></tr>
          <tr><td>22</td><td><a href="../22-chaos-failure-recovery-drill/lab.md">Chaos And Failure Recovery Drill</a></td><td>Controlled failure injection and recovery — kill processes, corrupt data, restore</td></tr>
        </tbody>
      </table>

      <div class="academy-phase-header">
        <span class="academy-phase-number">5</span>
        <h2>Platform</h2>
        <span>Labs 23–30</span>
      </div>
      <table class="academy-table">
        <thead>
          <tr><th>#</th><th>Lab</th><th>What you build</th></tr>
        </thead>
        <tbody>
          <tr><td>23</td><td><a href="../23-terraform-fundamentals/lab.md">Terraform Fundamentals</a></td><td>Plan, apply, state — Terraform basics with local provider and Yeast VMs</td></tr>
          <tr><td>24</td><td><a href="../24-terraform-modules-environments/lab.md">Terraform Modules And Environments</a></td><td>Reusable modules, workspaces, and environment-aware infrastructure</td></tr>
          <tr><td>25</td><td><a href="../25-gitops-argocd-flux/lab.md">GitOps With Argo CD</a></td><td>Reconciliation loop, desired state in Git, sync and rollback</td></tr>
          <tr><td>26</td><td><a href="../26-end-to-end-delivery-platform/lab.md">End-To-End Delivery Platform</a></td><td>VM platform capstone — full delivery flow from code to running service</td></tr>
          <tr><td>27</td><td><a href="../27-kubernetes-k3s-foundations/lab.md">Kubernetes Foundations With k3s</a></td><td>Cluster setup, pods, deployments, services — k3s on Yeast VMs</td></tr>
          <tr><td>28</td><td><a href="../28-kubernetes-networking-config-storage/lab.md">Kubernetes Networking, Config, Storage</a></td><td>Ingress, ConfigMaps, Secrets, PersistentVolumeClaims</td></tr>
          <tr><td>29</td><td><a href="../29-kubernetes-delivery-capstone/lab.md">Kubernetes Delivery Capstone</a></td><td>Full Kubernetes delivery flow — build, push, deploy, observe, roll back</td></tr>
          <tr><td>30</td><td><a href="../30-ai-assisted-devops-llm-ops/lab.md">AI-Assisted DevOps And Local LLM Ops</a></td><td>Local LLM on a Yeast VM, AI-assisted runbooks, and ops automation patterns</td></tr>
        </tbody>
      </table>
    </div>

    <div class="academy-section">
      <div class="academy-section-label">Where to go next</div>
      <div class="academy-link-grid academy-link-grid--emphasis">
        <a href="../modules/index.md"><strong>Modules</strong><br><span style="color:var(--yeast-text-secondary);font-weight:400">Browse the course in five themed blocks.</span></a>
        <a href="../labs/index.md"><strong>Labs</strong><br><span style="color:var(--yeast-text-secondary);font-weight:400">Open the full 30-lab set.</span></a>
        <a href="../support/index.md"><strong>Source contract</strong><br><span style="color:var(--yeast-text-secondary);font-weight:400">Read the course rules and constraints.</span></a>
      </div>
    </div>

  </div>
</div>
