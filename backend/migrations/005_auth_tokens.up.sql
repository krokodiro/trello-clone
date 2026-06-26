ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

CREATE TYPE auth_token_type AS ENUM ('email_verification', 'password_reset');

CREATE TABLE auth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    type auth_token_type NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_tokens_user_type ON auth_tokens (user_id, type);

-- Existing accounts are treated as already verified.
UPDATE users SET email_verified_at = NOW() WHERE email_verified_at IS NULL;
