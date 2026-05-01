-- =============================================================================
-- Control Plane DB — initial schema
--
-- This database is owned exclusively by the Execution Control Plane service.
-- The main API database is *not* touched by this service; communication
-- between the two services happens only over HTTP.
--
-- Tables
-- ------
--   workers          — live worker processes that poll for jobs
--   jobs             — execution jobs (one per user submission)
--   job_results      — overall verdict once a job finishes
--   job_tc_results   — per-testcase verdict/timing rows
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- Enums
-- ---------------------------------------------------------------------------

CREATE TYPE JOB_STATE AS ENUM (
  'queued',        -- just enqueued, no worker yet
  'assigned',      -- worker polled and holds a lease, not yet executing
  'running',       -- worker actively executing
  'completed',     -- worker reported a final result
  'failed',        -- terminal failure (retries exhausted or non-retryable error)
  'retry_pending'  -- lease expired / transient error; will be requeued shortly
);

CREATE TYPE WORKER_HEALTH AS ENUM (
  'healthy',   -- accepting new work
  'draining',  -- finishing current work, no new assignments
  'offline'    -- missed heartbeat deadline
);

-- ---------------------------------------------------------------------------
-- workers
--
-- One row per worker process.  Workers are responsible for calling the
-- heartbeat endpoint regularly; the control plane marks them offline when
-- their last_heartbeat becomes too old.
-- ---------------------------------------------------------------------------

CREATE TABLE workers (
  -- Stable identifier chosen by the worker (e.g. hostname + pid, or a UUID
  -- generated at startup).  Restarting a worker with the same id resumes its
  -- registration rather than creating a duplicate row.
  id             TEXT         PRIMARY KEY,

  -- Set of language keys this worker supports (e.g. '{"python","go"}').
  languages      TEXT[]       NOT NULL DEFAULT '{}',

  -- Maximum number of concurrent jobs this worker can handle.
  capacity       INT          NOT NULL DEFAULT 1 CHECK (capacity > 0),

  -- Current number of active (assigned/running) jobs on this worker.
  -- Maintained by the control plane; used for load-based scheduling.
  active_jobs    INT          NOT NULL DEFAULT 0 CHECK (active_jobs >= 0),

  health_status  WORKER_HEALTH NOT NULL DEFAULT 'healthy',

  -- Updated on every heartbeat call.
  last_heartbeat TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  registered_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX workers_health_idx    ON workers (health_status);
CREATE INDEX workers_heartbeat_idx ON workers (last_heartbeat);

-- ---------------------------------------------------------------------------
-- jobs
--
-- One row per execution request.  The control plane creates the row when the
-- API calls POST /jobs, and updates it as the lifecycle progresses.
--
-- Source code is stored as either:
--   • source_text  — inline (small submissions)
--   • source_ref   — object-storage key (large submissions)
-- Exactly one of the two must be set (enforced by CHECK constraint).
-- ---------------------------------------------------------------------------

CREATE TABLE jobs (
  id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Opaque reference back to the originating submission in the API database.
  -- The control plane does not join on this; it is returned in status
  -- responses so callers can correlate.
  submission_id  UUID         NOT NULL,

  -- Language key (e.g. "python", "go", "cpp").  Stored denormalised so the
  -- control plane is fully self-contained.
  language       TEXT         NOT NULL,

  source_text    TEXT,
  source_ref     TEXT,
  source_sha256  TEXT,

  time_limit_ms  INT          NOT NULL CHECK (time_limit_ms  > 0),
  memory_limit_mb INT         NOT NULL CHECK (memory_limit_mb > 0),

  state          JOB_STATE    NOT NULL DEFAULT 'queued',

  -- Set when a worker claims the job.
  worker_id      TEXT         REFERENCES workers(id) ON DELETE SET NULL,

  -- Absolute timestamp at which the current lease expires.
  -- NULL when the job is not assigned/running.
  lease_expires_at TIMESTAMPTZ,

  -- Retry bookkeeping.
  retry_count    INT          NOT NULL DEFAULT 0,
  max_retries    INT          NOT NULL DEFAULT 3,

  -- Human-readable reason stored on terminal failures.
  failure_reason TEXT,

  queued_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  assigned_at    TIMESTAMPTZ,
  started_at     TIMESTAMPTZ,
  finished_at    TIMESTAMPTZ,

  CONSTRAINT jobs_source_oneof_chk
    CHECK (
      (source_text IS NOT NULL AND source_ref IS NULL) OR
      (source_text IS NULL     AND source_ref IS NOT NULL)
    )
);

-- Fast queue scan: fetch next queued job for a given language.
CREATE INDEX jobs_state_queued_idx
  ON jobs (state, queued_at)
  WHERE state = 'queued';

-- Lease expiry scan: find assigned/running jobs whose lease has lapsed.
CREATE INDEX jobs_lease_expires_idx
  ON jobs (lease_expires_at)
  WHERE state IN ('assigned', 'running') AND lease_expires_at IS NOT NULL;

-- Submission lookup: API asks "what is the status of submission X?".
CREATE INDEX jobs_submission_idx ON jobs (submission_id);

-- ---------------------------------------------------------------------------
-- job_results
--
-- Written once when a job reaches the 'completed' state.
-- compiler_output is stored inline for simplicity; for large outputs a
-- reference to object storage can be stored instead.
-- ---------------------------------------------------------------------------

CREATE TABLE job_results (
  job_id              UUID         PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,

  overall_verdict     TEXT         NOT NULL,  -- e.g. "Accepted", "Wrong Answer"
  total_time_ms       INT,                    -- sum across all testcases
  max_memory_kb       INT,                    -- peak across all testcases
  wall_time_ms        INT,                    -- total wall-clock time
  compiler_output     TEXT,                   -- stdout/stderr from compilation step

  created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX job_results_verdict_idx ON job_results (overall_verdict);

-- ---------------------------------------------------------------------------
-- job_tc_results
--
-- One row per (job, testcase).  testcase_id is the external_id string from
-- the problem's test corpus — the control plane treats it as opaque.
-- ---------------------------------------------------------------------------

CREATE TABLE job_tc_results (
  job_id       UUID   NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  testcase_id  TEXT   NOT NULL,   -- matches testcases.external_id in main DB

  verdict      TEXT   NOT NULL,
  time_ms      INT,
  memory_kb    INT,

  -- Object-storage keys for captured output (may be NULL for hidden tests).
  stdout_ref   TEXT,
  stderr_ref   TEXT,

  PRIMARY KEY (job_id, testcase_id)
);

CREATE INDEX job_tc_results_job_idx ON job_tc_results (job_id);

COMMIT;
