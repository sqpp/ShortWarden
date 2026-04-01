-- Tags the user can manage/curate (optional; links still store tags as text[])
CREATE TABLE user_tags (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(user_id, name)
);

-- Default tags per domain applied to new links when tags omitted.
ALTER TABLE domains
  ADD COLUMN default_tags text[] NOT NULL DEFAULT '{}'::text[];

-- Restrict API keys to specific domains (optional).
CREATE TABLE api_key_domains (
  api_key_id uuid NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
  domain_id uuid NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
  PRIMARY KEY (api_key_id, domain_id)
);

-- User settings
ALTER TABLE users
  ADD COLUMN redirect_delay_seconds integer NOT NULL DEFAULT 0,
  ADD COLUMN keep_expired_links boolean NOT NULL DEFAULT false,
  ADD COLUMN timezone text NOT NULL DEFAULT 'UTC';

