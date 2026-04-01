ALTER TABLE users
  DROP COLUMN IF EXISTS timezone,
  DROP COLUMN IF EXISTS keep_expired_links,
  DROP COLUMN IF EXISTS redirect_delay_seconds;

DROP TABLE IF EXISTS api_key_domains;

ALTER TABLE domains
  DROP COLUMN IF EXISTS default_tags;

DROP TABLE IF EXISTS user_tags;

