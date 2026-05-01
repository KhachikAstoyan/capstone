BEGIN;

ALTER TABLE job_tc_results
    DROP COLUMN actual_output;

COMMIT;
