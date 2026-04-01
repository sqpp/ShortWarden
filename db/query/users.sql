-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT *
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DisableUser :one
UPDATE users
SET disabled_at = now()
WHERE id = $1
RETURNING *;

-- name: EnableUser :one
UPDATE users
SET disabled_at = NULL
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $2
WHERE id = $1
RETURNING *;

-- name: GetUserSettings :one
SELECT id, email, password_hash, created_at, disabled_at, redirect_delay_seconds, keep_expired_links, timezone
FROM users
WHERE id = $1;

-- name: UpdateUserSettings :one
UPDATE users
SET redirect_delay_seconds = $2,
    keep_expired_links = $3,
    timezone = $4
WHERE id = $1
RETURNING id, email, password_hash, created_at, disabled_at, redirect_delay_seconds, keep_expired_links, timezone;

