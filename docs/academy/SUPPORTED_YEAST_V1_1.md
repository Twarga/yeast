# Supported Yeast v1.1 Surface For This Bootcamp

This bootcamp must only teach commands and config fields that exist in Yeast v1.1.

Use this file as the source-of-truth checklist when writing or fixing labs.

## Supported Commands

These commands exist in the current Yeast v1.1 code/docs:

| Area | Supported commands |
|---|---|
| Host/project | `yeast doctor`, `yeast init`, `yeast version`, `yeast docs`, `yeast completion` |
| Images | `yeast pull [image]`, `yeast pull --list`, `yeast pull --cached`, `yeast images clean [image]`, `yeast images clean --all`, `yeast images clean --dry-run` |
| Lifecycle | `yeast up`, `yeast down`, `yeast destroy` |
| Guest access | `yeast status`, `yeast inspect <instance>`, `yeast logs <instance>`, `yeast ssh [instance]`, `yeast exec [instance] -- <command...>`, `yeast copy <instance> --to-guest <source> <destination>`, `yeast copy <instance> --from-guest <source> <destination>` |
| Provisioning | `yeast provision [instance]` |
| Snapshots | `yeast snapshot <instance> <name>`, `yeast snapshots <instance>`, `yeast restore <instance> <name>`, `yeast delete-snapshot <instance> <name>` |
| Updates | `yeast update`, `yeast update --check`, `yeast update --force`, `yeast update --version <tag>` |

## Supported Global Flags

These global flags are available from the root command:

| Flag | Use |
|---|---|
| `--json` | Machine-readable JSON output |
| `--events` | JSON Lines lifecycle events |
| `--quiet`, `-q` | Suppress progress output where supported |
| `--help`, `-h` | Help output |

## Important Command Limits

Do not document these as Yeast v1.1 features:

| Not supported | Correct approach |
|---|---|
| `yeast ssh <vm> -- -L ...` | Use normal `ssh -L ... -p <ssh_port> <user>@<management_ip>` |
| `yeast port-forward` | Use normal SSH local forwarding |
| `yeast expose` | Use normal SSH local forwarding |
| `yeast open` | Open the browser manually after a tunnel is running |
| `yeast vm ip` | Use `yeast status` or `yeast status --json` |
| `yeast logs` for guest app logs | `yeast logs` reads QEMU/runtime logs; use `journalctl`, `docker logs`, or app logs inside the VM for guest services |

## Supported `yeast.yaml` Fields

These fields are supported in v1.1:

```yaml
version: 1
management_host: 127.0.0.1

networks:
  - name: lab
    cidr: 10.10.10.0/24

provision:
  packages:
    - curl
  files:
    - source: ./local-file
      destination: /home/ubuntu/local-file
      permissions: "0644"
  shell:
    - echo "project-level provisioning"

instances:
  - name: web
    hostname: web
    image: ubuntu-22.04
    memory: 1024
    cpus: 2
    disk_size: 20G
    ssh_port: 2202
    user: ubuntu
    sudo: nopasswd
    env:
      APP_ENV: lab
    user_data: |
      #cloud-config
      package_update: true
    networks:
      - name: lab
        ipv4: 10.10.10.10
    provision:
      packages:
        - nginx
      files:
        - source: ./index.html
          destination: /var/www/html/index.html
          permissions: "0644"
      shell:
        - systemctl enable --now nginx
```

## Unsupported `yeast.yaml` Fields

Do not use these fields in this bootcamp until Yeast implements and documents them:

```yaml
ports:
host_port:
guest_port:
http_port:
forward:
port_forward:
mounts:
volumes:
```

Docker Compose and Kubernetes manifests may use their own `ports:` fields. That is different from Yeast `yeast.yaml`.

## Networking Rules

Yeast v1.1 has two networking ideas:

| Network type | What it does |
|---|---|
| Management SSH | Host-to-VM SSH through `management_host` and `ssh_port` |
| Private lab network | VM-to-VM traffic using one project network with static IPv4 addresses |

Yeast v1.1 does not document general host-to-guest HTTP port forwarding.

If a lab needs a browser on the laptop to reach Grafana, Prometheus, Nginx, Argo CD, Jaeger, Ollama, or a registry UI/API, use SSH local forwarding. See `ACCESS.md`.

## Lab Author Checklist

Before marking a lab as tested:

- Run `yeast doctor` on the host.
- Run `yeast up`.
- Run `yeast status`.
- Verify every VM is `running`.
- If the lab needs browser access, create the documented SSH tunnel.
- Run all commands exactly as written.
- Run `bash assets/validate.sh`.
- Run `yeast destroy`.
- Record the result in `PROGRESS.md`.
