BEGIN;

CREATE TABLE IF NOT EXISTS email_verification_tokens (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  token_hash  BYTEA NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX email_verification_tokens_token_hash_idx
  ON email_verification_tokens (token_hash);

COMMIT;
