# Yeast Limits

Current shipped v1.0 scope:

- VM lifecycle
- project-safe paths
- provisioning packages, files, and shell workflows
- stopped-VM snapshots and restore
- one private multi-VM lab network
- guest `exec`, `copy`, `logs`, and `inspect`
- `yeast init --list-templates`
- `yeast init --template <name-or-path>`
- built-in starter templates
- local filesystem templates
- stable JSON envelopes and lifecycle events
- LabsBakery local-engine integration contract

Still not included:

- remote template downloads or registries
- complex template variables
- project-wide atomic reset
- bridge networking
- log streaming/follow mode
- service health checks
- daemon or web API
- Yeast MCP integration
- Twarga Cloud workers
- Windows or macOS host support
