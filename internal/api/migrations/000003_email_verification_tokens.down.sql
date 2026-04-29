BEGIN;

DROP INDEX IF EXISTS email_verification_tokens_token_hash_idx;

DROP TABLE IF EXISTS email_verification_tokens;

COMMIT;
