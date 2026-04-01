CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  disabled_at timestamptz NULL
);

CREATE TABLE api_keys (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  key_hash text NOT NULL,
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz NULL,
  revoked_at timestamptz NULL,
  UNIQUE(user_id, name)
);

CREATE TABLE domains (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  hostname text NOT NULL,
  is_primary boolean NOT NULL DEFAULT false,
  status text NOT NULL DEFAULT 'pending',
  dns_token text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  verified_at timestamptz NULL,
  CONSTRAINT domains_status_chk CHECK (status IN ('pending', 'verified')),
  UNIQUE(user_id, hostname)
);

-- Enforce at most 1 primary domain per user.
CREATE UNIQUE INDEX domains_one_primary_per_user
  ON domains(user_id)
  WHERE is_primary;

CREATE TABLE links (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  domain_id uuid NULL REFERENCES domains(id) ON DELETE SET NULL,
  alias text NOT NULL,
  target_url text NOT NULL,
  title text NULL,
  tags text[] NOT NULL DEFAULT '{}'::text[],
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NULL,
  deleted_at timestamptz NULL,
  UNIQUE(alias)
);

CREATE INDEX links_user_created_at_idx ON links(user_id, created_at DESC);
CREATE INDEX links_domain_id_idx ON links(domain_id);

CREATE TABLE click_events (
  id bigserial PRIMARY KEY,
  link_id uuid NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  clicked_at timestamptz NOT NULL DEFAULT now(),
  referrer text NULL,
  user_agent text NULL,
  ip inet NULL,
  country text NULL,
  device text NULL
);

CREATE INDEX click_events_link_clicked_at_idx ON click_events(link_id, clicked_at);

