ALTER TABLE users
  ADD COLUMN redirect_mode text NOT NULL DEFAULT 'auto',
  ADD COLUMN redirect_show_screenshot boolean NOT NULL DEFAULT false,
  ADD COLUMN redirect_custom_buttons jsonb NOT NULL DEFAULT '[]'::jsonb;

