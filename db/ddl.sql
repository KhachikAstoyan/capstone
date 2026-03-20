-- =========================================================
-- Extensions
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

-- =========================================================
-- ENUMS
-- =========================================================
CREATE TYPE USER_STATUS AS ENUM ('ACTIVE', 'BANNED');
CREATE TYPE PROBLEM_VISIBILITY AS ENUM ('draft', 'published', 'archived');
CREATE TYPE TEST_GROUP_VISIBILITY AS ENUM ('public', 'hidden');
CREATE TYPE RUN_STATE AS ENUM ('queued', 'running', 'completed', 'failed');
CREATE TYPE SECURITY_SEVERITY AS ENUM ('info', 'warn', 'high');

-- =========================================================
-- Users
-- =========================================================
CREATE TABLE IF NOT EXISTS users (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  handle          CITEXT NOT NULL UNIQUE,
  email           CITEXT UNIQUE,
  email_verified  BOOLEAN NOT NULL DEFAULT FALSE,

  display_name    TEXT,
  avatar_url      TEXT,

  status          USER_STATUS NOT NULL DEFAULT 'ACTIVE',

  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================
-- Roles / Permissions (RBAC)
-- =========================================================
CREATE TABLE IF NOT EXISTS roles (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key         TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_roles (
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  granted_by  UUID REFERENCES users(id) ON DELETE SET NULL,
  granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS user_roles_role_idx ON user_roles (role_id);

CREATE TABLE IF NOT EXISTS role_permissions (
  role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS role_permissions_permission_idx
  ON role_permissions (permission_id);

-- =========================================================
-- Auth Identities
-- =========================================================
CREATE TABLE IF NOT EXISTS auth_identities (
  id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  provider                   TEXT NOT NULL,
  provider_subject           TEXT NOT NULL,

  password_hash              TEXT,
  password_algo              TEXT,

  email_at_provider          CITEXT,
  email_verified_at_provider BOOLEAN,

  created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_login_at              TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS auth_identities_provider_subject_uq
  ON auth_identities (provider, provider_subject);

CREATE INDEX IF NOT EXISTS auth_identities_user_id_idx
  ON auth_identities (user_id);

-- =========================================================
-- Refresh Tokens
-- =========================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  auth_identity_id UUID REFERENCES auth_identities(id) ON DELETE SET NULL,

  token_hash       BYTEA NOT NULL UNIQUE,

  issued_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at       TIMESTAMPTZ NOT NULL,
  revoked_at       TIMESTAMPTZ,

  replaced_by      UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_expires_idx 
  ON refresh_tokens (user_id, expires_at) 
  WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS refresh_tokens_expires_idx 
  ON refresh_tokens (expires_at) 
  WHERE revoked_at IS NULL;

-- =========================================================
-- Languages
-- =========================================================
CREATE TABLE IF NOT EXISTS languages (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  key          TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,

  is_enabled   BOOLEAN NOT NULL DEFAULT TRUE,

  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS languages_enabled_idx
  ON languages (is_enabled);

-- =========================================================
-- Problems
-- =========================================================
CREATE TABLE IF NOT EXISTS problems (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  slug               TEXT NOT NULL UNIQUE,
  title              TEXT NOT NULL,
  statement_markdown TEXT NOT NULL,

  time_limit_ms      INTEGER NOT NULL CHECK (time_limit_ms > 0),
  memory_limit_mb    INTEGER NOT NULL CHECK (memory_limit_mb > 0),

  tests_ref          TEXT NOT NULL,
  tests_hash         TEXT,

  visibility         PROBLEM_VISIBILITY NOT NULL DEFAULT 'draft',

  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS problems_visibility_created_idx 
  ON problems (visibility, created_at DESC);

-- =========================================================
-- Problem Languages
-- =========================================================
CREATE TABLE IF NOT EXISTS problem_languages (
  problem_id  UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  language_id UUID NOT NULL REFERENCES languages(id) ON DELETE RESTRICT,
  PRIMARY KEY (problem_id, language_id)
);

-- =========================================================
-- Tags
-- =========================================================
CREATE TABLE IF NOT EXISTS tags (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS problem_tags (
  problem_id UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  tag_id     UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (problem_id, tag_id)
);

CREATE INDEX IF NOT EXISTS problem_tags_tag_idx
  ON problem_tags (tag_id);

-- =========================================================
-- Test Groups
-- =========================================================
CREATE TABLE IF NOT EXISTS test_groups (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  problem_id   UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,

  name         TEXT NOT NULL,
  visibility   TEST_GROUP_VISIBILITY NOT NULL,
  order_index  INTEGER NOT NULL CHECK (order_index >= 0),

  points_weight NUMERIC,
  is_active    BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS test_groups_problem_idx
  ON test_groups (problem_id);

CREATE UNIQUE INDEX IF NOT EXISTS test_groups_problem_order_uq
  ON test_groups (problem_id, order_index);

-- =========================================================
-- Testcases
-- =========================================================
CREATE TABLE IF NOT EXISTS testcases (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  problem_id  UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  group_id    UUID NOT NULL REFERENCES test_groups(id) ON DELETE CASCADE,

  order_index INTEGER NOT NULL CHECK (order_index >= 0),
  external_id TEXT NOT NULL,

  points      NUMERIC,
  is_active   BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS testcases_problem_idx ON testcases (problem_id);
CREATE INDEX IF NOT EXISTS testcases_group_idx   ON testcases (group_id);

CREATE UNIQUE INDEX IF NOT EXISTS testcases_problem_external_uq
  ON testcases (problem_id, external_id);

CREATE UNIQUE INDEX IF NOT EXISTS testcases_problem_order_uq
  ON testcases (problem_id, order_index);

-- =========================================================
-- Submissions
-- =========================================================
CREATE TABLE IF NOT EXISTS submissions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  problem_id    UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  language_id   UUID NOT NULL REFERENCES languages(id) ON DELETE RESTRICT,

  source_ref    TEXT,
  source_text   TEXT,
  source_sha256 TEXT,

  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT submissions_source_oneof_chk
    CHECK (
      (source_ref IS NOT NULL AND source_text IS NULL) OR
      (source_ref IS NULL AND source_text IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS submissions_user_created_idx
  ON submissions (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS submissions_problem_created_idx
  ON submissions (problem_id, created_at DESC);

-- =========================================================
-- Runs
-- =========================================================
CREATE TABLE IF NOT EXISTS runs (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id  UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,

  state          RUN_STATE NOT NULL DEFAULT 'queued',

  queued_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at     TIMESTAMPTZ,
  finished_at    TIMESTAMPTZ,

  failure_reason TEXT,
  result_digest  TEXT,
  worker_id      TEXT
);

CREATE INDEX IF NOT EXISTS runs_submission_idx ON runs (submission_id);
CREATE INDEX IF NOT EXISTS runs_state_queued_idx ON runs (state, queued_at) 
  WHERE state IN ('queued', 'running');

-- =========================================================
-- Execution Summaries
-- =========================================================
CREATE TABLE IF NOT EXISTS execution_summaries (
  run_id             UUID PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,

  overall_verdict    TEXT NOT NULL,
  total_time_ms      INTEGER,
  max_memory_kb      INTEGER,
  wall_time_ms       INTEGER,

  compiler_output_ref TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS execution_summaries_verdict_idx
  ON execution_summaries (overall_verdict);

-- =========================================================
-- Testcase Results
-- =========================================================
CREATE TABLE IF NOT EXISTS testcase_results (
  run_id      UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  testcase_id UUID NOT NULL REFERENCES testcases(id) ON DELETE RESTRICT,

  verdict     TEXT NOT NULL,
  time_ms     INTEGER,
  memory_kb   INTEGER,

  stdout_ref  TEXT,
  stderr_ref  TEXT,

  PRIMARY KEY (run_id, testcase_id)
);

CREATE INDEX IF NOT EXISTS testcase_results_run_idx
  ON testcase_results (run_id);

CREATE INDEX IF NOT EXISTS testcase_results_testcase_idx
  ON testcase_results (testcase_id);

CREATE INDEX IF NOT EXISTS testcase_results_verdict_idx
  ON testcase_results (verdict);

-- =========================================================
-- Security Events
-- =========================================================
CREATE TABLE IF NOT EXISTS security_events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  run_id      UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,

  category    TEXT NOT NULL,
  severity    SECURITY_SEVERITY NOT NULL,

  detail_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS security_events_run_idx
  ON security_events (run_id);

CREATE INDEX IF NOT EXISTS security_events_severity_created_idx
  ON security_events (severity, created_at DESC) 
  WHERE severity IN ('warn', 'high');

