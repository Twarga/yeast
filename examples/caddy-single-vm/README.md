# caddy-single-vm

Single-VM provisioning and reset example for Yeast `v0.4`.

What this example does:

- boots one Ubuntu 24.04 VM
- installs `caddy`
- copies site files into the guest user home
- installs them into Caddy-owned paths during shell provisioning
- enables and restarts the `caddy` service
- supports a stopped-VM snapshot and restore loop after provisioning

What this example does not do:

- private networking
- multi-VM topologies
- guest exec/copy/logs beyond provisioning

## Files

- `yeast.yaml` - VM plus provisioning steps
- `site/index.html` - static page served by Caddy
- `site/Caddyfile` - minimal HTTP config

## Run

Create a fresh project directory first. `yeast init` creates the project metadata and starter config.

```bash
mkdir my-caddy-demo
cd my-caddy-demo
yeast init
cp /path/to/yeast/examples/caddy-single-vm/yeast.yaml ./yeast.yaml
mkdir -p site
cp /path/to/yeast/examples/caddy-single-vm/site/index.html ./site/index.html
cp /path/to/yeast/examples/caddy-single-vm/site/Caddyfile ./site/Caddyfile
```

Then run:

```bash
yeast doctor
yeast pull ubuntu-24.04
yeast up
yeast status
yeast ssh web
```

Expected result:

- `yeast up` finishes with the instance in `provisioned` state
- inside the guest, `curl http://127.0.0.1` returns the example HTML page

To rerun provisioning after editing files:

```bash
yeast provision web
```

## Reset Loop

Stop the VM before snapshot or restore. `v0.4` does not support live snapshots or live restore.

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
- privileged destination writes are handled through shell provisioning in `v0.3`
- snapshot create and restore are stopped-VM only in `v0.4`
- service verification is still manual in `v0.4`
