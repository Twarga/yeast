# ubuntu-basic

Minimal single-VM example for Yeast `v0.3`.

What this example does:

- starts one Ubuntu 24.04 VM
- uses the default Yeast SSH access flow
- keeps the scope to the lifecycle-only feature set

What this example does not do yet:

- post-boot provisioning
- private networking
- snapshots
- multi-VM labs

## Files

- `yeast.yaml` — one Ubuntu VM named `web`

## Run

Create a fresh project directory first. `yeast init` creates the project metadata and starter config.

```bash
mkdir my-ubuntu-basic
cd my-ubuntu-basic
yeast init
cp /path/to/yeast/examples/ubuntu-basic/yeast.yaml ./yeast.yaml
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
- the example directory in the repo is a reference example, not a pre-initialized runnable project
