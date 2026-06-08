# ubuntu-basic

Minimal single-VM starter template for Yeast `v0.7`.

What this template does:

- starts one Ubuntu 24.04 VM
- uses the default Yeast SSH access flow
- keeps the scope to the basic lifecycle feature set

What this template does not do:

- post-boot provisioning
- private networking
- snapshots
- multi-VM labs
- guest control automation beyond normal `yeast ssh`

## Files

- `yeast.yaml` - one Ubuntu VM named `web`

## Run

```bash
mkdir my-ubuntu-basic
cd my-ubuntu-basic
yeast init --template ubuntu-basic
```

Then run the basic lifecycle flow:

```bash
yeast doctor
yeast pull ubuntu-24.04
yeast up
yeast status
yeast ssh web
```

To stop the VM:

```bash
yeast down
```

To remove the tracked runtime files for this project:

```bash
yeast destroy
```

## Notes

- this example requires a Linux host with KVM, QEMU, `qemu-img`, `ssh`, and `genisoimage` or `mkisofs`
- Yeast stores shared base images under `~/.yeast/cache/images`
- Yeast stores project runtime state under `~/.yeast/projects/<project-id>/`
- generated template projects are normal editable Yeast projects
