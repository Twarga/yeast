---
title: Tutorial 14 - GitOps/CI Lab
description: Gitea + Drone CI pipeline with Docker registry
---

# Tutorial 14 - GitOps/CI Lab

This walkthrough demonstrates setting up a complete CI/CD pipeline with Gitea, Drone CI, and Docker registry.

## Create the project

```bash
mkdir 14-gitops-ci-lab
cd 14-gitops-ci-lab
yeast init
cp /path/to/yeast/examples/gitops-ci-lab/yeast.yaml ./yeast.yaml
```

## Boot and validate

```bash
yeast pull ubuntu-24.04
yeast up
yeast status
```

## Test the CI pipeline

```bash
yeast exec gitea -- curl -fsS http://localhost:3000
yeast exec drone -- curl -fsS http://localhost:80
```

## Cleanup

```bash
yeast down
yeast destroy
```

## What You Learned

- How to set up Gitea as a Git hosting platform
- How to configure Drone CI for continuous integration
- How to set up a Docker registry for artifact storage
- How to create a complete CI/CD pipeline

## Next Steps

- [Tutorials Index](./) - View all tutorials
