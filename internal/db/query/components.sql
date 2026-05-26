-- name: GetComponent :one
SELECT * FROM components WHERE id = ? LIMIT 1;
