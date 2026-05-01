BEGIN;

CREATE TABLE ai_code_validations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  submission_id   UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  problem_id      UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,

  code            TEXT NOT NULL,
  language_id     UUID NOT NULL REFERENCES languages(id) ON DELETE RESTRICT,

  is_allowed      BOOLEAN NOT NULL,
  severity        TEXT,
  reason          TEXT,

  validation_metadata JSONB,

  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ai_code_validations_submission_idx
  ON ai_code_validations (submission_id);

CREATE INDEX ai_code_validations_user_created_idx
  ON ai_code_validations (user_id, created_at DESC);

CREATE INDEX ai_code_validations_is_allowed_idx
  ON ai_code_validations (is_allowed);

CREATE TABLE ai_validation_logs (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  validation_id     UUID NOT NULL REFERENCES ai_code_validations(id) ON DELETE CASCADE,

  request_body      JSONB,
  response_body     JSONB,
  error_message     TEXT,

  tokens_used       INTEGER,
  response_time_ms  INTEGER,

  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ai_validation_logs_validation_idx
  ON ai_validation_logs (validation_id);

COMMIT;
