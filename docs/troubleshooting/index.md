# Troubleshooting

Start with:

```bash
yeast doctor
```

Then gather:

```bash
yeast status
yeast logs <instance> --tail 100
yeast inspect <instance>
```

For automation:

```bash
yeast status --json
yeast inspect <instance> --json
```

Use the topic pages for specific failures.
