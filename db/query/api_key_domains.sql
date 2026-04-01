-- name: ReplaceAPIKeyDomains :exec
DELETE FROM api_key_domains
WHERE api_key_id = $1;

-- name: AddAPIKeyDomain :exec
INSERT INTO api_key_domains (api_key_id, domain_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: CountAPIKeyDomains :one
SELECT count(*)::bigint
FROM api_key_domains
WHERE api_key_id = $1;

-- name: ListAPIKeyDomains :many
SELECT domain_id
FROM api_key_domains
WHERE api_key_id = $1;

