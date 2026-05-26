# Agent guide — Substrate

Instructions for AI agents (Cursor, Claude Code, etc.) working in this repository.

## Source of truth

- **Product design:** [docs/spec/substrate_technical_spec_v0.1.md](docs/spec/substrate_technical_spec_v0.1.md) — schema, MCP tools, milestones, data model.
- When the spec and code disagree, follow the spec and leave a `// TODO(spec): …` comment if you cannot resolve it in scope.
- **Human onboarding:** [README.md](README.md) — build, quick start, MCP config.

## Shipped vs out of scope

**Done (do not re-implement unless fixing bugs):**

- **M0** — SQLite migrations, config, go-git doc repo, `substrate init`, `substrate pod register|list`
- **M1** — Stdio MCP read tools, workstream/gap CLI, `core.Service` tests

**Do not start unless the task explicitly asks:**

- M2 — write MCP tools, git hooks, gap lifecycle
- M3 — HTMX web UI
- M4 — synthesis worker / LLM backends
- M5 — ed25519, real ACLs, audit export
- M6 — specs, decisions, quality gates, component population
- M7 — docs polish, backup/restore, deployment guides

## Stack (non-negotiable)

| Concern | Choice |
|--------|--------|
| Language | Go **1.26+** (`go.mod`) |
| SQLite | `modernc.org/sqlite` only — **no cgo**, no `mattn/go-sqlite3` |
| Queries | **sqlc** — SQL in `internal/db/query/`, generated code in `internal/db/sqlc/` (committed) |
| Git | `github.com/go-git/go-git/v5` — never `exec.Command("git", …)` |
| CLI | `github.com/spf13/cobra` |
| MCP | `github.com/modelcontextprotocol/go-sdk/mcp` — **stdio first** (`substrate mcp serve`) |
| HTTP/UI | chi + `html/template` later — not in M0–M1 |
| Logging | `log/slog` with `pod_id` / `workstream_id` where applicable |
| Tests | stdlib `testing` + `github.com/stretchr/testify` |

Module path: `github.com/Hampton-Black/substrate`

Binary must cross-compile for Linux, macOS, and Windows without cgo.

## Architecture invariants

```
CLI / MCP handlers  →  core.Service (ACL chokepoint)  →  sqlc + dynamic db helpers
                              ↓
                         SQLite (WAL)          go-git (documents)
```

- **All permission checks** go through `core.Service` (`checkScope` today is a stub; real ACLs are M5).
- **MCP and CLI** call service methods; they must not import `internal/db/sqlc` or use `database/sql` directly.
- **State changes** emit rows on the `events` table via `InsertEvent`.
- **Gap dedup (v0.1):** exact match on `pod_id` + `description` → increment `frequency`, do not insert a duplicate row.
- **Synthesis worker** is config-only until M4 — do not start background LLM jobs from the binary.
- **Single binary** — no separate daemon.

## Repository layout

```
cmd/substrate/          # main
internal/
  cli/                  # cobra commands
  config/               # YAML loader
  core/                 # domain types + Service
  db/
    migrations/         # embedded SQL (schema source for sqlc too)
    query/              # hand-written sqlc queries
    sqlc/               # generated — run sqlc generate, commit output
    query_dynamic.go    # optional filters (sqlc: too dynamic)
  git/                  # go-git wrapper
  mcp/                  # MCP server + read tools
web/                    # empty until M3
docs/spec/              # technical spec versions
```

## Common workflows

**After changing** `internal/db/migrations/*.sql` or `internal/db/query/*.sql`:

```bash
sqlc generate
go build ./...
go vet ./...
go test ./...
```

Keep migrations and sqlc schema in sync (both read `internal/db/migrations/`).

**Verify locally:**

```bash
go build -o substrate ./cmd/substrate
substrate init
substrate pod register --name solo-pod --owner me@example.com
```

**MCP read tools (M1):** `substrate.query_active_work`, `substrate.get_workstream`, `substrate.list_capability_gaps`, `substrate.who_owns`, `substrate.get_pod_state`.

## Coding conventions

- Minimize scope; match existing style; no premature abstractions.
- Pod IDs: slugified from `--name` (kebab-case). Workstream/gap IDs: UUIDs.
- Comments only for non-obvious business logic or `TODO(spec)` questions.
- Prefer meaningful `core` unit tests over trivial CLI/MCP tests.
- Conservative defaults when the spec is silent; document uncertainty with `TODO(spec)`.

## Git commits

- Format: `substrate(scope): description` (e.g. `substrate(db): add gap queries`).
- **Signed commits** require the maintainer’s local GPG passphrase — agents cannot complete signed commits interactively. Stage changes and suggest commit messages; the human runs `git commit` in their terminal when signing is required.

## Air-gapped / vendor

Dependencies are pure Go. For offline builds: `go mod vendor` before transfer; mirror direct deps listed in README.
