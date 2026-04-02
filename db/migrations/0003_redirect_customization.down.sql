ALTER TABLE users
  DROP COLUMN IF EXISTS redirect_custom_buttons,
  DROP COLUMN IF EXISTS redirect_show_screenshot,
  DROP COLUMN IF EXISTS redirect_mode;

