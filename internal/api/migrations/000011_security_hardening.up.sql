BEGIN;

-- Add 'blocked' to submission status enum (used when AI validation rejects code)
ALTER TYPE SUBMISSION_STATUS ADD VALUE IF NOT EXISTS 'blocked';

-- Add 'block' to security severity enum (matches AI validator severity values)
ALTER TYPE SECURITY_SEVERITY ADD VALUE IF NOT EXISTS 'block';

-- Fix security_events to support blocked submissions (which have no run_id).
-- Previously run_id was NOT NULL, but blocked submissions are rejected before
-- a run is ever created.
ALTER TABLE security_events
  ADD COLUMN submission_id UUID REFERENCES submissions(id) ON DELETE CASCADE,
  ALTER COLUMN run_id DROP NOT NULL;

ALTER TABLE security_events
  ADD CONSTRAINT security_events_target_chk
    CHECK (run_id IS NOT NULL OR submission_id IS NOT NULL);

CREATE INDEX security_events_submission_idx
  ON security_events (submission_id)
  WHERE submission_id IS NOT NULL;

COMMIT;
