# Manage Image Cache

Yeast stores shared base images under:

```text
~/.yeast/cache/images/
```

Projects reuse these cached images. Removing a cached image does not automatically destroy existing project disks.

## List Supported Images

```bash
yeast pull --list
```

## Show Cached Images

```bash
yeast pull --cached
```

## Pre-Cache An Image

```bash
yeast pull <image>
```

This is optional for supported auto-download images because `yeast up` downloads missing images automatically.

Auto-download images are downloaded and checksum-verified. Manual/setup-only images print setup instructions instead.

## Preview Cleanup

```bash
yeast images clean ubuntu-24.04 --dry-run
```

Use `--dry-run` first when you are not sure what will be removed.

## Remove One Cached Image

```bash
yeast images clean ubuntu-24.04
```

The next project that needs the image can pull it again.

## Remove All Cached Images

```bash
yeast images clean --all --dry-run
yeast images clean --all
```

## When To Clean The Cache

Clean cached images when:

- you need disk space
- an image download was interrupted
- a checksum verification failed and left a broken cache entry
- you want to force a fresh image download before the next `yeast up`

## What Not To Delete Manually

Prefer `yeast images clean` over manually deleting cache directories. Yeast can report what it removes and keeps the workflow predictable.
