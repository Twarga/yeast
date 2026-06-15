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
