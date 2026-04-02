-- name: GetUserStats :one
SELECT
  (SELECT count(*)::bigint FROM links l WHERE l.user_id = $1 AND l.deleted_at IS NULL) AS links_total,
  (SELECT count(*)::bigint
     FROM click_events ce
     JOIN links l ON l.id = ce.link_id
    WHERE l.user_id = $1 AND l.deleted_at IS NULL) AS clicks_total,
  (SELECT count(*)::bigint
     FROM click_events ce
     JOIN links l ON l.id = ce.link_id
    WHERE l.user_id = $1 AND l.deleted_at IS NULL AND ce.clicked_at >= now() - interval '24 hours') AS clicks_24h,
  (SELECT count(*)::bigint
     FROM click_events ce
     JOIN links l ON l.id = ce.link_id
    WHERE l.user_id = $1 AND l.deleted_at IS NULL AND ce.clicked_at >= now() - interval '7 days') AS clicks_7d;

