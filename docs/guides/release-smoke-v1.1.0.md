# Release Smoke Test v1.1.0

This page is for maintainers validating a release on a real Linux/KVM host.

For the current detailed checklist, see the repository smoke-test docs and the embedded terminal topic:

```bash
yeast docs release-smoke
```

Minimum release checks:

- download release tarball
- verify `SHA256SUMS.txt`
- confirm tarball contains `yeast`
- install manually
- install with `install.sh`
- test `yeast update --force --version v1.1.0`
- run single-VM lifecycle
- run guest control
- run snapshot/restore
- run two-VM networking
