# Events

Use `--json --events` to stream lifecycle events as JSON Lines.

```bash
yeast up --json --events
```

Events are useful for tools that need progress updates while a workflow runs.

`--events` requires `--json`.
