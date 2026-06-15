# Provisioning Troubleshooting

Check the VM log:

```bash
yeast logs web --tail 120
```

Check the service manually:

```bash
yeast exec web -- systemctl status <service> --no-pager
```

Rerun provisioning:

```bash
yeast provision web
```

Common causes:

- package name is wrong for the guest distribution
- file source path does not exist on the host
- shell command is not idempotent
- command needs `sudo`
