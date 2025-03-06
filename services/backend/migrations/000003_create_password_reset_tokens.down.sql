ALTER TABLE password_reset_tokens ALTER COLUMN id DROP DEFAULT;
DROP TABLE IF EXISTS password_reset_tokens;