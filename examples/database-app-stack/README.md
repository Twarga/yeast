# database-app-stack

A 2-VM database + application stack for Yeast.

## What it does

- `db` — PostgreSQL with a `todo` database and seeded data
- `app` — Node.js Express API that reads/writes todos via PostgreSQL

Host accesses the API at `http://127.0.0.1:3000/todos`.

## Quick start

```bash
mkdir my-db-lab && cd my-db-lab
yeast init
cp -r /path/to/yeast/examples/database-app-stack/* ./
yeast pull ubuntu-24.04
yeast up
bash scripts/verify.sh
```

## Full tutorial

See [Tutorial 11: Database + App Stack](../../tutorials/11-database-app-stack.md).
