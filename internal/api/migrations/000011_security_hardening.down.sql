BEGIN;

DROP INDEX IF EXISTS security_events_submission_idx;

ALTER TABLE security_events
  DROP CONSTRAINT IF EXISTS security_events_target_chk,
  DROP COLUMN IF EXISTS submission_id,
  ALTER COLUMN run_id SET NOT NULL;

-- NOTE: PostgreSQL does not support removing enum values.
-- 'blocked' and 'block' will remain in SUBMISSION_STATUS and SECURITY_SEVERITY enums.

COMMIT;
