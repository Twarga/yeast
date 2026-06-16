# Images

Images are the base disks Yeast uses to create VM overlays.

List supported images:

```bash
yeast pull --list
```

Start a project with an auto-downloadable image:

```bash
yeast up
```

If the image is not cached yet, `yeast up` downloads it before booting the VM.

Pre-cache an image manually when you want the download to happen before `yeast up`:

```bash
yeast pull <image>
```

Show cached images:

```bash
yeast pull --cached
```

## Auto Versus Manual Images

Auto-download images have direct URLs and checksums in Yeast.

Manual images are listed by Yeast, but require you to download or prepare the qcow2 file yourself.

See the [image reference](../reference/images.md) for the current list.
