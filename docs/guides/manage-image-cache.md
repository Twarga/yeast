# Manage Image Cache

Images are cached under `~/.yeast/cache/images`.

Show cached images:

```bash
yeast pull --cached
```

Preview cleanup:

```bash
yeast images clean ubuntu-24.04 --dry-run
```

Remove one cached image:

```bash
yeast images clean ubuntu-24.04
```

Remove all cached images:

```bash
yeast images clean --all
```
