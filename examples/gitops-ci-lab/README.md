# gitops-ci-lab

A 3-VM GitOps / CI lab for Yeast.

## What it does

- `gitea` — Git server at http://127.0.0.1:3000
- `runner` — Drone CI server that builds on push
- `registry` — Docker registry at http://127.0.0.1:5000 for storing built images

## Quick start

```bash
mkdir my-gitops-lab && cd my-gitops-lab
yeast init
cp -r /path/to/yeast/examples/gitops-ci-lab/* ./
yeast pull ubuntu-24.04
yeast up
bash scripts/verify.sh
```

## Browse

- Gitea: http://127.0.0.1:3000 (admin/admin)
- Registry: http://127.0.0.1:5000/v2/

## Full tutorial

See [Tutorial 14: GitOps / CI Lab](../../tutorials/14-gitops-ci-lab.md).
