-- name: InsertEvent :exec
INSERT INTO events (pod_id, workstream_id, event_type, payload, occurred_at)
VALUES (?, ?, ?, ?, ?);
