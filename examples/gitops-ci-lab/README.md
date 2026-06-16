# gitops-ci-lab

A 3-VM GitOps / CI lab for Yeast.

## What it does

- `gitea` — Git server inside the `gitea` VM on port `3000`
- `runner` — Drone CI server that builds on push
- `registry` — Docker registry inside the `registry` VM on port `5000` for storing built images

Public host port mappings are not part of Yeast v1.1, so verification uses `yeast exec` inside the relevant VM.

## Quick start

```bash
mkdir my-gitops-lab && cd my-gitops-lab
yeast init
cp -r /path/to/yeast/examples/gitops-ci-lab/* ./
yeast up
bash scripts/verify.sh
```

`yeast up` downloads the Ubuntu image automatically if it is not cached yet.

## Inspect

```bash
yeast exec gitea -- curl -fsS http://localhost:3000
yeast exec registry -- curl -fsS http://localhost:5000/v2/
```

## Note

This is an advanced example, not part of the beginner docs path yet.
