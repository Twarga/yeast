# Graphical & Desktop VMs

Yeast VMs are headless by default. Enable `display` to get a VNC endpoint —
optionally with GPU acceleration and UEFI firmware — plus framebuffer
screenshots and programmatic input through QEMU's monitor.

## yeast.yaml

```yaml
instances:
  - name: desk
    image: ubuntu-24.04
    memory: 4096
    cpus: 4
    firmware: uefi            # bios (default) | uefi
    display:
      enabled: true
      type: vnc               # vnc
      gpu: virtio-gpu         # virtio-vga | virtio-gpu | virgl
      width: 1920
      height: 1080
      host: 127.0.0.1         # VNC bind host
      port: 5900              # VNC bind port
```

`yeast up` prints the `vnc://` URL; `yeast status` shows it in the DISPLAY
column and the `display` JSON field.

## GPU modes

| `gpu` | What it does |
|---|---|
| `virtio-vga` | Basic paravirt display adapter — safest, works everywhere |
| `virtio-gpu` | virtio GPU device, 2D acceleration |
| `virgl` | VirGL 3D acceleration — needs `/dev/dri` + a virgl-capable QEMU; relaxes the seccomp `resourcecontrol` rule (only this mode does) |

`yeast doctor` reports whether `virgl` is actually available — it checks,
it does not guess.

## Screenshots — `yeast screenshot`

QMP `screendump` dumps the framebuffer to PNG (pure-Go PPM→PNG fallback):

```bash
yeast screenshot desk -o screen.png
```

Returns the path, dimensions, byte size, and SHA-256. Atomic `.part`→rename
writes mean a partial capture never lands in place.

## Programmatic input — `yeast input`

Send QEMU key events without VNC:

```bash
yeast input desk --key ctrl_l+alt_l+f3    # chord
yeast input desk --text 'root'            # printable ASCII + ret/tab/spc
```

The full printable-ASCII→QKeyCode table handles shift modifiers.

## WebSocket VNC via the daemon

`yeast daemon` proxies a browser-friendly WebSocket to the VM's VNC TCP
endpoint — point [noVNC](https://novnc.com) at
`GET /v1/projects/{id}/instances/{name}/vnc` (see
[Agents & MCP](agents-mcp.md)).

## Shared folders

`mounts:` share host directories into the guest via **virtiofs** (preferred;
one rootless `virtiofsd` sidecar per mount, auto-reaped on stop/destroy) or
**9p** fallback (`auto` picks virtiofs when available):

```yaml
mounts:
  - source: ./shared
    target: /mnt/shared
    driver: auto        # auto | virtiofs | 9p
    readonly: false
```

cloud-init `mounts:`/`bootcmd:` entries wire them inside the guest on first
boot.
