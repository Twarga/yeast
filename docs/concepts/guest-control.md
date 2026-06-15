# Guest Control

Guest control commands let you work with a running VM without remembering raw SSH arguments.

## Interactive SSH

```bash
yeast ssh web
```

## Run One Command

```bash
yeast exec web -- hostname
```

## Copy Files

```bash
yeast copy web --to-guest ./hello.txt /home/yeast/hello.txt
yeast copy web --from-guest /home/yeast/hello.txt ./hello-back.txt
```

## Logs And Inspect

```bash
yeast logs web --tail 80
yeast inspect web
```
