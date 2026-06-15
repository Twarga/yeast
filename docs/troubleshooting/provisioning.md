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

## Check File Provisioning

If a file did not appear in the guest, confirm the host source path exists from the project folder:

```bash
pwd
ls -la ./site/index.html
```

Then check the guest destination:

```bash
yeast exec web -- ls -la /home/yeast/site/index.html
```

## Debug Shell Commands

Run the failing command manually:

```bash
yeast exec web -- sudo systemctl status caddy --no-pager
```

If the manual command fails, fix the command before rerunning provisioning.
