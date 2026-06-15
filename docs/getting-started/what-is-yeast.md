# What Is Yeast?

Yeast is a command-line tool for running local Linux VMs as project files.

Instead of manually downloading cloud images, creating qcow2 disks, writing cloud-init data, composing QEMU commands, and remembering SSH ports, you describe the machines you want in `yeast.yaml`.

Then you run:

```bash
yeast up
```

Yeast creates the VM runtime files, starts QEMU/KVM, waits for SSH, and runs any configured provisioning.

## The Mental Model

```text
folder
├── yeast.yaml          desired VM configuration
└── .yeast/            project identity and state

~/.yeast/
├── cache/images/      shared trusted image cache
└── projects/          VM disks, runtime files, logs, snapshots
```

`yeast.yaml` is desired state. It says what should exist.

Yeast state is runtime reality. It records what is running, what SSH port was assigned, where disks live, and what snapshots exist.

## What You Can Do

- start one VM for quick Linux testing
- start multiple VMs in one project
- attach VMs to one private lab network
- provision packages, files, and shell commands
- run commands inside guests
- copy files in and out
- inspect VM state and logs
- snapshot stopped VMs and restore them later
- script workflows with JSON output

## Good Fit

Yeast is a good fit when you want:

- real Linux VMs, not containers
- local repeatable labs
- SSH-ready machines
- simple YAML configuration
- automation-friendly command output

## Not A Good Fit

Yeast is not currently meant for:

- Windows guests
- multiple private networks per project
- DHCP lab networks
- bridge-mode LAN exposure
- live migration
- production VM hosting

See [Limitations](../reference/limitations.md) for the full current scope.
