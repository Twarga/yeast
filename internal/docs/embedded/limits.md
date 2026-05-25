# Yeast Limits

Current shipped scope through v0.6:

- VM lifecycle
- project-safe paths
- provisioning packages, files, and shell workflows
- stopped-VM snapshots and restore
- one private multi-VM lab network
- guest `exec`, `copy`, `logs`, and `inspect`

Planned first template scope for v0.7:

- `yeast init --list-templates`
- `yeast init --template <name-or-path>`
- built-in starter templates
- local filesystem templates

Still not included:

- remote template downloads or registries
- complex template variables
- project-wide atomic reset
- bridge networking
- log streaming/follow mode
- service health checks
- daemon or web API
- LabsBackery integration contract
- Yeast MCP integration
- Twarga Cloud workers
- Windows or macOS host support
