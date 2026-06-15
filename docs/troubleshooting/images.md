# Image Troubleshooting

List supported images:

```bash
yeast pull --list
```

Show cached images:

```bash
yeast pull --cached
```

If an image cache entry is broken, remove it and pull again:

```bash
yeast images clean ubuntu-24.04
yeast pull ubuntu-24.04
```

Manual images require manual download or preparation before use.
