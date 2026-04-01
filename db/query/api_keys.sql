-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, key_hash, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListAPIKeys :many
SELECT *
FROM api_keys
WHERE user_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAPIKeyByID :one
SELECT *
FROM api_keys
WHERE id = $1 AND user_id = $2;

-- name: GetActiveAPIKeyByHash :one
SELECT *
FROM api_keys
WHERE key_hash = $1 AND revoked_at IS NULL;

-- name: TouchAPIKeyLastUsed :exec
UPDATE api_keys
SET last_used_at = now()
WHERE id = $1;

-- name: RevokeAPIKey :one
UPDATE api_keys
SET revoked_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

