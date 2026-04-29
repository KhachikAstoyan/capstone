BEGIN;

CREATE TYPE SUBMISSION_STATUS AS ENUM (
    'pending',
    'queued',
    'running',
    'accepted',
    'wrong_answer',
    'time_limit_exceeded',
    'memory_limit_exceeded',
    'runtime_error',
    'compilation_error',
    'internal_error'
);

ALTER TABLE submissions
    ADD COLUMN cp_job_id UUID,
    ADD COLUMN status    SUBMISSION_STATUS NOT NULL DEFAULT 'pending';

CREATE INDEX submissions_cp_job_id_idx ON submissions (cp_job_id) WHERE cp_job_id IS NOT NULL;
CREATE INDEX submissions_status_idx    ON submissions (status);

CREATE TABLE problem_test_cases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    problem_id  UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    input       TEXT NOT NULL,
    expected    TEXT NOT NULL,
    order_index INTEGER NOT NULL CHECK (order_index >= 0),
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (problem_id, external_id)
);

CREATE INDEX problem_test_cases_problem_idx ON problem_test_cases (problem_id);

CREATE TABLE submission_results (
    submission_id    UUID PRIMARY KEY REFERENCES submissions(id) ON DELETE CASCADE,
    overall_verdict  TEXT NOT NULL,
    total_time_ms    INTEGER,
    max_memory_kb    INTEGER,
    wall_time_ms     INTEGER,
    compiler_output  TEXT,
    testcase_results JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
