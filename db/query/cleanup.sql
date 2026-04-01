-- name: CleanupExpiredLinks :exec
DELETE FROM links l
USING users u
WHERE l.user_id = u.id
  AND u.keep_expired_links = false
  AND l.deleted_at IS NULL
  AND l.expires_at IS NOT NULL
  AND l.expires_at <= now();

