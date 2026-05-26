# Substrate

Shared world model for developer+agent pods. See the technical spec for architecture and milestones.

## Requirements

- Go 1.26+
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) for query codegen

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

## Build

```bash
go build -o substrate ./cmd/substrate
go test ./...
```

After changing SQL in `internal/db/query/` or `internal/db/migrations/`:

```bash
sqlc generate
```

## Quick start (M0–M1)

```bash
substrate init
substrate pod register --name solo-pod --owner me@example.com
substrate workstream add --pod solo-pod --title "Auth refactor" --intent "Migrate to OIDC" --branch feat/oidc
substrate gap add --pod solo-pod --category ambiguous_spec --description "Unclear spec" --priority 2
```

## MCP (stdio)

Point Cursor or Claude Code at:

```json
{
  "mcpServers": {
    "substrate": {
      "command": "/path/to/substrate",
      "args": ["mcp", "serve"]
    }
  }
}
```

Read tools: `substrate.query_active_work`, `substrate.get_workstream`, `substrate.list_capability_gaps`, `substrate.who_owns`, `substrate.get_pod_state`.

## Configuration

Default: `~/.substrate/config.yaml`. Override with `SUBSTRATE_CONFIG` or `--config`.

See [substrate.example.yaml](substrate.example.yaml).

## Dependencies (direct)

| Package                                  | Purpose                 |
| ---------------------------------------- | ----------------------- |
| `modernc.org/sqlite`                     | Pure-Go SQLite (no cgo) |
| `github.com/go-git/go-git/v5`            | Document repo           |
| `github.com/spf13/cobra`                 | CLI                     |
| `github.com/modelcontextprotocol/go-sdk` | MCP server              |
| `gopkg.in/yaml.v3`                       | Config                  |
| `github.com/google/uuid`                 | Entity IDs              |
| `github.com/stretchr/testify`            | Tests                   |

Air-gapped: `go mod vendor` before deploy.
