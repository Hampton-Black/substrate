-- schema_migrations: tracks applied migration versions
CREATE TABLE IF NOT EXISTS schema_migrations (
  version     INTEGER PRIMARY KEY,
  applied_at  TIMESTAMP NOT NULL
);

-- Pods: a developer+agent fleet
CREATE TABLE pods (
  id              TEXT PRIMARY KEY,
  display_name    TEXT NOT NULL,
  owner           TEXT NOT NULL,
  public_key      TEXT,
  created_at      TIMESTAMP NOT NULL,
  active          BOOLEAN NOT NULL DEFAULT 1,
  metadata        JSON
);

-- Workstreams: active threads of work within a pod
CREATE TABLE workstreams (
  id              TEXT PRIMARY KEY,
  pod_id          TEXT NOT NULL REFERENCES pods(id),
  title           TEXT NOT NULL,
  intent          TEXT,
  status          TEXT NOT NULL,
  spec_ref        TEXT,
  branch          TEXT,
  components      JSON,
  scope           TEXT NOT NULL DEFAULT 'team',
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
  category        TEXT NOT NULL,
  description     TEXT NOT NULL,
  priority        INTEGER NOT NULL DEFAULT 3,
  status          TEXT NOT NULL,
  resolution_ref  TEXT,
  frequency       INTEGER NOT NULL DEFAULT 1,
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
  event_type      TEXT NOT NULL,
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
  filter_expr     TEXT NOT NULL,
  delivery        TEXT NOT NULL,
  webhook_url     TEXT,
  active          BOOLEAN NOT NULL DEFAULT 1,
  created_at      TIMESTAMP NOT NULL
);

-- Indices for git-backed entities
CREATE TABLE spec_index (
  id              TEXT PRIMARY KEY,
  path            TEXT NOT NULL,
  current_version TEXT NOT NULL,
  status          TEXT NOT NULL,
  owner_pod       TEXT REFERENCES pods(id),
  scope           TEXT NOT NULL,
  components      JSON,
  dependents      JSON,
  last_updated    TIMESTAMP NOT NULL,
  metadata        JSON
);

CREATE TABLE decision_index (
  id              TEXT PRIMARY KEY,
  path            TEXT NOT NULL,
  title           TEXT NOT NULL,
  components      JSON,
  decided_by      TEXT NOT NULL,
  decided_at      TIMESTAMP NOT NULL,
  scope           TEXT NOT NULL,
  metadata        JSON
);

CREATE TABLE quality_gate_index (
  id              TEXT PRIMARY KEY,
  path            TEXT NOT NULL,
  scope           TEXT NOT NULL,
  target_pods     JSON,
  enabled         BOOLEAN NOT NULL DEFAULT 1,
  last_updated    TIMESTAMP NOT NULL
);

CREATE TABLE components (
  id              TEXT PRIMARY KEY,
  display_name    TEXT NOT NULL,
  description     TEXT,
  owner_pod       TEXT REFERENCES pods(id),
  paths           JSON,
  last_updated    TIMESTAMP NOT NULL,
  metadata        JSON
);
