# Lab 07: Templates And Reusable Labs

In this lab, you will start from built-in templates and inspect what they create.

You will learn:

- `yeast init --list-templates`
- `yeast init --template`
- how templates become normal editable projects
- when to use local templates

## List Templates

```bash
yeast init --list-templates
```

Built-in templates include:

- `ubuntu-basic`
- `caddy-single-vm`
- `two-vm-lab`

## Create A Template Project

```bash
mkdir yeast-lab-07
cd yeast-lab-07
yeast init --template caddy-single-vm
```

## Inspect The Files

```bash
find . -maxdepth 3 -type f | sort
sed -n '1,180p' yeast.yaml
```

After a template is copied, it is just a normal Yeast project. You can edit `yeast.yaml` and the copied files.

## Start And Clean Up

```bash
yeast up
yeast down
yeast destroy
```

## What You Learned

Templates are project starters. They do not hide state or create a special project type.

You finished the Yeast mini bootcamp.
