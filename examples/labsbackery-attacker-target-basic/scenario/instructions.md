# Attacker Target Basic

You have two machines:

- `attacker` at `10.20.30.10`
- `target` at `10.20.30.20`

Use the attacker terminal to confirm the target is reachable on the private lab network:

```bash
bash -lc 'echo > /dev/tcp/10.20.30.20/22'
```

Use the target terminal to inspect the provisioned marker file:

```bash
cat /home/yeast/labsbackery-target.txt
```

Expected marker:

```text
labsbackery-ready
```
