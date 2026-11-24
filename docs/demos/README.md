# Demo recordings

[VHS](https://github.com/charmbracelet/vhs) `.tape` scripts that record the
real animated CLI into GIFs for docs and the website.

| Tape | Shows | Output |
|---|---|---|
| `yeast-up.tape` | `init` → animated Pixel Yeast Cell boot → `status` | `yeast-up.gif` |
| `reset-superpowers.tape` | live snapshot → break guest → ~2s restore | `yeast-reset.gif` |
| `screenshot-input.tape` | QMP `screendump` PNG + `send-key`/`--text` | `yeast-screenshot.gif` |
| `fork.tape` | `fork --count 3 --boot` → status → `destroy --forks` | `yeast-fork.gif` |

Record (needs a KVM host with a cached `ubuntu-24.04` image):

```bash
vhs docs/demos/reset-superpowers.tape
```

GIFs are gitignored — regenerate them per release, don't commit binaries.
