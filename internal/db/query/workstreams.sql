-- name: CreateWorkstream :exec
INSERT INTO workstreams (
  id, pod_id, title, intent, status, spec_ref, branch, components,
  scope, last_activity, created_at, metadata
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetWorkstream :one
SELECT * FROM workstreams WHERE id = ? LIMIT 1;

-- name: ListWorkstreamsByPod :many
SELECT * FROM workstreams WHERE pod_id = ? ORDER BY last_activity DESC;

-- name: ListActiveWorkstreamsByPod :many
SELECT * FROM workstreams
WHERE pod_id = ? AND status IN ('planning', 'active', 'blocked', 'review')
ORDER BY last_activity DESC;

-- name: UpdateWorkstreamStatus :exec
UPDATE workstreams SET status = ?, last_activity = ? WHERE id = ?;

-- name: TouchWorkstreamActivity :exec
UPDATE workstreams SET last_activity = ? WHERE id = ?;
