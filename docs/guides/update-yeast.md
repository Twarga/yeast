# Update Yeast

Use `yeast update` to replace the current Yeast binary with a GitHub release binary.

The updater downloads the release tarball, verifies checksums, extracts `yeast`, and replaces the current binary.

## Check For Updates

```bash
yeast update --check
```

This checks the release without installing it.

## Install The Latest Release

```bash
yeast update
```

## Install A Specific Version

```bash
yeast update --version v1.1.5
```

## Force Reinstall

```bash
yeast update --force --version v1.1.5
```

Use `--force` when you want to reinstall the same version or replace a local smoke-test binary.

## Verify The Result

```bash
yeast version
yeast doctor
```

## Troubleshooting

If update fails:

1. check network access to GitHub releases
2. confirm the requested tag exists
3. confirm the release has `yeast_linux_amd64.tar.gz`
4. confirm the release has `SHA256SUMS.txt`
5. run `yeast version` to see what is still installed

For release validation, see [Release Smoke Test](release-smoke-v1.1.5.md).
