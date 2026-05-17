# Yeast v0.1.0 Manual Test

Run this as a real client-side test:

```bash
curl -fsSL https://raw.githubusercontent.com/Twarga/yeast/v0.1.0/install.sh | YEAST_REF=v0.1.0 bash
yeast version
yeast doctor
mkdir -p ~/yeast-v010-test
cd ~/yeast-v010-test
yeast init
yeast pull ubuntu-24.04
yeast up
yeast status
yeast status --json
yeast ssh web
yeast down
yeast up
yeast destroy
```

Pass criteria:

- version is `v0.1.0`
- doctor passes blockers
- Ubuntu image verifies
- VM becomes reachable by SSH
- `down` stops the VM
- `up` starts it again
- `destroy` removes runtime files
- image cache remains in `~/.yeast/cache`
