# Update Yeast

Check for an update:

```bash
yeast update --check
```

Install the latest release:

```bash
yeast update
```

Install a specific version:

```bash
yeast update --version v1.1.0
```

Force reinstall:

```bash
yeast update --force --version v1.1.0
```

The updater downloads the release tarball, verifies checksums, extracts `yeast`, and replaces the current binary.
