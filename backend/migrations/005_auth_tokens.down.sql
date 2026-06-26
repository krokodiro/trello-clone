DROP TABLE IF EXISTS auth_tokens;
DROP TYPE IF EXISTS auth_token_type;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
