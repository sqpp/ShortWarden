-- name: CreateDomain :one
INSERT INTO domains (user_id, hostname, dns_token)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListDomains :many
SELECT *
FROM domains
WHERE user_id = $1
ORDER BY is_primary DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateDomainDefaultTags :one
UPDATE domains
SET default_tags = $3
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: GetDomainByID :one
SELECT *
FROM domains
WHERE id = $1 AND user_id = $2;

-- name: GetDomainByHostname :one
SELECT *
FROM domains
WHERE user_id = $1 AND hostname = $2;

-- name: GetVerifiedDomainByID :one
SELECT *
FROM domains
WHERE id = $1 AND user_id = $2 AND status = 'verified';

-- name: GetPrimaryDomain :one
SELECT *
FROM domains
WHERE user_id = $1 AND is_primary = true;

-- name: MarkDomainVerified :one
UPDATE domains
SET status = 'verified',
    verified_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: SetPrimaryDomain :exec
UPDATE domains
SET is_primary = (id = $2)
WHERE user_id = $1;

-- name: DeleteDomain :exec
DELETE FROM domains
WHERE id = $1 AND user_id = $2;

