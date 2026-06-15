# SSH Troubleshooting

If Yeast waits for SSH too long:

```bash
yeast status
yeast logs web --tail 100
ssh -p 2222 yeast@127.0.0.1 -v
```

Common causes:

- first boot is still running cloud-init
- SSH key is missing
- VM failed to boot
- requested `ssh_port` is already used

Check your SSH public key:

```bash
ls ~/.ssh/id_ed25519.pub ~/.ssh/id_rsa.pub
```

## Check The Port

Use `yeast status` to find the management port:

```bash
yeast status
```

Then test SSH manually:

```bash
ssh -p <port> yeast@127.0.0.1 -v
```

If your project uses a custom `user`, replace `yeast` with that user.

## First Boot Can Be Slow

Cloud images may take a few minutes on first boot, especially while cloud-init is expanding disks, configuring users, or installing packages.

Check logs before assuming the VM is broken:

```bash
yeast logs web --tail 120
```
