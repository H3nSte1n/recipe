ALTER TABLE email_verification_tokens ALTER COLUMN id DROP DEFAULT;
DROP TABLE IF EXISTS email_verification_tokens;

ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
