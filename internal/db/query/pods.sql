-- name: CreatePod :exec
INSERT INTO pods (id, display_name, owner, public_key, created_at, active, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetPod :one
SELECT * FROM pods WHERE id = ? LIMIT 1;

-- name: GetPodByDisplayName :one
SELECT * FROM pods WHERE display_name = ? LIMIT 1;

-- name: ListPods :many
SELECT * FROM pods ORDER BY created_at ASC;
