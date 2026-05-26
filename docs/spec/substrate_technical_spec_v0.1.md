# Substrate — Technical Spec v0.1

**Status:** Draft
**Working codename:** Substrate (rename TBD)
**Target stack:** Go, SQLite, Git, HTMX
**Deployment:** Single binary, air-gap capable

---

## Overview

Substrate is a shared world model for developer+agent pods. It provides a queryable, MCP-native system of record for active work, specs, decisions, and capability gaps. Agents publish state to Substrate as a side effect of doing work (commits, spec edits, decisions). Other agents query Substrate for context before acting. A synthesis layer produces digests, detects cross-pod conflicts, and prioritizes capability gaps.

The design goal is to eat the ceremonies that exist to transfer state between humans — daily standups, status reports, manual backlog grooming, ad-hoc "is anyone else touching this" conversations — while preserving the ceremonies that exist for judgment, intent alignment, and human learning.

---

## Goals

- **Harness-agnostic.** Works with Claude Code, OpenCode, Codex, Cursor, or any MCP-capable harness.
- **Auto-maintained.** Agents publish state during normal work. No human writes status updates.
- **Air-gap capable.** Runs entirely offline. LLM backend is configurable (Claude API, GenAI.mil, local).
- **Single binary.** No external infrastructure. SQLite + git on disk.
- **Capability gaps as first-class.** When an agent gets blocked, the block is logged, prioritized, and surfaced as roadmap input.
- **Diffable documents.** Specs and decisions live in git so they can be reviewed via PR and have provenance.

## Non-Goals (v0.1)

- Not a replacement for version control (lives alongside git).
- Not a ticketing or PM tool replacement (can integrate, not replace).
- Not a chat or messaging system.
- Not a code review tool (gates handle automated review; humans still review PRs).
- No multi-region / HA story. Single-instance is fine for v0.1.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Substrate Binary                          │
│                                                                  │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────┐   │
│  │  MCP Server  │   │  HTTP Server │   │  Synthesis       │   │
│  │  (stdio +    │   │  (UI + API)  │   │  Worker          │   │
│  │   HTTP/SSE)  │   │              │   │  (scheduled jobs)│   │
│  └──────┬───────┘   └──────┬───────┘   └──────┬───────────┘   │
│         │                  │                   │                │
│         └──────────────────┼───────────────────┘                │
│                            │                                    │
│                    ┌───────▼────────┐                           │
│                    │  Core Service  │                           │
│                    │  (entity ops,  │                           │
│                    │  ACL, events)  │                           │
│                    └───────┬────────┘                           │
│                            │                                    │
│              ┌─────────────┼──────────────┐                     │
│              │             │              │                     │
│      ┌───────▼──────┐  ┌───▼─────┐  ┌────▼────────┐           │
│      │   SQLite     │  │  go-git │  │  LLM Client │           │
│      │   (state)    │  │ (docs)  │  │  (pluggable)│           │
│      └──────────────┘  └─────────┘  └─────────────┘           │
└─────────────────────────────────────────────────────────────────┘
                            │
                ┌───────────┼────────────┐
                ▼           ▼            ▼
        ┌─────────────┐ ┌────────┐ ┌──────────┐
        │ Agent       │ │ Web UI │ │ Git Hooks│
        │ Harnesses   │ │ (HTMX) │ │          │
        │ (via MCP)   │ │        │ │          │
        └─────────────┘ └────────┘ └──────────┘
```

**Components:**

- **MCP Server** — exposes read/write/subscribe tools to agent harnesses. Supports stdio (local), HTTP, and SSE.
- **HTTP Server** — serves the HTMX UI and a JSON API for tooling.
- **Synthesis Worker** — runs scheduled jobs (digests, conflict detection, gap prioritization). Calls the configured LLM backend.
- **Core Service** — entity CRUD, ACL enforcement, event sourcing. Single source of truth in-process.
- **SQLite** — state storage (pods, workstreams, gaps, events, subscriptions, indices).
- **go-git** — document storage (specs, decisions, gates) as files in a git repo.
- **LLM Client** — pluggable backend (Claude, GenAI.mil, OpenAI-compatible, local).
- **Git Hooks** — pre-built post-commit/post-push hooks that auto-publish workstream state.

---

## Data Model

### SQLite schema

```sql
-- Pods: a developer+agent fleet
CREATE TABLE pods (
  id              TEXT PRIMARY KEY,           -- e.g., "hampton-pod"
  display_name    TEXT NOT NULL,
  owner           TEXT NOT NULL,              -- human identity
  public_key      TEXT,                       -- ed25519, optional in v0.1
  created_at      TIMESTAMP NOT NULL,
  active          BOOLEAN NOT NULL DEFAULT 1,
  metadata        JSON
);

-- Workstreams: active threads of work within a pod
CREATE TABLE workstreams (
  id              TEXT PRIMARY KEY,           -- UUID
  pod_id          TEXT NOT NULL REFERENCES pods(id),
  title           TEXT NOT NULL,
  intent          TEXT,                       -- human or agent-summarized goal
  status          TEXT NOT NULL,              -- planning|active|blocked|review|merged|abandoned
  spec_ref        TEXT,                       -- git path to current spec
  branch          TEXT,
  components      JSON,                       -- array of component IDs touched
  scope           TEXT NOT NULL DEFAULT 'team', -- pod|team|org|public
  last_activity   TIMESTAMP NOT NULL,
  created_at      TIMESTAMP NOT NULL,
  metadata        JSON
);
CREATE INDEX idx_workstreams_pod ON workstreams(pod_id);
CREATE INDEX idx_workstreams_status ON workstreams(status);
CREATE INDEX idx_workstreams_activity ON workstreams(last_activity);

-- Capability gaps: anything that blocked an agent
CREATE TABLE capability_gaps (
  id              TEXT PRIMARY KEY,
  pod_id          TEXT NOT NULL REFERENCES pods(id),
  workstream_id   TEXT REFERENCES workstreams(id),
  category        TEXT NOT NULL,              -- missing_tool|missing_skill|ambiguous_spec|missing_context|env_constraint|other
  description     TEXT NOT NULL,
  priority        INTEGER NOT NULL DEFAULT 3, -- 1=high ... 5=low
  status          TEXT NOT NULL,              -- open|acknowledged|in_progress|resolved|wontfix
  resolution_ref  TEXT,                       -- pointer to spec/PR/decision
  frequency       INTEGER NOT NULL DEFAULT 1, -- times this gap has been hit
  scope           TEXT NOT NULL DEFAULT 'pod',
  occurred_at     TIMESTAMP NOT NULL,
  resolved_at     TIMESTAMP,
  metadata        JSON
);
CREATE INDEX idx_gaps_status ON capability_gaps(status);
CREATE INDEX idx_gaps_priority ON capability_gaps(priority);

-- Events: append-only log of everything
CREATE TABLE events (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  pod_id          TEXT REFERENCES pods(id),
  workstream_id   TEXT,
  event_type      TEXT NOT NULL,              -- workstream.published|gap.registered|spec.proposed|...
  payload         JSON NOT NULL,
  occurred_at     TIMESTAMP NOT NULL,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_occurred ON events(occurred_at);

-- Subscriptions: who wants to be notified about what
CREATE TABLE subscriptions (
  id              TEXT PRIMARY KEY,
  pod_id          TEXT NOT NULL REFERENCES pods(id),
  filter_expr     TEXT NOT NULL,              -- JSON filter or CEL-style expression
  delivery        TEXT NOT NULL,              -- digest|immediate|webhook
  webhook_url     TEXT,
  active          BOOLEAN NOT NULL DEFAULT 1,
  created_at      TIMESTAMP NOT NULL
);

-- Indices for git-backed entities
CREATE TABLE spec_index (
  id              TEXT PRIMARY KEY,
  path            TEXT NOT NULL,              -- git path
  current_version TEXT NOT NULL,              -- semver or git sha
  status          TEXT NOT NULL,              -- draft|proposed|ratified|deprecated
  owner_pod       TEXT REFERENCES pods(id),
  scope           TEXT NOT NULL,
  components      JSON,                       -- components this spec governs
  dependents      JSON,                       -- pod IDs that depend on this spec
  last_updated    TIMESTAMP NOT NULL,
  metadata        JSON
);

CREATE TABLE decision_index (
  id              TEXT PRIMARY KEY,
  path            TEXT NOT NULL,
  title           TEXT NOT NULL,
  components      JSON,
  decided_by      TEXT NOT NULL,              -- pod or human ID
  decided_at      TIMESTAMP NOT NULL,
  scope           TEXT NOT NULL,
  metadata        JSON
);

CREATE TABLE quality_gate_index (
  id              TEXT PRIMARY KEY,
  path            TEXT NOT NULL,
  scope           TEXT NOT NULL,              -- pod|team|org
  target_pods     JSON,                       -- nullable; null = all pods in scope
  enabled         BOOLEAN NOT NULL DEFAULT 1,
  last_updated    TIMESTAMP NOT NULL
);

CREATE TABLE components (
  id              TEXT PRIMARY KEY,           -- e.g., "auth-module"
  display_name    TEXT NOT NULL,
  description     TEXT,
  owner_pod       TEXT REFERENCES pods(id),
  paths           JSON,                       -- code paths/globs that constitute this component
  last_updated    TIMESTAMP NOT NULL,
  metadata        JSON
);
```

### Git-backed document layout

```
substrate-repo/
├── specs/
│   ├── auth-service-v3.md
│   └── billing-flow-v1.md
├── decisions/
│   └── 2026-05-15-auth-provider-choice.md
├── gates/
│   ├── team-architecture.yaml
│   └── pod-hampton.yaml
├── components/
│   ├── auth-module.md
│   └── billing-flow.md
└── README.md
```

**Spec document format** (markdown with YAML frontmatter):

```markdown
---
id: auth-service-v3
version: 3.0.0
status: ratified
owner_pod: platform-pod
scope: team
components: [auth-module, session-store]
dependents: [hampton-pod, jane-pod]
ratified_by: jane-pod
ratified_at: 2026-05-20T14:32:00Z
---

# Auth Service v3

## Intent
[What this spec exists to accomplish]

## Contract
[The API/behavior other pods can rely on]

## Implementation notes
[Constraints, non-obvious choices]
```

**Decision document format:**

```markdown
---
id: 2026-05-15-auth-provider-choice
title: Use OIDC provider X for service-to-service auth
decided_by: platform-pod
decided_at: 2026-05-15T10:00:00Z
components: [auth-module]
scope: team
supersedes: []
---

## Context
[What problem prompted the decision]

## Decision
[What we chose]

## Alternatives considered
[What we didn't choose, and why]

## Consequences
[Expected and accepted tradeoffs]
```

**Quality gate format** (YAML, machine-readable):

```yaml
id: team-architecture
scope: team
target_pods: null  # null = all pods
enabled: true
gates:
  - name: no_new_globals
    type: lint
    command: "rg -n '^var [A-Z]' --type go"
    expect_exit: 1
  - name: contract_tests_pass
    type: test
    command: "go test ./contracts/..."
    expect_exit: 0
  - name: spec_referenced
    type: custom
    command: "substrate verify-spec-ref"
    expect_exit: 0
```

---

## MCP Tool Surface

All tools follow `substrate.<verb>_<noun>` naming. Inputs and outputs are JSON. Each tool enforces ACLs based on calling pod identity.

### Read tools

| Tool | Purpose |
|---|---|
| `substrate.query_active_work` | List active workstreams. Params: `scope?`, `pod_id?`, `component?`, `status?`. Returns: `Workstream[]`. |
| `substrate.get_workstream` | Get a single workstream by ID with full detail. |
| `substrate.get_spec` | Retrieve a spec document. Params: `spec_id`, `version?`. Returns: rendered markdown + metadata. |
| `substrate.list_specs` | List specs. Params: `status?`, `component?`, `owner_pod?`. |
| `substrate.find_overlap` | Detect overlap. Params: `components: string[]`, `change_description?: string`. Returns: list of pods/workstreams touching the same area, with a synthesis-generated risk note. |
| `substrate.who_owns` | Resolve a component to its owner pod. Params: `component`. |
| `substrate.list_capability_gaps` | List gaps. Params: `status?`, `category?`, `pod_id?`, `priority_max?`. |
| `substrate.get_decision_log` | Retrieve decisions. Params: `component?`, `since?`, `pod_id?`. |
| `substrate.get_quality_gates` | List gates that apply. Params: `pod_id`. Returns: composed gate set from all applicable scopes. |
| `substrate.get_pod_state` | Snapshot of a pod: active workstreams, open gaps, recent decisions. |
| `substrate.get_digest` | Retrieve the most recent synthesized digest for a pod. |

### Write tools

| Tool | Purpose |
|---|---|
| `substrate.publish_workstream_state` | Create or update a workstream. Usually fired automatically by git hooks; agents may call directly to add intent/notes. |
| `substrate.register_capability_gap` | Log a gap. Agents call this when they get blocked. Substrate dedupes by description similarity. |
| `substrate.publish_decision` | Log an architectural/product decision. Writes a decision doc to git and indexes it. |
| `substrate.propose_spec` | Submit a new spec or new version. Status starts as `proposed`. |
| `substrate.ratify_spec` | Mark a spec as `ratified`. Requires caller to be authorized for the scope. |
| `substrate.deprecate_spec` | Mark a spec as `deprecated`. Requires `supersedes` reference. |
| `substrate.add_quality_gate` | Define a new gate at a scope. |
| `substrate.request_review` | Request review from another pod or human. Generates a notification and a review-pending event. |

### Subscribe tools

| Tool | Purpose |
|---|---|
| `substrate.subscribe` | Register a filter. Params: `filter_expr`, `delivery: digest\|immediate\|webhook`. |
| `substrate.unsubscribe` | Remove a subscription. |
| `substrate.poll_notifications` | Pull pending notifications for the calling pod. |

### Go type sketches

```go
type Workstream struct {
    ID           string            `json:"id"`
    PodID        string            `json:"pod_id"`
    Title        string            `json:"title"`
    Intent       string            `json:"intent,omitempty"`
    Status       WorkstreamStatus  `json:"status"`
    SpecRef      string            `json:"spec_ref,omitempty"`
    Branch       string            `json:"branch,omitempty"`
    Components   []string          `json:"components"`
    Scope        Scope             `json:"scope"`
    LastActivity time.Time         `json:"last_activity"`
    CreatedAt    time.Time         `json:"created_at"`
    Metadata     map[string]any    `json:"metadata,omitempty"`
}

type CapabilityGap struct {
    ID            string         `json:"id"`
    PodID         string         `json:"pod_id"`
    WorkstreamID  string         `json:"workstream_id,omitempty"`
    Category      GapCategory    `json:"category"`
    Description   string         `json:"description"`
    Priority      int            `json:"priority"`
    Status        GapStatus      `json:"status"`
    ResolutionRef string         `json:"resolution_ref,omitempty"`
    Frequency     int            `json:"frequency"`
    OccurredAt    time.Time      `json:"occurred_at"`
    ResolvedAt    *time.Time     `json:"resolved_at,omitempty"`
}

type OverlapReport struct {
    Component     string             `json:"component"`
    Overlaps      []OverlapEntry     `json:"overlaps"`
    RiskNote      string             `json:"risk_note"`  // synthesis-generated
}

type OverlapEntry struct {
    PodID         string    `json:"pod_id"`
    WorkstreamID  string    `json:"workstream_id"`
    WorkstreamTitle string  `json:"workstream_title"`
    LastActivity  time.Time `json:"last_activity"`
}
```

---

## Storage Strategy

**Hybrid by design.** Two storage layers, each chosen to match the access pattern of its data.

### SQLite (state)

- High-frequency reads and writes
- Workstreams, gaps, events, subscriptions, indices
- Single-file embedded DB at `~/.substrate/substrate.db`
- Driver: `modernc.org/sqlite` (pure-Go, cross-compiles cleanly, no cgo)
- WAL mode enabled for concurrent reads during writes

### Git (documents)

- Specs, decisions, quality gates, component definitions
- Diffable, blameable, reviewable via PR
- Library: `github.com/go-git/go-git/v5` for in-process operations
- Two deployment modes:
  - **Dedicated repo** (default): Substrate manages its own repo
  - **Colocated**: a `substrate/` directory inside the project repo

When a write tool modifies a git-backed entity, Substrate stages, commits, and (if remote configured) pushes. The commit message follows a structured format so it's machine-parseable:

```
substrate: ratify spec auth-service-v3

Pod: platform-pod
Type: spec.ratified
Spec: auth-service-v3
Version: 3.0.0
```

---

## Synthesis Layer

A separate worker (in-binary goroutines for v0.1; could split to a separate process later) that runs scheduled jobs.

### Jobs

| Job | Cadence | Output |
|---|---|---|
| Morning digest | Daily, configurable per pod | A summary doc covering: yesterday's deltas, today's open gaps for this pod, cross-pod overlaps, decisions needing input. Stored as an event; surfaced in UI and via `get_digest`. |
| Conflict detection | Every 15 min (configurable) | Detects when multiple pods touch the same components. Generates `overlap_alert` events. |
| Capability gap reprioritization | Daily | Re-scores gaps by recency, frequency, and impact. Updates `priority` field. |
| Stale workstream check | Daily | Flags workstreams with no activity for N days; emits `workstream_stale` event. |
| Spec ratification reminder | Weekly | Surfaces proposed specs awaiting ratification. |

### LLM backend

Pluggable. Config-driven:

```yaml
synthesis:
  backend: claude_api  # claude_api | genai_mil | openai_compatible | local
  model: claude-opus-4-7
  endpoint: ""  # used for genai_mil and openai_compatible
  api_key_env: ANTHROPIC_API_KEY  # env var name, never the key itself
  max_tokens: 4000
  timeout_seconds: 60
```

For air-gapped deployments, `genai_mil` or `local` is used. The backend interface is small:

```go
type LLMBackend interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
```

---

## Identity, Scope, and Audit

### Pod identity

- Pods register on first run via `substrate pod register --name hampton-pod --owner hampton@blackcape.io`
- Substrate generates an ed25519 keypair and stores the public key
- The private key lives in the pod's local config (`~/.substrate/pod-key`)
- In v0.1, signing is optional; in v0.2, all writes will be signed

### Scopes

Every entity has a `scope` field:

- `pod` — visible only to the owning pod
- `team` — visible to all pods in the same team (configurable group)
- `org` — visible to all registered pods
- `public` — exportable

ACL enforcement happens at the Core Service layer before storage access.

### Quality gate targeting

Gates have a `scope` and optional `target_pods`. A pod's effective gate set is the union of all gates whose scope and targeting apply to it. Conflicts are resolved by precedence: `pod > team > org`.

### Audit

Every state change emits an event with `pod_id`, `event_type`, `payload`, `occurred_at`. The events table is append-only. For air-gapped/government deployments, an `audit export` command produces a standard-format log (initially CSV; CEF/LEEF if needed later).

---

## Human UI

HTMX + Tailwind, served by the same binary at `http://localhost:7777` by default.

### Views

- **Dashboard** — current state across all visible pods. Active workstreams, recent gaps, recent decisions.
- **Workstream view** — single workstream detail: linked spec, components, gaps encountered, recent events.
- **Capability gap board** — kanban-style: open / acknowledged / in progress / resolved. Filterable by pod, category, priority.
- **Decision log** — timeline of decisions, filterable by component or pod.
- **Spec viewer** — rendered markdown with status badge, ratification history, dependents.
- **Pod profile** — for a given pod: active work, owned components, recent activity, gap stats.
- **Digest viewer** — yesterday's, today's morning digests per pod.

No JS framework. HTMX for interactivity, server-rendered templates (`html/template`).

---

## External Interfaces

The MCP interface is the substrate's contract, and the HTMX UI is one consumer of it. Other consumers can be built without modifying the substrate itself. The most useful near-term external interface is a **persistent team-agent** — a chat-channel-resident process that brings substrate state into the team's existing communication tools.

### Architectural framing

Two planes:

- **Substrate (data plane):** state, MCP interface, persistence, ACLs, synthesis jobs.
- **Team-agent (human-interface plane):** lives in the team's chat (Slack, Mattermost, Discord), speaks the substrate's MCP, brings notifications and conversational queries to where human attention already lives.

The team-agent is just another MCP client of the substrate — the same architectural role as a developer's coding harness, with conversation and coordination as its job instead of code execution. The HTMX UI handles deliberate "let me check the dashboard" moments; the team-agent solves the harder problem of "no one ever checks the dashboard."

### Reference implementation: Hermes (Nous Research)

[Hermes Agent](https://hermes-agent.nousresearch.com/) is open-source (MIT) and fits this role out of the box. Relevant features:

- Multi-platform presence (Slack, Mattermost, Discord, Telegram, Signal, Email, CLI)
- Persistent memory and auto-generated skills
- Configurable LLM backend (works air-gapped with local or GenAI.mil endpoints)
- Natural-language scheduled automations
- Sandboxed subagent delegation

A Substrate + Hermes deployment wires Hermes to the substrate via MCP and configures Hermes to:

- Post morning digests to designated team channels
- Respond to natural-language queries: *"is anyone working on auth this week?"*, *"what gaps are blocking Jane's pod?"*, *"what specs are awaiting ratification?"*
- Surface overlap alerts and PR review requests to the appropriate person
- Accept natural-language inputs and translate them into substrate writes: *"Hermes, remind me to ratify auth-service-v3 by Friday"* → creates a subscription + scheduled reminder

### What this changes for v0.1

Nothing in the substrate itself. The MCP interface and ACL model are designed to support an external agent as a peer pod. The team-agent is treated as a special pod with `scope: team` whose role is coordination rather than code production. It registers like any other pod (`substrate pod register --name team-coordinator --kind team-agent`) and is granted appropriate read/write scopes.

### Future considerations (post-v0.1)

- Whether to ship a recommended Hermes config / skill pack alongside the substrate
- Whether team-agents should be allowed to ratify specs autonomously (default: no)
- Channel routing for multi-team deployments
- A "minimum viable team-agent" reference implementation we ship ourselves, for teams that want something lighter than Hermes

### Other consumers

Anything that speaks MCP can be a consumer. Examples worth keeping in mind during design review:

- A CLI tool for one-off queries (`substrate query "active work in auth-module"`)
- IDE extensions that show substrate state in a side panel
- CI/CD integrations that consult substrate before running expensive builds
- An exporter that produces compliance reports from the audit log

---

## Tech Stack

| Concern | Choice |
|---|---|
| Language | Go 1.22+ |
| Storage (state) | `modernc.org/sqlite` |
| Storage (docs) | `github.com/go-git/go-git/v5` |
| MCP server | Anthropic Go MCP SDK |
| HTTP router | stdlib `net/http` + `github.com/go-chi/chi/v5` |
| Templating | `html/template` |
| Frontend | HTMX + Tailwind (CDN or vendored) |
| Config | YAML via `gopkg.in/yaml.v3` |
| CLI | `github.com/spf13/cobra` |
| Logging | `log/slog` |
| Testing | stdlib `testing` + `github.com/stretchr/testify` |

Single binary, cross-compiled for Linux, macOS, and Windows. No cgo dependencies.

---

## Milestones

### M0 — Foundation (week 1)
- Project skeleton, module layout, CI
- SQLite schema setup with migrations
- Git repo init via go-git
- `substrate init`, `substrate pod register` CLI commands
- Config loading

### M1 — Core MCP read surface (week 2)
- MCP server scaffolding (stdio + HTTP/SSE)
- Read tools: `query_active_work`, `get_workstream`, `list_capability_gaps`, `who_owns`, `get_pod_state`
- Manual workstream entry via CLI (for testing before git hooks)
- Integration test against Claude Code

### M2 — Auto-publish + capability gaps (week 3)
- `publish_workstream_state`, `register_capability_gap` write tools
- Git hook templates (post-commit, post-push) that publish workstream state
- Capability gap dedup logic
- Gap lifecycle (open → acknowledged → resolved)

### M3 — Web UI (week 4)
- HTMX dashboard, workstream view, gap board
- Read-only in this milestone (writes happen via MCP)
- Solo-usable end-to-end at this point

### M4 — Synthesis layer (weeks 5-6)
- Scheduled job framework
- Morning digest job
- Conflict detection job
- LLM backend interface + Claude API implementation
- GenAI.mil and local backends

### M5 — Multi-pod, ACLs, identity (week 7)
- Ed25519 pod keypairs
- Scope enforcement in Core Service
- Team membership config
- Audit export command

### M6 — Specs and decisions (weeks 8-9)
- `propose_spec`, `ratify_spec`, `deprecate_spec`
- Decision log tools
- Quality gate definition and enforcement
- Spec viewer in UI

### M7 — Polish + production (weeks 10-12)
- Documentation (user, ops, MCP tool reference)
- Backup/restore commands
- Performance pass
- Deployment guides (single-host, air-gapped, etc.)
- Onboarding docs for new pods

---

## MVP Slice

The smallest version that produces real value, suitable for solo dogfooding:

- **M0 + M1 + minimal M2** (write tools: `publish_workstream_state`, `register_capability_gap`) + **minimal M3** (read-only dashboard with workstreams and gaps)
- Target: ~3-4 weeks of focused work
- At the end you can: run Substrate locally, register a pod for yourself, have your agents register gaps as they hit them, see active work and gaps in a web UI

This is enough to test the core hypothesis (does auto-published workstream state + capability gap logging change how you work?) before investing in the synthesis layer or multi-pod features.

---

## Open Questions

- **Spec ratification authority.** Who can ratify a team-scoped spec? Default to "any pod owner" or require explicit ratifier list per spec? Probably the latter; needs a small ACL design.
- **Conflict detection sensitivity.** How aggressive should overlap detection be? File-level granularity or component-level? Starting at component-level seems right.
- **Capability gap deduplication.** Similarity-based (embedding match) or exact-match? Exact-match for v0.1, embedding-based when synthesis is in place.
- **Schema evolution.** How do we version Substrate's own SQLite schema and document formats? Standard migration table + semver on document frontmatter.
- **Cross-pod review delivery.** Async pull (poll_notifications) only for v0.1; consider webhooks or local notifications later.
- **Substrate-of-substrates.** When an org has multiple teams each running a Substrate, is there a federation story? Out of scope for v0.1.

---

## Implementation Notes

- The MCP server should accept both stdio (for local harness use) and HTTP/SSE (for remote / hosted harness use). Start with stdio.
- Git operations should be fast — batch writes and amend commits where possible to avoid noisy history.
- The synthesis worker should be skippable in dev (`--no-synthesis` flag) so the MVP can be tested without LLM access.
- All ACL checks should run at a single chokepoint in the Core Service, not scattered through tool handlers.
- Use structured logging with `pod_id` and `workstream_id` as standard fields for easier debugging.

---

*Spec author: Hampton. Generated as a v0.1 draft for handoff to Claude Code or Cursor.*
