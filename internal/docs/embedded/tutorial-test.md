# Yeast Manual Test

Run this as a real client-side test:

```bash
yeast version
yeast doctor
mkdir -p ~/yeast-template-test
cd ~/yeast-template-test
yeast init --list-templates
yeast init --template ubuntu-basic
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

- doctor passes blockers
- templates list successfully
- Ubuntu image verifies
- VM becomes reachable by SSH
- `down` stops the VM
- `up` starts it again
- `destroy` removes runtime files
- image cache remains in `~/.yeast/cache`
