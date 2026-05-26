-- name: CreateGap :exec
INSERT INTO capability_gaps (
  id, pod_id, workstream_id, category, description, priority, status,
  resolution_ref, frequency, scope, occurred_at, resolved_at, metadata
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetGapByPodAndDescription :one
SELECT * FROM capability_gaps
WHERE pod_id = ? AND description = ?
LIMIT 1;

-- name: GetOpenGapByPodAndDescription :one
SELECT * FROM capability_gaps
WHERE pod_id = ? AND description = ? AND status = 'open'
LIMIT 1;

-- name: AcknowledgeGap :exec
UPDATE capability_gaps SET status = 'acknowledged' WHERE id = ?;

-- name: ResolveGap :exec
UPDATE capability_gaps
SET status = 'resolved', resolved_at = ?, resolution_ref = ?
WHERE id = ?;

-- name: IncrementGapFrequency :exec
UPDATE capability_gaps
SET frequency = frequency + 1, occurred_at = ?
WHERE id = ?;

-- name: GetGap :one
SELECT * FROM capability_gaps WHERE id = ? LIMIT 1;

-- name: ListGapsByPod :many
SELECT * FROM capability_gaps WHERE pod_id = ? ORDER BY priority ASC, occurred_at DESC;

-- name: ListOpenGapsByPod :many
SELECT * FROM capability_gaps
WHERE pod_id = ? AND status = 'open'
ORDER BY priority ASC, occurred_at DESC;

-- name: ListGapsAll :many
SELECT * FROM capability_gaps ORDER BY priority ASC, occurred_at DESC;
