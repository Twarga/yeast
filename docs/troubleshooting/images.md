# Image Troubleshooting

List supported images:

```bash
yeast pull --list
```

Show cached images:

```bash
yeast pull --cached
```

If an image cache entry is broken, remove it and start the project again:

```bash
yeast images clean ubuntu-24.04
yeast up
```

`yeast up` downloads the image again if it is missing.

Manual images require manual download or preparation before use.
