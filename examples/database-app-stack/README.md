# database-app-stack

A 2-VM database + application stack for Yeast.

## What it does

- `db` — PostgreSQL with a `todo` database and seeded data
- `app` — Node.js Express API that reads/writes todos via PostgreSQL

Verify the API through `yeast exec app -- curl http://localhost:3000/todos`. Public host port mappings are not part of Yeast v1.1.

## Quick start

```bash
mkdir my-db-lab && cd my-db-lab
yeast init
cp -r /path/to/yeast/examples/database-app-stack/* ./
yeast up
bash scripts/verify.sh
```

`yeast up` downloads the Ubuntu image automatically if it is not cached yet.

## Note

This is an advanced example, not part of the beginner docs path yet.
