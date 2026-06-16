# caddy-single-vm

Single-VM provisioning starter template for Yeast.

What this template does:

- boots one Ubuntu 24.04 VM
- installs `caddy`
- copies site files into the guest user home
- installs them into Caddy-owned paths during shell provisioning
- enables and restarts the `caddy` service
- leaves you with a normal project that can use snapshot/restore after provisioning

What this template does not do:

- private networking
- multi-VM topologies
- automatic snapshot creation
- hidden health checks
- MCP, cloud, or LabsBackery-specific behavior

## Files

- `yeast.yaml` - VM plus provisioning steps
- `site/index.html` - static page served by Caddy
- `site/Caddyfile` - minimal HTTP config

## Run

```bash
mkdir my-caddy-demo
cd my-caddy-demo
yeast init --template caddy-single-vm
```

Then run:

```bash
yeast doctor
yeast up
yeast status
yeast ssh web
```

`yeast up` downloads the Ubuntu image automatically if it is not cached yet.

Expected result:

- `yeast up` finishes with the instance in `provisioned` state
- inside the guest, `curl http://127.0.0.1` returns the example HTML page

To rerun provisioning after editing files:

```bash
yeast provision web
```

## Reset Loop

Stop the VM before snapshot or restore. Snapshot and restore operations only work on stopped VMs.

Create a clean baseline:

```bash
yeast down
yeast snapshot web clean --description "Provisioned Caddy baseline"
yeast snapshots web
```

Break the guest, then stop it again:

```bash
yeast up
yeast ssh web
sudo rm -f /var/www/html/index.html
exit
yeast down
```

Restore the clean baseline and boot again:

```bash
yeast restore web clean
yeast up
yeast ssh web
curl http://127.0.0.1
```

Expected result after restore:

- the Caddy site responds again from the restored disk state
- `yeast snapshots web` still lists `clean`

Delete the snapshot when you no longer need it:

```bash
yeast down
yeast delete-snapshot web clean
```

To stop or remove the VM:

```bash
yeast down
yeast destroy
```

## Notes

- this example assumes Ubuntu or Debian package management because package provisioning currently uses `apt-get`
- file sources are resolved relative to the project root
- privileged destination writes are handled through shell provisioning
- snapshot create and restore are stopped-VM only
- service verification is still manual
- generated template projects are normal editable Yeast projects
