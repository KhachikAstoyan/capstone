BEGIN;

ALTER TABLE job_tc_results
    ADD COLUMN actual_output TEXT;

COMMIT;
